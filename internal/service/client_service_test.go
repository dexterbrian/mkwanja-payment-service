package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// mockClientRepo is a test double for repository.ClientRepo.
type mockClientRepo struct {
	createClientFn        func(ctx context.Context, params db.CreateClientParams) (db.Client, error)
	getClientByIDFn       func(ctx context.Context, id string) (db.Client, error)
	getClientByExtIDFn    func(ctx context.Context, externalID string) (db.Client, error)
	listClientsFn         func(ctx context.Context) ([]db.Client, error)
	deactivateClientFn    func(ctx context.Context, id string) (db.Client, error)
	createCredentialsFn   func(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error)
	getActiveCredentialsFn func(ctx context.Context, clientID string) (db.ClientCredential, error)
	deactivateCredentialsFn func(ctx context.Context, clientID string) error
	createJournalAcctFn   func(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error)
	listJournalAcctsFn    func(ctx context.Context, clientID string) ([]db.JournalAccount, error)
}

func (m *mockClientRepo) CreateClient(ctx context.Context, params db.CreateClientParams) (db.Client, error) {
	return m.createClientFn(ctx, params)
}
func (m *mockClientRepo) GetClientByID(ctx context.Context, id string) (db.Client, error) {
	return m.getClientByIDFn(ctx, id)
}
func (m *mockClientRepo) GetClientByExternalID(ctx context.Context, externalID string) (db.Client, error) {
	return m.getClientByExtIDFn(ctx, externalID)
}
func (m *mockClientRepo) ListClients(ctx context.Context) ([]db.Client, error) {
	return m.listClientsFn(ctx)
}
func (m *mockClientRepo) DeactivateClient(ctx context.Context, id string) (db.Client, error) {
	return m.deactivateClientFn(ctx, id)
}
func (m *mockClientRepo) CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error) {
	return m.createCredentialsFn(ctx, params)
}
func (m *mockClientRepo) GetActiveCredentials(ctx context.Context, clientID string) (db.ClientCredential, error) {
	return m.getActiveCredentialsFn(ctx, clientID)
}
func (m *mockClientRepo) DeactivateCredentials(ctx context.Context, clientID string) error {
	return m.deactivateCredentialsFn(ctx, clientID)
}
func (m *mockClientRepo) CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
	return m.createJournalAcctFn(ctx, params)
}
func (m *mockClientRepo) ListJournalAccounts(ctx context.Context, clientID string) ([]db.JournalAccount, error) {
	return m.listJournalAcctsFn(ctx, clientID)
}

var _ repository.ClientRepo = (*mockClientRepo)(nil)

func TestRegisterClientRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterClientRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: false,
		},
		{
			name:    "empty request",
			req:     RegisterClientRequest{},
			wantErr: true,
			errMsg:  "external_id is required",
		},
		{
			name: "missing name",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing shortcode",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Name: "Acme Ltd",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "shortcode is required",
		},
		{
			name: "missing consumer_key",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerSecret: "secret", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "consumer_key is required",
		},
		{
			name: "missing consumer_secret",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", Passkey: "passkey",
			},
			wantErr: true,
			errMsg:  "consumer_secret is required",
		},
		{
			name: "missing passkey",
			req: RegisterClientRequest{
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
				ClientID: "client-1", ConsumerKey: "k", ConsumerSecret: "s", Passkey: "p",
			},
			wantErr: false,
		},
		{
			name:    "missing client_id",
			req:     UpdateCredentialsRequest{ConsumerKey: "k", ConsumerSecret: "s", Passkey: "p"},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name:    "missing consumer_key",
			req:     UpdateCredentialsRequest{ClientID: "client-1", ConsumerSecret: "s", Passkey: "p"},
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

func TestClientService_RegisterClient(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name       string
		req        RegisterClientRequest
		mockSetup  func(m *mockClientRepo)
		wantErr    bool
		errContain string
		wantID     string
	}{
		{
			name: "validation fails — missing external_id",
			req: RegisterClientRequest{
				Name: "Acme", Shortcode: "123", ConsumerKey: "k", ConsumerSecret: "s", Passkey: "p",
			},
			mockSetup:  func(m *mockClientRepo) {},
			wantErr:    true,
			errContain: "validation",
		},
		{
			name: "create client succeeds",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			mockSetup: func(m *mockClientRepo) {
				m.createClientFn = func(ctx context.Context, params db.CreateClientParams) (db.Client, error) {
					return db.Client{ID: "client-001", ExternalID: params.ExternalID, Name: params.Name, Active: true}, nil
				}
				m.createCredentialsFn = func(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error) {
					return db.ClientCredential{ID: "cred-001", ClientID: params.ClientID, IsActive: true}, nil
				}
				m.createJournalAcctFn = func(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
					return db.JournalAccount{ID: params.ID, ClientID: params.ClientID, Name: params.Name}, nil
				}
			},
			wantErr: false,
			wantID:  "client-001",
		},
		{
			name: "create client fails",
			req: RegisterClientRequest{
				ExternalID: "ext-001", Name: "Acme Ltd", Shortcode: "123456",
				ConsumerKey: "key", ConsumerSecret: "secret", Passkey: "passkey",
			},
			mockSetup: func(m *mockClientRepo) {
				m.createClientFn = func(ctx context.Context, params db.CreateClientParams) (db.Client, error) {
					return db.Client{}, context.DeadlineExceeded
				}
			},
			wantErr:    true,
			errContain: "create client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClientRepo{}
			tt.mockSetup(mock)

			svc := NewClientService(mock, encryptKey, nil)
			client, err := svc.RegisterClient(context.Background(), tt.req)

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
				if client.ID != tt.wantID {
					t.Fatalf("expected client ID %q, got %q", tt.wantID, client.ID)
				}
			}
		})
	}
}

func TestClientService_DeactivateClient(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name       string
		clientID   string
		mockFn     func(ctx context.Context, id string) (db.Client, error)
		wantErr    bool
		errContain string
	}{
		{
			name:       "empty client_id",
			clientID:   "",
			mockFn:     nil,
			wantErr:    true,
			errContain: "client_id is required",
		},
		{
			name:  "successful deactivation",
			clientID: "client-001",
			mockFn: func(ctx context.Context, id string) (db.Client, error) {
				return db.Client{ID: id, Active: false}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClientRepo{
				deactivateClientFn: tt.mockFn,
			}
			svc := NewClientService(mock, encryptKey, nil)
			err := svc.DeactivateClient(context.Background(), tt.clientID)

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

func TestClientService_GetClient(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name    string
		id      string
		mockFn  func(ctx context.Context, id string) (db.Client, error)
		wantErr bool
		wantID  string
	}{
		{
			name: "found",
			id:   "client-001",
			mockFn: func(ctx context.Context, id string) (db.Client, error) {
				return db.Client{ID: id, Name: "Acme"}, nil
			},
			wantErr: false,
			wantID:  "client-001",
		},
		{
			name: "not found",
			id:   "client-999",
			mockFn: func(ctx context.Context, id string) (db.Client, error) {
				return db.Client{}, sql.ErrNoRows
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClientRepo{
				getClientByIDFn: tt.mockFn,
			}
			svc := NewClientService(mock, encryptKey, nil)
			client, err := svc.GetClient(context.Background(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if client.ID != tt.wantID {
					t.Fatalf("expected ID %q, got %q", tt.wantID, client.ID)
				}
			}
		})
	}
}
