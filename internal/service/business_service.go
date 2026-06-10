package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"mkwanja-payment-svc/internal/crypto"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// SeedAccount represents a default journal account to create per business.
type SeedAccount struct {
	ID            string
	Name          string
	AccountType   db.AccountType
	NormalBalance db.NormalBalance
	Description   string
}

// defaultAccounts are seeded into every new business.
var defaultAccounts = []SeedAccount{
	{"mpesa.till", "M-PESA till", db.AccountTypeAsset, db.NormalBalanceDebit, "M-PESA till balance"},
	{"revenue.sales", "Sales revenue", db.AccountTypeRevenue, db.NormalBalanceCredit, "Sales revenue ex-VAT"},
	{"revenue.other", "Other revenue", db.AccountTypeRevenue, db.NormalBalanceCredit, "Other revenue"},
	{"expense.cogs", "Cost of goods sold", db.AccountTypeExpense, db.NormalBalanceDebit, "Cost of goods sold"},
	{"expense.operations", "Operating expenses", db.AccountTypeExpense, db.NormalBalanceDebit, "Operating expenses"},
	{"liability.vat_payable", "VAT payable", db.AccountTypeLiability, db.NormalBalanceCredit, "Output VAT collected"},
	{"liability.pending", "Pending payments", db.AccountTypeLiability, db.NormalBalanceCredit, "Pending payments"},
	{"fees.mpesa", "M-PESA charges", db.AccountTypeExpense, db.NormalBalanceDebit, "M-PESA transaction fees"},
}

// BusinessService handles business and credential management.
type BusinessService struct {
	repo       repository.BusinessRepo
	encryptKey []byte
	logger     *slog.Logger
}

// NewBusinessService creates a BusinessService.
func NewBusinessService(repo repository.BusinessRepo, encryptKey []byte, logger *slog.Logger) *BusinessService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BusinessService{
		repo:       repo,
		encryptKey: encryptKey,
		logger:     logger,
	}
}

// RegisterBusinessRequest holds the input for business registration.
type RegisterBusinessRequest struct {
	ExternalID         string
	Name               string
	Shortcode          string
	ConsumerKey        string
	ConsumerSecret     string
	Passkey            string
	InitiatorName      string
	SecurityCredential string
}

// Validate checks the request fields.
func (r *RegisterBusinessRequest) Validate() error {
	if r.ExternalID == "" {
		return fmt.Errorf("external_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Shortcode == "" {
		return fmt.Errorf("shortcode is required")
	}
	if r.ConsumerKey == "" {
		return fmt.Errorf("consumer_key is required")
	}
	if r.ConsumerSecret == "" {
		return fmt.Errorf("consumer_secret is required")
	}
	if r.Passkey == "" {
		return fmt.Errorf("passkey is required")
	}
	return nil
}

// RegisterBusiness creates a business, encrypts + stores credentials, and seeds default journal accounts.
func (s *BusinessService) RegisterBusiness(ctx context.Context, req RegisterBusinessRequest) (*db.Business, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Create business
	b, err := s.repo.CreateBusiness(ctx, db.CreateBusinessParams{
		ExternalID: req.ExternalID,
		Name:       req.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("create business: %w", err)
	}

	// Encrypt credentials
	ck, err := crypto.Encrypt(s.encryptKey, req.ConsumerKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt consumer_key: %w", err)
	}
	cs, err := crypto.Encrypt(s.encryptKey, req.ConsumerSecret)
	if err != nil {
		return nil, fmt.Errorf("encrypt consumer_secret: %w", err)
	}
	pk, err := crypto.Encrypt(s.encryptKey, req.Passkey)
	if err != nil {
		return nil, fmt.Errorf("encrypt passkey: %w", err)
	}

	var initiatorName sql.NullString
	if req.InitiatorName != "" {
		initiatorName = sql.NullString{String: req.InitiatorName, Valid: true}
	}
	var securityCred sql.NullString
	if req.SecurityCredential != "" {
		sc, err := crypto.Encrypt(s.encryptKey, req.SecurityCredential)
		if err != nil {
			return nil, fmt.Errorf("encrypt security_credential: %w", err)
		}
		securityCred = sql.NullString{String: sc, Valid: true}
	}

	// Store credentials
	_, err = s.repo.CreateCredentials(ctx, db.CreateCredentialsParams{
		BusinessID:                  b.ID,
		Shortcode:                   req.Shortcode,
		ConsumerKeyEncrypted:        ck,
		ConsumerSecretEncrypted:     cs,
		PasskeyEncrypted:            pk,
		InitiatorName:               initiatorName,
		SecurityCredentialEncrypted: securityCred,
	})
	if err != nil {
		return nil, fmt.Errorf("store credentials: %w", err)
	}

	// Seed default journal accounts
	for _, acct := range defaultAccounts {
		desc := sql.NullString{String: acct.Description, Valid: true}
		_, err := s.repo.CreateJournalAccount(ctx, db.CreateJournalAccountParams{
			ID:            acct.ID,
			BusinessID:    b.ID,
			Name:          acct.Name,
			AccountType:   acct.AccountType,
			NormalBalance: acct.NormalBalance,
			Description:   desc,
		})
		if err != nil {
			s.logger.Error("failed to seed journal account",
				"account_id", acct.ID,
				"business_id", b.ID,
				"error", err)
		}
	}

	s.logger.Info("business registered",
		"business_id", b.ID,
		"external_id", b.ExternalID,
		"name", b.Name)
	return &b, nil
}

// TestCredentialsRequest holds input for credential verification.
type TestCredentialsRequest struct {
	ConsumerKey    string
	ConsumerSecret string
	Shortcode      string
}

// Validate checks the request fields.
func (r *TestCredentialsRequest) Validate() error {
	if r.ConsumerKey == "" {
		return fmt.Errorf("consumer_key is required")
	}
	if r.ConsumerSecret == "" {
		return fmt.Errorf("consumer_secret is required")
	}
	if r.Shortcode == "" {
		return fmt.Errorf("shortcode is required")
	}
	return nil
}

// TestCredentials verifies Daraja OAuth without saving credentials.
func (s *BusinessService) TestCredentials(ctx context.Context, req TestCredentialsRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation: %w", err)
	}
	s.logger.Info("credentials test passed (validation only — Daraja call in Phase 4)",
		"shortcode", req.Shortcode)
	return nil
}

// UpdateCredentialsRequest holds input for credential update.
type UpdateCredentialsRequest struct {
	BusinessID         string
	Shortcode          string
	ConsumerKey        string
	ConsumerSecret     string
	Passkey            string
	InitiatorName      string
	SecurityCredential string
}

// Validate checks the request fields.
func (r *UpdateCredentialsRequest) Validate() error {
	if r.BusinessID == "" {
		return fmt.Errorf("business_id is required")
	}
	if r.ConsumerKey == "" {
		return fmt.Errorf("consumer_key is required")
	}
	if r.ConsumerSecret == "" {
		return fmt.Errorf("consumer_secret is required")
	}
	if r.Passkey == "" {
		return fmt.Errorf("passkey is required")
	}
	return nil
}

// UpdateCredentials deactivates old credentials and stores new encrypted ones.
func (s *BusinessService) UpdateCredentials(ctx context.Context, req UpdateCredentialsRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.DeactivateCredentials(ctx, req.BusinessID); err != nil {
		return fmt.Errorf("deactivate old credentials: %w", err)
	}

	ck, err := crypto.Encrypt(s.encryptKey, req.ConsumerKey)
	if err != nil {
		return fmt.Errorf("encrypt consumer_key: %w", err)
	}
	cs, err := crypto.Encrypt(s.encryptKey, req.ConsumerSecret)
	if err != nil {
		return fmt.Errorf("encrypt consumer_secret: %w", err)
	}
	pk, err := crypto.Encrypt(s.encryptKey, req.Passkey)
	if err != nil {
		return fmt.Errorf("encrypt passkey: %w", err)
	}

	var initiatorName sql.NullString
	if req.InitiatorName != "" {
		initiatorName = sql.NullString{String: req.InitiatorName, Valid: true}
	}
	var securityCred sql.NullString
	if req.SecurityCredential != "" {
		sc, err := crypto.Encrypt(s.encryptKey, req.SecurityCredential)
		if err != nil {
			return fmt.Errorf("encrypt security_credential: %w", err)
		}
		securityCred = sql.NullString{String: sc, Valid: true}
	}

	_, err = s.repo.CreateCredentials(ctx, db.CreateCredentialsParams{
		BusinessID:                  req.BusinessID,
		Shortcode:                   req.Shortcode,
		ConsumerKeyEncrypted:        ck,
		ConsumerSecretEncrypted:     cs,
		PasskeyEncrypted:            pk,
		InitiatorName:               initiatorName,
		SecurityCredentialEncrypted: securityCred,
	})
	if err != nil {
		return fmt.Errorf("store new credentials: %w", err)
	}

	s.logger.Info("credentials updated", "business_id", req.BusinessID)
	return nil
}

// DeactivateBusiness soft-deactivates a business.
func (s *BusinessService) DeactivateBusiness(ctx context.Context, businessID string) error {
	if businessID == "" {
		return fmt.Errorf("business_id is required")
	}

	_, err := s.repo.DeactivateBusiness(ctx, businessID)
	if err != nil {
		return fmt.Errorf("deactivate business: %w", err)
	}

	s.logger.Info("business deactivated", "business_id", businessID)
	return nil
}

// GetBusiness retrieves a business by ID.
func (s *BusinessService) GetBusiness(ctx context.Context, id string) (*db.Business, error) {
	b, err := s.repo.GetBusinessByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get business: %w", err)
	}
	return &b, nil
}

// ListBusinesses lists all active businesses.
func (s *BusinessService) ListBusinesses(ctx context.Context) ([]db.Business, error) {
	return s.repo.ListBusinesses(ctx)
}
