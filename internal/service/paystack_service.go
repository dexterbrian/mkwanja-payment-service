package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"mkwanja-payment-svc/internal/crypto"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/paystack"
	"mkwanja-payment-svc/internal/repository"
)

// PaystackAPI is the subset of paystack.Client used by PaystackService.
type PaystackAPI interface {
	InitializeTransaction(ctx context.Context, req paystack.InitializeTransactionRequest) (*paystack.InitializeTransactionData, error)
	VerifyTransaction(ctx context.Context, reference string) (*paystack.VerifyTransactionData, error)
}

// paymentCompleter is the subset of PaymentService used to settle payments.
type paymentCompleter interface {
	CompletePayment(ctx context.Context, paymentID, receipt, txID string) error
	FailPayment(ctx context.Context, paymentID, reason string) error
}

// PaystackService handles Paystack credential registration, charges, and webhooks.
type PaystackService struct {
	paymentRepo  repository.PaymentRepo
	paystackRepo repository.PaystackRepo
	completer    paymentCompleter
	encryptKey   []byte
	logger       *slog.Logger
	buildClient  func(secretKey string) PaystackAPI
}

// NewPaystackService creates a PaystackService.
func NewPaystackService(paymentRepo repository.PaymentRepo, paystackRepo repository.PaystackRepo, completer paymentCompleter, encryptKey []byte, baseURL string, logger *slog.Logger) *PaystackService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaystackService{
		paymentRepo:  paymentRepo,
		paystackRepo: paystackRepo,
		completer:    completer,
		encryptKey:   encryptKey,
		logger:       logger,
		buildClient: func(secretKey string) PaystackAPI {
			return paystack.NewClient(baseURL, secretKey)
		},
	}
}

// RegisterPaystackCredentialsRequest holds BYO Paystack keys for a client.
type RegisterPaystackCredentialsRequest struct {
	ClientID  string `json:"client_id"`
	SecretKey string `json:"secret_key"`
	PublicKey string `json:"public_key"`
}

// Validate checks required fields.
func (r *RegisterPaystackCredentialsRequest) Validate() error {
	if r.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if r.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}
	if r.PublicKey == "" {
		return fmt.Errorf("public_key is required")
	}
	return nil
}

// RegisterCredentials stores a business's own Paystack keys (secret key
// encrypted at rest), replacing any previously active set.
func (s *PaystackService) RegisterCredentials(ctx context.Context, req RegisterPaystackCredentialsRequest) (*db.ClientPaystackCredential, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	secretEncrypted, err := crypto.Encrypt(s.encryptKey, req.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt secret key: %w", err)
	}

	if err := s.paystackRepo.DeactivatePaystackCredentials(ctx, req.ClientID); err != nil {
		return nil, fmt.Errorf("deactivate old credentials: %w", err)
	}

	cred, err := s.paystackRepo.CreatePaystackCredentials(ctx, db.CreatePaystackCredentialsParams{
		ClientID:           req.ClientID,
		SecretKeyEncrypted: secretEncrypted,
		PublicKey:          req.PublicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create paystack credentials: %w", err)
	}

	s.logger.Info("paystack credentials registered", "client_id", req.ClientID)
	return &cred, nil
}

// InitiatePaystackChargeRequest holds input for a Paystack charge.
type InitiatePaystackChargeRequest struct {
	ClientID       string `json:"client_id"`
	ConsumerID     string `json:"consumer_id"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	Email          string `json:"email"`
	Reference      string `json:"reference"`
	Description    string `json:"description"`
	CallbackURL    string `json:"callback_url"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Validate checks required fields. Paystack takes amounts in currency
// subunits, so any positive integer-cents amount is representable.
func (r *InitiatePaystackChargeRequest) Validate() error {
	if r.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if r.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if r.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	return nil
}

// InitiatePaystackChargeResult is the response from a charge initiation.
type InitiatePaystackChargeResult struct {
	PaymentID           string `json:"payment_id"`
	AuthorizationURL    string `json:"authorization_url"`
	AccessCode          string `json:"access_code"`
	Status              string `json:"status"`
	IdempotencyReplayed bool   `json:"-"`
}

// InitiateCharge creates a pending payment and a Paystack hosted-checkout
// transaction. The Paystack reference is the payment ID, so webhooks route
// back without extra state.
func (s *PaystackService) InitiateCharge(ctx context.Context, req InitiatePaystackChargeRequest) (*InitiatePaystackChargeResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	existing, err := s.paymentRepo.GetPaymentByIdempotencyKey(ctx, req.ClientID, req.IdempotencyKey)
	if err == nil {
		s.logger.Info("idempotency hit", "payment_id", existing.ID, "idempotency_key", req.IdempotencyKey)
		return &InitiatePaystackChargeResult{
			PaymentID:           existing.ID,
			Status:              string(existing.Status),
			IdempotencyReplayed: true,
		}, nil
	}

	currency := req.Currency
	if currency == "" {
		currency = "KES"
	}

	payment, err := s.paymentRepo.CreatePayment(ctx, db.CreatePaymentParams{
		ClientID:       req.ClientID,
		IdempotencyKey: req.IdempotencyKey,
		Provider:       db.PaymentProviderPaystack,
		PaymentType:    db.PaymentTypePaystackCharge,
		Direction:      db.PaymentDirectionInbound,
		AmountCents:    req.AmountCents,
		Currency:       currency,
		Reference:      req.Reference,
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	creds, err := s.paystackRepo.GetActivePaystackCredentials(ctx, req.ClientID)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("get paystack credentials: %w", err)
	}

	secretKey, err := crypto.Decrypt(s.encryptKey, creds.SecretKeyEncrypted)
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("decrypt secret key: %w", err)
	}

	pc := s.buildClient(secretKey)
	data, err := pc.InitializeTransaction(ctx, paystack.InitializeTransactionRequest{
		Email:       req.Email,
		Amount:      req.AmountCents, // subunits — no conversion
		Currency:    currency,
		Reference:   payment.ID,
		CallbackURL: req.CallbackURL,
	})
	if err != nil {
		_, _ = s.paymentRepo.FailPayment(ctx, payment.ID)
		return nil, fmt.Errorf("paystack initialize: %w", err)
	}

	_, err = s.paymentRepo.UpdateProviderRequestID(ctx, payment.ID,
		sql.NullString{String: data.Reference, Valid: data.Reference != ""},
		sql.NullString{},
	)
	if err != nil {
		s.logger.Error("failed to update provider request id", "payment_id", payment.ID, "error", err)
	}

	s.logger.Info("paystack charge initiated", "payment_id", payment.ID, "access_code", data.AccessCode)

	return &InitiatePaystackChargeResult{
		PaymentID:        payment.ID,
		AuthorizationURL: data.AuthorizationURL,
		AccessCode:       data.AccessCode,
		Status:           string(payment.Status),
	}, nil
}

// HandleWebhook verifies and processes a Paystack webhook delivery.
// The signature is HMAC-SHA512 of the raw body keyed with the business's
// secret key, which is resolved via the payment referenced by the event.
func (s *PaystackService) HandleWebhook(ctx context.Context, rawBody []byte, signature string) error {
	var event paystack.WebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return fmt.Errorf("parse webhook: %w", err)
	}
	if event.Data.Reference == "" {
		return fmt.Errorf("webhook event has no reference")
	}

	// Reference is our payment ID (set at initialization).
	payment, err := s.paymentRepo.GetPaymentByID(ctx, event.Data.Reference)
	if err != nil {
		return fmt.Errorf("lookup payment %s: %w", event.Data.Reference, err)
	}

	creds, err := s.paystackRepo.GetActivePaystackCredentials(ctx, payment.ClientID)
	if err != nil {
		return fmt.Errorf("get paystack credentials: %w", err)
	}
	secretKey, err := crypto.Decrypt(s.encryptKey, creds.SecretKeyEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt secret key: %w", err)
	}

	if !paystack.VerifyWebhookSignature(secretKey, rawBody, signature) {
		return fmt.Errorf("invalid webhook signature for payment %s", payment.ID)
	}

	switch event.Event {
	case "charge.success":
		if event.Data.Amount != payment.AmountCents {
			return fmt.Errorf("amount mismatch for payment %s: webhook %d, recorded %d",
				payment.ID, event.Data.Amount, payment.AmountCents)
		}
		if err := s.completer.CompletePayment(ctx, payment.ID, event.Data.Reference, fmt.Sprintf("%d", event.Data.ID)); err != nil {
			return fmt.Errorf("complete payment %s: %w", payment.ID, err)
		}
		s.logger.Info("paystack payment completed", "payment_id", payment.ID)
	case "charge.failed":
		if err := s.completer.FailPayment(ctx, payment.ID, event.Data.Status); err != nil {
			return fmt.Errorf("fail payment %s: %w", payment.ID, err)
		}
		s.logger.Info("paystack payment failed", "payment_id", payment.ID)
	default:
		s.logger.Info("paystack webhook ignored", "event", event.Event, "payment_id", payment.ID)
	}

	return nil
}

// NewPaystackServiceForTest creates a PaystackService with a custom client builder.
func NewPaystackServiceForTest(paymentRepo repository.PaymentRepo, paystackRepo repository.PaystackRepo, completer paymentCompleter, encryptKey []byte, buildClient func(secretKey string) PaystackAPI, logger *slog.Logger) *PaystackService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaystackService{
		paymentRepo:  paymentRepo,
		paystackRepo: paystackRepo,
		completer:    completer,
		encryptKey:   encryptKey,
		logger:       logger,
		buildClient:  buildClient,
	}
}
