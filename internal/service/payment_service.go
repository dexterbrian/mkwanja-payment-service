package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"mkwanja-payment-svc/internal/crypto"
	"mkwanja-payment-svc/internal/daraja"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/domain"
	"mkwanja-payment-svc/internal/repository"
)

// DarajaClient is the subset of daraja.Client used by PaymentService and ReconciliationService.
type DarajaClient interface {
	InitiateSTKPush(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error)
	InitiateB2C(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error)
	InitiateB2B(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error)
	QueryTransactionStatus(ctx context.Context, req daraja.TransactionStatusRequest) (*daraja.TransactionStatusResponse, error)
}

// InitiateSTKPushRequest holds input for an STK push initiation.
type InitiateSTKPushRequest struct {
	ClientID       string `json:"client_id"`
	ConsumerID     string `json:"consumer_id"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	PhoneNumber    string `json:"phone_number"`
	Reference      string `json:"reference"`
	Description    string `json:"description"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Validate checks that all required fields are present.
func (r *InitiateSTKPushRequest) Validate() error {
	if r.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if r.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	// M-PESA amounts are whole shillings; a remainder would be silently
	// truncated by the cents→KES conversion and undercharge the customer.
	if r.AmountCents%100 != 0 {
		return fmt.Errorf("amount_cents must be a whole number of shillings (multiple of 100)")
	}
	if r.PhoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}
	if r.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	return nil
}

// InitiateSTKPushResult is the response from an STK push initiation.
type InitiateSTKPushResult struct {
	PaymentID            string `json:"payment_id"`
	CheckoutRequestID    string `json:"checkout_request_id"`
	Status               string `json:"status"`
	IdempotencyReplayed  bool   `json:"-"`
}

// InitiateB2CRequest holds input for a B2C payment.
type InitiateB2CRequest struct {
	ClientID       string `json:"client_id"`
	ConsumerID     string `json:"consumer_id"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	PhoneNumber    string `json:"phone_number"`
	Reference      string `json:"reference"`
	Description    string `json:"description"`
	IdempotencyKey string `json:"idempotency_key"`
	CommandID      string `json:"command_id"`
	Remarks        string `json:"remarks"`
	Occasion       string `json:"occasion"`
}

// Validate checks that all required fields are present.
func (r *InitiateB2CRequest) Validate() error {
	if r.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if r.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	// M-PESA amounts are whole shillings; a remainder would be silently
	// truncated by the cents→KES conversion and undercharge the customer.
	if r.AmountCents%100 != 0 {
		return fmt.Errorf("amount_cents must be a whole number of shillings (multiple of 100)")
	}
	if r.PhoneNumber == "" {
		return fmt.Errorf("phone_number is required")
	}
	if r.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	return nil
}

// InitiateB2CResult is the response from a B2C payment initiation.
type InitiateB2CResult struct {
	PaymentID           string `json:"payment_id"`
	ConversationID      string `json:"conversation_id"`
	OriginatorConvID    string `json:"originator_conversation_id"`
	Status              string `json:"status"`
	IdempotencyReplayed bool   `json:"-"`
}

// InitiateB2BRequest holds input for a B2B payment.
type InitiateB2BRequest struct {
	ClientID            string `json:"client_id"`
	ConsumerID          string `json:"consumer_id"`
	AmountCents         int64  `json:"amount_cents"`
	Currency            string `json:"currency"`
	ReceiverShortcode   string `json:"receiver_shortcode"`
	Reference           string `json:"reference"`
	Description         string `json:"description"`
	IdempotencyKey      string `json:"idempotency_key"`
	CommandID           string `json:"command_id"`
	SenderIDType        string `json:"sender_identifier_type"`
	ReceiverIDType      string `json:"receiver_identifier_type"`
	Remarks             string `json:"remarks"`
}

// Validate checks that all required fields are present.
func (r *InitiateB2BRequest) Validate() error {
	if r.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if r.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	// M-PESA amounts are whole shillings; a remainder would be silently
	// truncated by the cents→KES conversion and undercharge the customer.
	if r.AmountCents%100 != 0 {
		return fmt.Errorf("amount_cents must be a whole number of shillings (multiple of 100)")
	}
	if r.ReceiverShortcode == "" {
		return fmt.Errorf("receiver_shortcode is required")
	}
	if r.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	return nil
}

// InitiateB2BResult is the response from a B2B payment initiation.
type InitiateB2BResult struct {
	PaymentID           string `json:"payment_id"`
	ConversationID      string `json:"conversation_id"`
	OriginatorConvID    string `json:"originator_conversation_id"`
	Status              string `json:"status"`
	IdempotencyReplayed bool   `json:"-"`
}

// PaymentService handles payment operations.
type PaymentService struct {
	paymentRepo repository.PaymentRepo
	clientRepo  repository.ClientRepo
	journalRepo repository.JournalRepo
	encryptKey  []byte
	callbackURL string
	rdb         *redis.Client
	logger      *slog.Logger
	buildClient func(consumerKey, consumerSecret, shortcode, passkey string) DarajaClient
}

// NewPaymentService creates a PaymentService.
func NewPaymentService(paymentRepo repository.PaymentRepo, clientRepo repository.ClientRepo, journalRepo repository.JournalRepo, encryptKey []byte, baseURL, callbackURL string, tokenCache daraja.TokenCache, rdb *redis.Client, logger *slog.Logger) *PaymentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaymentService{
		paymentRepo: paymentRepo,
		clientRepo:  clientRepo,
		journalRepo: journalRepo,
		encryptKey:  encryptKey,
		callbackURL: callbackURL,
		rdb:         rdb,
		logger:      logger,
		buildClient: func(consumerKey, consumerSecret, shortcode, passkey string) DarajaClient {
			return daraja.NewClient(baseURL, consumerKey, consumerSecret, shortcode, passkey, tokenCache)
		},
	}
}

// InitiateSTKPush initiates an STK push payment.
func (s *PaymentService) InitiateSTKPush(ctx context.Context, req InitiateSTKPushRequest) (*InitiateSTKPushResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	phone, err := domain.NormalisePhone(req.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("phone validation: %w", err)
	}

	existing, err := s.paymentRepo.GetPaymentByIdempotencyKey(ctx, req.ClientID, req.IdempotencyKey)
	if err == nil {
		s.logger.Info("idempotency hit", "payment_id", existing.ID, "idempotency_key", req.IdempotencyKey)
		return &InitiateSTKPushResult{
			PaymentID:           existing.ID,
			Status:              string(existing.Status),
			IdempotencyReplayed: true,
		}, nil
	}

	currency := req.Currency
	if currency == "" {
		currency = "KES"
	}

	desc := sql.NullString{String: req.Description, Valid: req.Description != ""}

	payment, err := s.paymentRepo.CreatePayment(ctx, db.CreatePaymentParams{
		ClientID:       req.ClientID,
		IdempotencyKey: req.IdempotencyKey,
		Provider:       db.PaymentProviderMpesa,
		PaymentType:    db.PaymentTypeStkPush,
		Direction:      db.PaymentDirectionInbound,
		AmountCents:    req.AmountCents,
		Currency:       currency,
		PhoneNumber:    sql.NullString{String: phone, Valid: true},
		Reference:      req.Reference,
		Description:    desc,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	creds, err := s.clientRepo.GetActiveCredentials(ctx, req.ClientID)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("get active credentials: %w", err)
	}

	consumerKey, err := crypto.Decrypt(s.encryptKey, creds.ConsumerKeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt consumer key: %w", err)
	}

	consumerSecret, err := crypto.Decrypt(s.encryptKey, creds.ConsumerSecretEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt consumer secret: %w", err)
	}

	passkey, err := crypto.Decrypt(s.encryptKey, creds.PasskeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt passkey: %w", err)
	}

	amountKES := req.AmountCents / 100

	dc := s.buildClient(consumerKey, consumerSecret, creds.Shortcode, passkey)

	callbackURL := s.callbackURL
	if req.ConsumerID != "" {
		callbackURL = s.callbackURL + "/webhooks/mpesa/stk/" + req.ConsumerID
	}

	resp, err := dc.InitiateSTKPush(ctx, daraja.STKPushRequest{
		BusinessShortCode: creds.Shortcode,
		Amount:            fmt.Sprintf("%d", amountKES),
		PartyA:            phone,
		PartyB:            creds.Shortcode,
		PhoneNumber:       phone,
		CallBackURL:       callbackURL,
		AccountReference:  req.Reference,
		TransactionDesc:   req.Description,
	})
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("daraja stk push: %w", err)
	}

	_, err = s.paymentRepo.UpdateProviderRequestID(ctx, payment.ID,
		sql.NullString{String: resp.CheckoutRequestID, Valid: resp.CheckoutRequestID != ""},
		sql.NullString{},
	)
	if err != nil {
		s.logger.Error("failed to update provider request id", "payment_id", payment.ID, "error", err)
	}

	routingKey := "stk:" + resp.CheckoutRequestID
	routingValue := req.ConsumerID + ":" + req.ClientID + ":" + payment.ID
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, routingKey, routingValue, 30*time.Minute).Err(); err != nil {
			s.logger.Error("failed to store routing key", "key", routingKey, "error", err)
		}
	}

	s.logger.Info("stk push initiated",
		"payment_id", payment.ID,
		"checkout_request_id", resp.CheckoutRequestID,
		"phone", phone,
	)

	return &InitiateSTKPushResult{
		PaymentID:         payment.ID,
		CheckoutRequestID: resp.CheckoutRequestID,
		Status:            string(payment.Status),
	}, nil
}

// InitiateB2C initiates a B2C payment.
func (s *PaymentService) InitiateB2C(ctx context.Context, req InitiateB2CRequest) (*InitiateB2CResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	phone, err := domain.NormalisePhone(req.PhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("phone validation: %w", err)
	}

	existing, err := s.paymentRepo.GetPaymentByIdempotencyKey(ctx, req.ClientID, req.IdempotencyKey)
	if err == nil {
		s.logger.Info("idempotency hit", "payment_id", existing.ID, "idempotency_key", req.IdempotencyKey)
		return &InitiateB2CResult{
			PaymentID:           existing.ID,
			Status:              string(existing.Status),
			IdempotencyReplayed: true,
		}, nil
	}

	currency := req.Currency
	if currency == "" {
		currency = "KES"
	}

	desc := sql.NullString{String: req.Description, Valid: req.Description != ""}

	payment, err := s.paymentRepo.CreatePayment(ctx, db.CreatePaymentParams{
		ClientID:       req.ClientID,
		IdempotencyKey: req.IdempotencyKey,
		Provider:       db.PaymentProviderMpesa,
		PaymentType:    db.PaymentTypeB2c,
		Direction:      db.PaymentDirectionOutbound,
		AmountCents:    req.AmountCents,
		Currency:       currency,
		PhoneNumber:    sql.NullString{String: phone, Valid: true},
		Reference:      req.Reference,
		Description:    desc,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	creds, err := s.clientRepo.GetActiveCredentials(ctx, req.ClientID)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("get active credentials: %w", err)
	}

	consumerKey, err := crypto.Decrypt(s.encryptKey, creds.ConsumerKeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt consumer key: %w", err)
	}

	consumerSecret, err := crypto.Decrypt(s.encryptKey, creds.ConsumerSecretEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt consumer secret: %w", err)
	}

	passkey, err := crypto.Decrypt(s.encryptKey, creds.PasskeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt passkey: %w", err)
	}

	initiatorName := ""
	if creds.InitiatorName.Valid {
		initiatorName = creds.InitiatorName.String
	}

	securityCred := ""
	if creds.SecurityCredentialEncrypted.Valid && creds.SecurityCredentialEncrypted.String != "" {
		sc, err := crypto.Decrypt(s.encryptKey, creds.SecurityCredentialEncrypted.String)
		if err != nil {
			_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
			return nil, fmt.Errorf("decrypt security credential: %w", err)
		}
		securityCred = sc
	}

	dc := s.buildClient(consumerKey, consumerSecret, creds.Shortcode, passkey)

	amountKES := req.AmountCents / 100

	callbackURL := s.callbackURL + "/webhooks/mpesa/b2c/" + req.ConsumerID

	resp, err := dc.InitiateB2C(ctx, daraja.B2CRequest{
		InitiatorName:      initiatorName,
		SecurityCredential: securityCred,
		CommandID:          req.CommandID,
		Amount:             fmt.Sprintf("%d", amountKES),
		PartyA:             creds.Shortcode,
		PartyB:             phone,
		Remarks:            req.Remarks,
		QueueTimeOutURL:    callbackURL,
		ResultURL:          callbackURL,
		Occasion:           req.Occasion,
	})
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("daraja b2c: %w", err)
	}

	_, err = s.paymentRepo.UpdateProviderRequestID(ctx, payment.ID,
		sql.NullString{String: resp.ConversationID, Valid: resp.ConversationID != ""},
		sql.NullString{},
	)
	if err != nil {
		s.logger.Error("failed to update provider request id", "payment_id", payment.ID, "error", err)
	}

	s.logger.Info("b2c initiated",
		"payment_id", payment.ID,
		"conversation_id", resp.ConversationID,
		"phone", phone,
	)

	return &InitiateB2CResult{
		PaymentID:        payment.ID,
		ConversationID:   resp.ConversationID,
		OriginatorConvID: resp.OriginatorConversationID,
		Status:           string(payment.Status),
	}, nil
}

// InitiateB2B initiates a B2B payment.
func (s *PaymentService) InitiateB2B(ctx context.Context, req InitiateB2BRequest) (*InitiateB2BResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	existing, err := s.paymentRepo.GetPaymentByIdempotencyKey(ctx, req.ClientID, req.IdempotencyKey)
	if err == nil {
		s.logger.Info("idempotency hit", "payment_id", existing.ID, "idempotency_key", req.IdempotencyKey)
		return &InitiateB2BResult{
			PaymentID:           existing.ID,
			Status:              string(existing.Status),
			IdempotencyReplayed: true,
		}, nil
	}

	currency := req.Currency
	if currency == "" {
		currency = "KES"
	}

	desc := sql.NullString{String: req.Description, Valid: req.Description != ""}

	payment, err := s.paymentRepo.CreatePayment(ctx, db.CreatePaymentParams{
		ClientID:          req.ClientID,
		IdempotencyKey:    req.IdempotencyKey,
		Provider:          db.PaymentProviderMpesa,
		PaymentType:       db.PaymentTypeB2b,
		Direction:         db.PaymentDirectionOutbound,
		AmountCents:       req.AmountCents,
		Currency:          currency,
		ReceiverShortcode: sql.NullString{String: req.ReceiverShortcode, Valid: req.ReceiverShortcode != ""},
		Reference:         req.Reference,
		Description:       desc,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	creds, err := s.clientRepo.GetActiveCredentials(ctx, req.ClientID)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("get active credentials: %w", err)
	}

	consumerKey, err := crypto.Decrypt(s.encryptKey, creds.ConsumerKeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt consumer key: %w", err)
	}

	consumerSecret, err := crypto.Decrypt(s.encryptKey, creds.ConsumerSecretEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt consumer secret: %w", err)
	}

	passkey, err := crypto.Decrypt(s.encryptKey, creds.PasskeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt passkey: %w", err)
	}

	initiatorName := ""
	if creds.InitiatorName.Valid {
		initiatorName = creds.InitiatorName.String
	}

	securityCred := ""
	if creds.SecurityCredentialEncrypted.Valid && creds.SecurityCredentialEncrypted.String != "" {
		sc, err := crypto.Decrypt(s.encryptKey, creds.SecurityCredentialEncrypted.String)
		if err != nil {
			_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
			return nil, fmt.Errorf("decrypt security credential: %w", err)
		}
		securityCred = sc
	}

	dc := s.buildClient(consumerKey, consumerSecret, creds.Shortcode, passkey)

	amountKES := req.AmountCents / 100

	senderIDType := req.SenderIDType
	if senderIDType == "" {
		senderIDType = "4"
	}
	receiverIDType := req.ReceiverIDType
	if receiverIDType == "" {
		receiverIDType = "4"
	}

	commandID := req.CommandID
	if commandID == "" {
		commandID = "BusinessPayBill"
	}

	callbackURL := s.callbackURL + "/webhooks/mpesa/b2b/" + req.ConsumerID

	resp, err := dc.InitiateB2B(ctx, daraja.B2BRequest{
		Initiator:              initiatorName,
		SecurityCredential:     securityCred,
		CommandID:              commandID,
		SenderIdentifierType:   senderIDType,
		ReceiverIdentifierType: receiverIDType,
		Amount:                 fmt.Sprintf("%d", amountKES),
		PartyA:                 creds.Shortcode,
		PartyB:                 req.ReceiverShortcode,
		AccountReference:       req.Reference,
		Remarks:                req.Remarks,
		QueueTimeOutURL:        callbackURL,
		ResultURL:              callbackURL,
	})
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("daraja b2b: %w", err)
	}

	_, err = s.paymentRepo.UpdateProviderRequestID(ctx, payment.ID,
		sql.NullString{String: resp.ConversationID, Valid: resp.ConversationID != ""},
		sql.NullString{},
	)
	if err != nil {
		s.logger.Error("failed to update provider request id", "payment_id", payment.ID, "error", err)
	}

	s.logger.Info("b2b initiated",
		"payment_id", payment.ID,
		"conversation_id", resp.ConversationID,
		"receiver", req.ReceiverShortcode,
	)

	return &InitiateB2BResult{
		PaymentID:        payment.ID,
		ConversationID:   resp.ConversationID,
		OriginatorConvID: resp.OriginatorConversationID,
		Status:           string(payment.Status),
	}, nil
}

// GetPayment retrieves a payment by ID.
func (s *PaymentService) GetPayment(ctx context.Context, id string) (*db.Payment, error) {
	p, err := s.paymentRepo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return &p, nil
}

// ListPayments lists payments for a client with pagination.
func (s *PaymentService) ListPayments(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
	return s.paymentRepo.ListPaymentsByClient(ctx, clientID, limit, offset)
}

// CompletePayment updates a payment to completed, creates a payment event, and writes journal entries.
func (s *PaymentService) CompletePayment(ctx context.Context, paymentID, receipt, txID string) error {
	p, err := s.paymentRepo.CompletePayment(ctx, paymentID,
		sql.NullString{String: receipt, Valid: receipt != ""},
		sql.NullString{String: txID, Valid: txID != ""},
	)
	if err != nil {
		return fmt.Errorf("complete payment: %w", err)
	}

	_, err = s.paymentRepo.CreatePaymentEvent(ctx, db.CreatePaymentEventParams{
		PaymentID:  paymentID,
		FromStatus: db.NullPaymentStatus{PaymentStatus: db.PaymentStatusPending, Valid: true},
		ToStatus:   db.PaymentStatusCompleted,
	})
	if err != nil {
		s.logger.Error("failed to create payment complete event", "payment_id", paymentID, "error", err)
	}

	// Write journal entries based on payment direction
	journalSvc := NewJournalService(s.journalRepo, s.logger)
	switch p.Direction {
	case db.PaymentDirectionInbound:
		if err := journalSvc.WriteInboundEntries(ctx, p); err != nil {
			s.logger.Error("failed to write inbound journal entries", "payment_id", paymentID, "error", err)
		}
	case db.PaymentDirectionOutbound:
		if err := journalSvc.WriteOutboundEntries(ctx, p); err != nil {
			s.logger.Error("failed to write outbound journal entries", "payment_id", paymentID, "error", err)
		}
	default:
		s.logger.Warn("unknown payment direction, no journal entries written", "payment_id", paymentID, "direction", p.Direction)
	}

	s.logger.Info("payment completed", "payment_id", paymentID, "receipt", receipt)
	return nil
}

// FailPayment updates a payment to failed and creates a payment event.
func (s *PaymentService) FailPayment(ctx context.Context, paymentID, reason string) error {
	_, err := s.paymentRepo.FailPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("fail payment: %w", err)
	}

	_, err = s.paymentRepo.CreatePaymentEvent(ctx, db.CreatePaymentEventParams{
		PaymentID:  paymentID,
		FromStatus: db.NullPaymentStatus{PaymentStatus: db.PaymentStatusPending, Valid: true},
		ToStatus:   db.PaymentStatusFailed,
		Reason:     sql.NullString{String: reason, Valid: reason != ""},
	})
	if err != nil {
		s.logger.Error("failed to create payment fail event", "payment_id", paymentID, "error", err)
	}

	s.logger.Info("payment failed", "payment_id", paymentID, "reason", reason)
	return nil
}

// NewPaymentServiceForTest creates a PaymentService with a custom daraja client builder for testing.
func NewPaymentServiceForTest(paymentRepo repository.PaymentRepo, clientRepo repository.ClientRepo, journalRepo repository.JournalRepo, encryptKey []byte, callbackURL string, rdb *redis.Client, buildClient func(consumerKey, consumerSecret, shortcode, passkey string) DarajaClient, logger *slog.Logger) *PaymentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaymentService{
		paymentRepo: paymentRepo,
		clientRepo:  clientRepo,
		journalRepo: journalRepo,
		encryptKey:  encryptKey,
		callbackURL: callbackURL,
		rdb:         rdb,
		logger:      logger,
		buildClient: buildClient,
	}
}
