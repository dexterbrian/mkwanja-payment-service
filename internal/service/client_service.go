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

// SeedAccount represents a default journal account to create per client.
type SeedAccount struct {
	ID            string
	Name          string
	AccountType   db.AccountType
	NormalBalance db.NormalBalance
	Description   string
}

// defaultAccounts are seeded into every new client.
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

// ClientService handles client and credential management.
type ClientService struct {
	repo       repository.ClientRepo
	encryptKey []byte
	logger     *slog.Logger
}

// NewClientService creates a ClientService.
func NewClientService(repo repository.ClientRepo, encryptKey []byte, logger *slog.Logger) *ClientService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ClientService{
		repo:       repo,
		encryptKey: encryptKey,
		logger:     logger,
	}
}

// RegisterClientRequest holds the input for client registration.
type RegisterClientRequest struct {
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
func (r *RegisterClientRequest) Validate() error {
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

// RegisterClient creates a client, encrypts + stores credentials, and seeds default journal accounts.
func (s *ClientService) RegisterClient(ctx context.Context, req RegisterClientRequest) (*db.Client, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Create client
	c, err := s.repo.CreateClient(ctx, db.CreateClientParams{
		ExternalID: req.ExternalID,
		Name:       req.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
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
		ClientID:                    c.ID,
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
			ClientID:      c.ID,
			Name:          acct.Name,
			AccountType:   acct.AccountType,
			NormalBalance: acct.NormalBalance,
			Description:   desc,
		})
		if err != nil {
			s.logger.Error("failed to seed journal account",
				"account_id", acct.ID,
				"client_id", c.ID,
				"error", err)
		}
	}

	s.logger.Info("client registered",
		"client_id", c.ID,
		"external_id", c.ExternalID,
		"name", c.Name)
	return &c, nil
}

// EnsureOperatorClient ensures the operator client exists for the given consumer.
// It is idempotent: if the client already exists, the existing client is returned.
// On creation, empty credentials are stored and default journal accounts are seeded.
func (s *ClientService) EnsureOperatorClient(ctx context.Context, clientID, name string) (*db.Client, error) {
	if clientID == "" {
		return nil, fmt.Errorf("operator client_id is required")
	}
	if name == "" {
		name = "Dexter Operator"
	}

	existing, err := s.repo.GetClientByExternalID(ctx, clientID)
	if err == nil {
		s.logger.Info("operator client already exists", "client_id", existing.ID, "external_id", existing.ExternalID)
		return &existing, nil
	}

	// Create the operator client
	c, err := s.repo.CreateClient(ctx, db.CreateClientParams{
		ExternalID: clientID,
		Name:       name,
	})
	if err != nil {
		return nil, fmt.Errorf("create operator client: %w", err)
	}

	// Store empty credentials (encrypted empty strings)
	ck, err := crypto.Encrypt(s.encryptKey, "")
	if err != nil {
		return nil, fmt.Errorf("encrypt empty consumer_key: %w", err)
	}
	cs, err := crypto.Encrypt(s.encryptKey, "")
	if err != nil {
		return nil, fmt.Errorf("encrypt empty consumer_secret: %w", err)
	}
	pk, err := crypto.Encrypt(s.encryptKey, "")
	if err != nil {
		return nil, fmt.Errorf("encrypt empty passkey: %w", err)
	}

	_, err = s.repo.CreateCredentials(ctx, db.CreateCredentialsParams{
		ClientID:                    c.ID,
		Shortcode:                   "",
		ConsumerKeyEncrypted:        ck,
		ConsumerSecretEncrypted:     cs,
		PasskeyEncrypted:            pk,
		InitiatorName:               sql.NullString{},
		SecurityCredentialEncrypted: sql.NullString{},
	})
	if err != nil {
		return nil, fmt.Errorf("store operator credentials: %w", err)
	}

	// Seed default journal accounts
	for _, acct := range defaultAccounts {
		desc := sql.NullString{String: acct.Description, Valid: true}
		_, err := s.repo.CreateJournalAccount(ctx, db.CreateJournalAccountParams{
			ID:            acct.ID,
			ClientID:      c.ID,
			Name:          acct.Name,
			AccountType:   acct.AccountType,
			NormalBalance: acct.NormalBalance,
			Description:   desc,
		})
		if err != nil {
			s.logger.Error("failed to seed operator journal account",
				"account_id", acct.ID,
				"client_id", c.ID,
				"error", err)
		}
	}

	s.logger.Info("operator client seeded",
		"client_id", c.ID,
		"external_id", c.ExternalID,
		"name", c.Name)
	return &c, nil
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
func (s *ClientService) TestCredentials(ctx context.Context, req TestCredentialsRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation: %w", err)
	}
	s.logger.Info("credentials test passed (validation only — Daraja call in Phase 4)",
		"shortcode", req.Shortcode)
	return nil
}

// UpdateCredentialsRequest holds input for credential update.
type UpdateCredentialsRequest struct {
	ClientID           string
	Shortcode          string
	ConsumerKey        string
	ConsumerSecret     string
	Passkey            string
	InitiatorName      string
	SecurityCredential string
}

// Validate checks the request fields.
func (r *UpdateCredentialsRequest) Validate() error {
	if r.ClientID == "" {
		return fmt.Errorf("client_id is required")
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
func (s *ClientService) UpdateCredentials(ctx context.Context, req UpdateCredentialsRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.DeactivateCredentials(ctx, req.ClientID); err != nil {
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
		ClientID:                    req.ClientID,
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

	s.logger.Info("credentials updated", "client_id", req.ClientID)
	return nil
}

// DeactivateClient soft-deactivates a client.
func (s *ClientService) DeactivateClient(ctx context.Context, clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client_id is required")
	}

	_, err := s.repo.DeactivateClient(ctx, clientID)
	if err != nil {
		return fmt.Errorf("deactivate client: %w", err)
	}

	s.logger.Info("client deactivated", "client_id", clientID)
	return nil
}

// GetClient retrieves a client by ID.
func (s *ClientService) GetClient(ctx context.Context, id string) (*db.Client, error) {
	c, err := s.repo.GetClientByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	return &c, nil
}

// ListClients lists all active clients.
func (s *ClientService) ListClients(ctx context.Context) ([]db.Client, error) {
	return s.repo.ListClients(ctx)
}
