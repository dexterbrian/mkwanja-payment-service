package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"mkwanja-payment-svc/internal/crypto"
	"mkwanja-payment-svc/internal/daraja"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// dbPgRegistry is the subset of db.Registry used by ReconciliationService.
type dbPgRegistry interface {
	Get(consumerID string) (*pgxpool.Pool, error)
	Ping(ctx context.Context) map[string]error
}

// ReconciliationService resolves pending payments by querying Daraja transaction status.
type ReconciliationService struct {
	registry    dbPgRegistry
	encryptKey  []byte
	baseURL     string
	tokenCache  daraja.TokenCache
	callbackURL string
	rdb         *redis.Client
	logger      *slog.Logger
	buildClient func(consumerKey, consumerSecret, shortcode, passkey string) DarajaClient
}

// NewReconciliationService creates a ReconciliationService.
func NewReconciliationService(registry dbPgRegistry, encryptKey []byte, baseURL, callbackURL string, tokenCache daraja.TokenCache, rdb *redis.Client, logger *slog.Logger) *ReconciliationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReconciliationService{
		registry:    registry,
		encryptKey:  encryptKey,
		baseURL:     baseURL,
		tokenCache:  tokenCache,
		callbackURL: callbackURL,
		rdb:         rdb,
		logger:      logger,
		buildClient: func(consumerKey, consumerSecret, shortcode, passkey string) DarajaClient {
			return daraja.NewClient(baseURL, consumerKey, consumerSecret, shortcode, passkey, tokenCache)
		},
	}
}

// RunOnce runs one full reconciliation cycle over all consumer databases.
func (s *ReconciliationService) RunOnce(ctx context.Context) error {
	consumers := s.registry.Ping(ctx)
	if len(consumers) == 0 {
		s.logger.Warn("no consumers registered for reconciliation")
		return nil
	}

	for consumerID := range consumers {
		pool, err := s.registry.Get(consumerID)
		if err != nil {
			s.logger.Error("reconciliation: cannot get pool for consumer", "consumer_id", consumerID, "error", err)
			continue
		}
		if err := s.reconcileConsumer(ctx, consumerID, pool); err != nil {
			s.logger.Error("reconciliation: consumer cycle failed", "consumer_id", consumerID, "error", err)
		}
	}
	return nil
}

// reconcileConsumer reconciles all pending payments older than 2 minutes for one consumer.
func (s *ReconciliationService) reconcileConsumer(ctx context.Context, consumerID string, pool *pgxpool.Pool) error {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	q := db.New(stdlibDB)
	paymentRepo := repository.NewPgxPaymentRepo(q)
	clientRepo := repository.NewPgxClientRepo(q)
	journalRepo := repository.NewPgxJournalRepo(q, stdlibDB)

	cutoff := time.Now().Add(-2 * time.Minute)
	pending, err := paymentRepo.ListPendingPaymentsOlderThan(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list pending payments for %s: %w", consumerID, err)
	}

	if len(pending) == 0 {
		s.logger.Debug("reconciliation: no pending payments", "consumer_id", consumerID)
		return nil
	}

	s.logger.Info("reconciliation: pending payments found",
		"consumer_id", consumerID,
		"count", len(pending),
	)

	for _, payment := range pending {
		if err := s.reconcilePayment(ctx, payment, consumerID, paymentRepo, clientRepo, journalRepo); err != nil {
			s.logger.Error("reconciliation: payment reconciliation failed",
				"consumer_id", consumerID,
				"payment_id", payment.ID,
				"error", err,
			)
		}
	}
	return nil
}

// reconcilePayment checks the status of a single pending payment and resolves it.
func (s *ReconciliationService) reconcilePayment(
	ctx context.Context,
	payment db.Payment,
	consumerID string,
	paymentRepo repository.PaymentRepo,
	clientRepo repository.ClientRepo,
	journalRepo repository.JournalRepo,
) error {
	creds, err := clientRepo.GetActiveCredentials(ctx, payment.ClientID)
	if err != nil {
		return fmt.Errorf("get credentials for payment %s: %w", payment.ID, err)
	}

	consumerKey, err := crypto.Decrypt(s.encryptKey, creds.ConsumerKeyEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt consumer key for payment %s: %w", payment.ID, err)
	}

	consumerSecret, err := crypto.Decrypt(s.encryptKey, creds.ConsumerSecretEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt consumer secret for payment %s: %w", payment.ID, err)
	}

	passkey, err := crypto.Decrypt(s.encryptKey, creds.PasskeyEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt passkey for payment %s: %w", payment.ID, err)
	}

	initiatorName := ""
	if creds.InitiatorName.Valid {
		initiatorName = creds.InitiatorName.String
	}
	securityCred := ""
	if creds.SecurityCredentialEncrypted.Valid && creds.SecurityCredentialEncrypted.String != "" {
		sc, err := crypto.Decrypt(s.encryptKey, creds.SecurityCredentialEncrypted.String)
		if err != nil {
			return fmt.Errorf("decrypt security credential for payment %s: %w", payment.ID, err)
		}
		securityCred = sc
	}

	dc := s.buildClient(consumerKey, consumerSecret, creds.Shortcode, passkey)

	callbackURL := s.callbackURL + "/webhooks/mpesa/reconciliation/" + consumerID

	txID := ""
	if payment.ProviderRequestID.Valid {
		txID = payment.ProviderRequestID.String
	}

	txStatus, err := dc.QueryTransactionStatus(ctx, daraja.TransactionStatusRequest{
		Initiator:          initiatorName,
		SecurityCredential: securityCred,
		CommandID:          "TransactionStatusQuery",
		TransactionID:      txID,
		PartyA:             creds.Shortcode,
		IdentifierType:     "1",
		ResultURL:          callbackURL,
		QueueTimeOutURL:    callbackURL,
		Remarks:            "Reconciliation",
		Occasion:           "reconciliation",
	})
	if err != nil {
		return fmt.Errorf("query transaction status for payment %s: %w", payment.ID, err)
	}

	if txStatus.ResultCode == "0" || txStatus.TransactionStatus == "completed" {
		receipt := txStatus.ReceiptNo
		providerTxID := txStatus.ConversationID

		svc := &PaymentService{
			paymentRepo: paymentRepo,
			journalRepo: journalRepo,
			logger:      s.logger,
		}
		if err := svc.CompletePayment(ctx, payment.ID, receipt, providerTxID); err != nil {
			return fmt.Errorf("complete payment %s: %w", payment.ID, err)
		}

		s.logger.Info("reconciliation: payment completed",
			"payment_id", payment.ID,
			"consumer_id", consumerID,
			"receipt", receipt,
		)
	} else {
		svc := &PaymentService{
			paymentRepo: paymentRepo,
			logger:      s.logger,
		}
		if err := svc.FailPayment(ctx, payment.ID, "reconciliation: daraja returned result code "+txStatus.ResultCode); err != nil {
			return fmt.Errorf("fail payment %s: %w", payment.ID, err)
		}

		s.logger.Info("reconciliation: payment failed",
			"payment_id", payment.ID,
			"consumer_id", consumerID,
			"result_code", txStatus.ResultCode,
			"result_desc", txStatus.ResultDesc,
		)
	}
	return nil
}

// NewReconciliationServiceForTest creates a ReconciliationService with a custom daraja client builder for testing.
func NewReconciliationServiceForTest(
	registry dbPgRegistry,
	encryptKey []byte,
	callbackURL string,
	buildClient func(consumerKey, consumerSecret, shortcode, passkey string) DarajaClient,
	logger *slog.Logger,
) *ReconciliationService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReconciliationService{
		registry:    registry,
		encryptKey:  encryptKey,
		callbackURL: callbackURL,
		logger:      logger,
		buildClient: buildClient,
	}
}
