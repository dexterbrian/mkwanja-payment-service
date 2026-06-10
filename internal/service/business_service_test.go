package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// mockBusinessRepo is a test double for repository.BusinessRepo.
type mockBusinessRepo struct {
	createBusinessFn        func(ctx context.Context, params db.CreateBusinessParams) (db.Business, error)
	getBusinessByIDFn       func(ctx context.Context, id string) (db.Business, error)
	getBusinessByExtIDFn    func(ctx context.Context, externalID string) (db.Business, error)
	listBusinessesFn        func(ctx context.Context) ([]db.Business, error)
	deactivateBusinessFn    func(ctx context.Context, id string) (db.Business, error)
	createCredentialsFn     func(ctx context.Context, params db.CreateCredentialsParams) (db.BusinessCredential, error)
	getActiveCredentialsFn  func(ctx context.Context, businessID string) (db.BusinessCredential, error)
	deactivateCredentialsFn func(ctx context.Context, businessID string) error
	createJournalAcctFn     func(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error)
	listJournalAcctsFn      func(ctx context.Context, businessID string) ([]db.JournalAccount, error)
}

func (m *mockBusinessRepo) CreateBusiness(ctx context.Context, params db.CreateBusinessParams) (db.Business, error) {
	return m.createBusinessFn(ctx, params)
}
func (m *mockBusinessRepo) GetBusinessByID(ctx context.Context, id string) (db.Business, error) {
	return m.getBusinessByIDFn(ctx, id)
}
func (m *mockBusinessRepo) GetBusinessByExternalID(ctx context.Context, externalID string) (db.Business, error) {
	return m.getBusinessByExtIDFn(ctx, externalID)
}
func (m *mockBusinessRepo) ListBusinesses(ctx context.Context) ([]db.Business, error) {
	return m.listBusinessesFn(ctx)
}
func (m *mockBusinessRepo) DeactivateBusiness(ctx context.Context, id string) (db.Business, error) {
	return m.deactivateBusinessFn(ctx, id)
}
func (m *mockBusinessRepo) CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.BusinessCredential, error) {
	return m.createCredentialsFn(ctx, params)
}
func (m *mockBusinessRepo) GetActiveCredentials(ctx context.Context, businessID string) (db.BusinessCredential, error) {
	return m.getActiveCredentialsFn(ctx, businessID)
}
func (m *mockBusinessRepo) DeactivateCredentials(ctx context.Context, businessID string) error {
	return m.deactivateCredentialsFn(ctx, businessID)
}
func (m *mockBusinessRepo) CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
	return m.createJournalAcctFn(ctx, params)
}
func (m *mockBusinessRepo) ListJournalAccounts(ctx context.Context, businessID string) ([]db.JournalAccount, error) {
	return m.listJournalAcctsFn(ctx, businessID)
}

var _ repository.BusinessRepo = (*mockBusinessRepo)(nil)

func TestRegisterBusinessRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterBusinessRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: false,
		},
		{
			name:    "empty request",
			req:     RegisterBusinessRequest{},
			wantErr: true,
			errMsg:  "external_id is required",
		},
		{
			name: "missing name",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing shortcode",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "shortcode is required",
		},
		{
			name: "missing consumer_key",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "consumer_key is required",
		},
		{
			name: "missing consumer_secret",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "consumer_secret is required",
		},
		{
			name: "missing passkey",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret",
			},
			wantErr: true,
			errMsg:  "passkey is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.errMsg)
				}
				if err.Error() != tt.errMsg {
					t.Fatalf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}

func TestTestCredentialsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     TestCredentialsRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid",
			req:     TestCredentialsRequest{ConsumerKey: "k", ConsumerSecret: "s", Shortcode: "123"},
			wantErr: false,
		},
		{
			name:    "missing consumer_key",
			req:     TestCredentialsRequest{ConsumerSecret: "s", Shortcode: "123"},
			wantErr: true,
			errMsg:  "consumer_key is required",
		},
		{
			name:    "missing consumer_secret",
			req:     TestCredentialsRequest{ConsumerKey: "k", Shortcode: "123"},
			wantErr: true,
			errMsg:  "consumer_secret is required",
		},
		{
			name:    "missing shortcode",
			req:     TestCredentialsRequest{ConsumerKey: "k", ConsumerSecret: "s"},
			wantErr: true,
			errMsg:  "shortcode is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.errMsg)
				}
				if err.Error() != tt.errMsg {
					t.Fatalf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}

func TestUpdateCredentialsRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateCredentialsRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			req: UpdateCredentialsRequest{
				BusinessID: "biz-1", ConsumerKey: "k", ConsumerSecret: "s", Passkey: "p",
			},
			wantErr: false,
		},
		{
			name:    "missing business_id",
			req:     UpdateCredentialsRequest{ConsumerKey: "k", ConsumerSecret: "s", Passkey: "p"},
			wantErr: true,
			errMsg:  "business_id is required",
		},
		{
			name:    "missing consumer_key",
			req:     UpdateCredentialsRequest{BusinessID: "biz-1", ConsumerSecret: "s", Passkey: "p"},
			wantErr: true,
			errMsg:  "consumer_key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.errMsg)
				}
				if err.Error() != tt.errMsg {
					t.Fatalf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}

func TestBusinessService_RegisterBusiness(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name       string
		req        RegisterBusinessRequest
		mockSetup  func(m *mockBusinessRepo)
		wantErr    bool
		errContain string
		wantBizID  string
	}{
		{
			name: "validation fails — missing external_id",
			req: RegisterBusinessRequest{
				Name: "Acme", Shortcode: "123", ConsumerKey: "k", ConsumerSecret: "s", Passkey: "p",
			},
			mockSetup:  func(m *mockBusinessRepo) {},
			wantErr:    true,
			errContain: "validation",
		},
		{
			name: "create business succeeds",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			mockSetup: func(m *mockBusinessRepo) {
				m.createBusinessFn = func(ctx context.Context, params db.CreateBusinessParams) (db.Business, error) {
					return db.Business{ID: "biz-001", ExternalID: params.ExternalID, Name: params.Name, Active: true}, nil
				}
				m.createCredentialsFn = func(ctx context.Context, params db.CreateCredentialsParams) (db.BusinessCredential, error) {
					return db.BusinessCredential{ID: "cred-001", BusinessID: params.BusinessID, IsActive: true}, nil
				}
				m.createJournalAcctFn = func(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
					return db.JournalAccount{ID: params.ID, BusinessID: params.BusinessID, Name: params.Name}, nil
				}
			},
			wantErr:   false,
			wantBizID: "biz-001",
		},
		{
			name: "create business fails",
			req: RegisterBusinessRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			mockSetup: func(m *mockBusinessRepo) {
				m.createBusinessFn = func(ctx context.Context, params db.CreateBusinessParams) (db.Business, error) {
					return db.Business{}, context.DeadlineExceeded
				}
			},
			wantErr:    true,
			errContain: "create business",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBusinessRepo{}
			tt.mockSetup(mock)

			svc := NewBusinessService(mock, encryptKey, nil)
			biz, err := svc.RegisterBusiness(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error containing %q, got %q", tt.errContain, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if biz.ID != tt.wantBizID {
					t.Fatalf("expected biz ID %q, got %q", tt.wantBizID, biz.ID)
				}
			}
		})
	}
}

func TestBusinessService_DeactivateBusiness(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name       string
		bizID      string
		mockFn     func(ctx context.Context, id string) (db.Business, error)
		wantErr    bool
		errContain string
	}{
		{
			name:       "empty business_id",
			bizID:      "",
			mockFn:     nil,
			wantErr:    true,
			errContain: "business_id is required",
		},
		{
			name:  "successful deactivation",
			bizID: "biz-001",
			mockFn: func(ctx context.Context, id string) (db.Business, error) {
				return db.Business{ID: id, Active: false}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBusinessRepo{
				deactivateBusinessFn: tt.mockFn,
			}
			svc := NewBusinessService(mock, encryptKey, nil)
			err := svc.DeactivateBusiness(context.Background(), tt.bizID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("expected error containing %q, got %q", tt.errContain, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBusinessService_GetBusiness(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name    string
		id      string
		mockFn  func(ctx context.Context, id string) (db.Business, error)
		wantErr bool
		wantID  string
	}{
		{
			name: "found",
			id:   "biz-001",
			mockFn: func(ctx context.Context, id string) (db.Business, error) {
				return db.Business{ID: id, Name: "Acme"}, nil
			},
			wantErr: false,
			wantID:  "biz-001",
		},
		{
			name: "not found",
			id:   "biz-999",
			mockFn: func(ctx context.Context, id string) (db.Business, error) {
				return db.Business{}, sql.ErrNoRows
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockBusinessRepo{
				getBusinessByIDFn: tt.mockFn,
			}
			svc := NewBusinessService(mock, encryptKey, nil)
			biz, err := svc.GetBusiness(context.Background(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if biz.ID != tt.wantID {
					t.Fatalf("expected ID %q, got %q", tt.wantID, biz.ID)
				}
			}
		})
	}
}
