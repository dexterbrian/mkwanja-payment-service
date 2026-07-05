package service

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"mkwanja-payment-svc/internal/crypto"
	"mkwanja-payment-svc/internal/daraja"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// mockPaymentRepo is a test double for repository.PaymentRepo.
type mockPaymentRepo struct {
	createPaymentFn               func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error)
	getPaymentByIDFn              func(ctx context.Context, id string) (db.Payment, error)
	getPaymentByIdempotencyKeyFn  func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error)
	updatePaymentStatusFn         func(ctx context.Context, id string, status db.PaymentStatus) (db.Payment, error)
	completePaymentFn             func(ctx context.Context, id string, providerReceipt, providerTxID sql.NullString) (db.Payment, error)
	failPaymentFn                 func(ctx context.Context, id string) (db.Payment, error)
	updateProviderRequestIDFn     func(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error)
	listPaymentsByClientFn        func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error)
	listPendingPaymentsOlderThanFn func(ctx context.Context, createdAt time.Time) ([]db.Payment, error)
	createPaymentEventFn          func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error)
}

func (m *mockPaymentRepo) CreatePayment(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
	return m.createPaymentFn(ctx, params)
}
func (m *mockPaymentRepo) GetPaymentByID(ctx context.Context, id string) (db.Payment, error) {
	return m.getPaymentByIDFn(ctx, id)
}
func (m *mockPaymentRepo) GetPaymentByIdempotencyKey(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
	return m.getPaymentByIdempotencyKeyFn(ctx, clientID, idempotencyKey)
}
func (m *mockPaymentRepo) UpdatePaymentStatus(ctx context.Context, id string, status db.PaymentStatus) (db.Payment, error) {
	return m.updatePaymentStatusFn(ctx, id, status)
}
func (m *mockPaymentRepo) CompletePayment(ctx context.Context, id string, providerReceipt, providerTxID sql.NullString) (db.Payment, error) {
	return m.completePaymentFn(ctx, id, providerReceipt, providerTxID)
}
func (m *mockPaymentRepo) FailPayment(ctx context.Context, id string) (db.Payment, error) {
	return m.failPaymentFn(ctx, id)
}
func (m *mockPaymentRepo) UpdateProviderRequestID(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error) {
	return m.updateProviderRequestIDFn(ctx, id, providerRequestID, providerTxID)
}
func (m *mockPaymentRepo) ListPaymentsByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
	return m.listPaymentsByClientFn(ctx, clientID, limit, offset)
}
func (m *mockPaymentRepo) ListPendingPaymentsOlderThan(ctx context.Context, createdAt time.Time) ([]db.Payment, error) {
	return m.listPendingPaymentsOlderThanFn(ctx, createdAt)
}
func (m *mockPaymentRepo) CreatePaymentEvent(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
	return m.createPaymentEventFn(ctx, params)
}

var _ repository.PaymentRepo = (*mockPaymentRepo)(nil)

// mockDarajaClient is a test double for DarajaClient.
type mockDarajaClient struct {
	initiateSTKPushFn func(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error)
	initiateB2CFn     func(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error)
	initiateB2BFn     func(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error)
}

func (m *mockDarajaClient) InitiateSTKPush(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error) {
	return m.initiateSTKPushFn(ctx, req)
}
func (m *mockDarajaClient) InitiateB2C(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error) {
	return m.initiateB2CFn(ctx, req)
}
func (m *mockDarajaClient) InitiateB2B(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error) {
	return m.initiateB2BFn(ctx, req)
}

func TestInitiateSTKPushRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     InitiateSTKPushRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: false,
		},
		{
			name:    "empty request",
			req:     InitiateSTKPushRequest{},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name: "zero amount_cents",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: true,
			errMsg:  "amount_cents must be positive",
		},
		{
			name: "missing phone_number",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 100,
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: true,
			errMsg:  "phone_number is required",
		},
		{
			name: "missing reference",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "254712345678",
				IdempotencyKey: "key-1",
			},
			wantErr: true,
			errMsg:  "reference is required",
		},
		{
			name: "missing idempotency_key",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001",
			},
			wantErr: true,
			errMsg:  "idempotency_key is required",
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

func TestInitiateB2CRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     InitiateB2CRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: InitiateB2CRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: false,
		},
		{
			name:    "empty request",
			req:     InitiateB2CRequest{},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name: "missing phone_number",
			req: InitiateB2CRequest{
				ClientID: "client-1", AmountCents: 100,
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: true,
			errMsg:  "phone_number is required",
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

func TestInitiateB2BRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     InitiateB2BRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: InitiateB2BRequest{
				ClientID: "client-1", AmountCents: 100, ReceiverShortcode: "654321",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: false,
		},
		{
			name:    "empty request",
			req:     InitiateB2BRequest{},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name: "missing receiver_shortcode",
			req: InitiateB2BRequest{
				ClientID: "client-1", AmountCents: 100,
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			wantErr: true,
			errMsg:  "receiver_shortcode is required",
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

func TestPaymentService_InitiateSTKPush(t *testing.T) {
	encryptKey := make([]byte, 32)
	now := time.Now()

	tests := []struct {
		name       string
		req        InitiateSTKPushRequest
		mockSetup  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient)
		wantErr    bool
		errContain string
		wantID     string
	}{
		{
			name: "validation fails — missing client_id",
			req: InitiateSTKPushRequest{
				AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			mockSetup:  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {},
			wantErr:    true,
			errContain: "validation",
		},
		{
			name: "invalid phone format",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "12345",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			mockSetup:  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {},
			wantErr:    true,
			errContain: "phone validation",
		},
		{
			name: "idempotency — existing payment returned",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "dup-key",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{ID: "existing-payment", Status: db.PaymentStatusPending}, nil
				}
			},
			wantErr: false,
			wantID:  "existing-payment",
		},
		{
			name: "successful stk push initiation",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 10000, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-001",
				Description: "test payment",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				ek, _ := crypto.Encrypt(encryptKey, "")
				es, _ := crypto.Encrypt(encryptKey, "")
				ep, _ := crypto.Encrypt(encryptKey, "")

				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
				m.createPaymentFn = func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
					return db.Payment{
						ID: "payment-001", ClientID: params.ClientID,
						Status: db.PaymentStatusPending, CreatedAt: now,
						Provider: db.PaymentProviderMpesa, PaymentType: db.PaymentTypeStkPush,
					}, nil
				}
				mc.getActiveCredentialsFn = func(ctx context.Context, clientID string) (db.ClientCredential, error) {
					return db.ClientCredential{
						ID: "cred-001", Shortcode: "123456",
						ConsumerKeyEncrypted:    ek,
						ConsumerSecretEncrypted: es,
						PasskeyEncrypted:        ep,
						IsActive: true,
					}, nil
				}
				m.updateProviderRequestIDFn = func(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error) {
					return db.Payment{}, nil
				}
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
				}
				dc.initiateSTKPushFn = func(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error) {
					return &daraja.STKPushResponse{
						CheckoutRequestID: "checkout-001",
						ResponseCode:      "0",
						ResponseDescription: "Success",
					}, nil
				}
			},
			wantErr: false,
			wantID:  "payment-001",
		},
		{
			name: "daraja call fails — payment failed",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 10000, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-002",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				ek, _ := crypto.Encrypt(encryptKey, "")
				es, _ := crypto.Encrypt(encryptKey, "")
				ep, _ := crypto.Encrypt(encryptKey, "")

				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
				m.createPaymentFn = func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
					return db.Payment{ID: "payment-002", Status: db.PaymentStatusPending}, nil
				}
				mc.getActiveCredentialsFn = func(ctx context.Context, clientID string) (db.ClientCredential, error) {
					return db.ClientCredential{
						ID: "cred-001", Shortcode: "123456",
						ConsumerKeyEncrypted:    ek,
						ConsumerSecretEncrypted: es,
						PasskeyEncrypted:        ep,
						IsActive: true,
					}, nil
				}
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
				}
				dc.initiateSTKPushFn = func(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error) {
					return nil, context.DeadlineExceeded
				}
			},
			wantErr:    true,
			errContain: "daraja stk push",
		},
		{
			name: "missing credentials — payment failed",
			req: InitiateSTKPushRequest{
				ClientID: "client-1", AmountCents: 10000, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-003",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
				m.createPaymentFn = func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
					return db.Payment{ID: "payment-003", Status: db.PaymentStatusPending}, nil
				}
				mc.getActiveCredentialsFn = func(ctx context.Context, clientID string) (db.ClientCredential, error) {
					return db.ClientCredential{}, sql.ErrNoRows
				}
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
				}
			},
			wantErr:    true,
			errContain: "get active credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{}
			mockClient := &mockClientRepo{}
			mockDC := &mockDarajaClient{}
			tt.mockSetup(mockPay, mockClient, mockDC)

			svc := NewPaymentServiceForTest(mockPay, mockClient, nil, encryptKey, "https://example.com/callback",
				redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
				func(_, _, _, _ string) DarajaClient {
					return mockDC
				}, nil)

			result, err := svc.InitiateSTKPush(context.Background(), tt.req)

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
				if result.PaymentID != tt.wantID {
					t.Fatalf("expected payment ID %q, got %q", tt.wantID, result.PaymentID)
				}
			}
		})
	}
}

func TestPaymentService_InitiateB2C(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name       string
		req        InitiateB2CRequest
		mockSetup  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient)
		wantErr    bool
		errContain string
		wantID     string
	}{
		{
			name: "validation fails",
			req: InitiateB2CRequest{
				AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			mockSetup:  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {},
			wantErr:    true,
			errContain: "validation",
		},
		{
			name: "idempotency — returns existing",
			req: InitiateB2CRequest{
				ClientID: "client-1", AmountCents: 100, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "dup-key",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{ID: "existing-payment", Status: db.PaymentStatusPending}, nil
				}
			},
			wantErr: false,
			wantID:  "existing-payment",
		},
		{
			name: "successful b2c",
			req: InitiateB2CRequest{
				ClientID: "client-1", AmountCents: 5000, PhoneNumber: "254712345678",
				Reference: "ref-001", IdempotencyKey: "key-001",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				ek, _ := crypto.Encrypt(encryptKey, "")
				es, _ := crypto.Encrypt(encryptKey, "")
				ep, _ := crypto.Encrypt(encryptKey, "")

				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
				m.createPaymentFn = func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
					return db.Payment{ID: "payment-b2c-001", Status: db.PaymentStatusPending}, nil
				}
				mc.getActiveCredentialsFn = func(ctx context.Context, clientID string) (db.ClientCredential, error) {
					return db.ClientCredential{
						ID: "cred-001", Shortcode: "123456",
						ConsumerKeyEncrypted:    ek,
						ConsumerSecretEncrypted: es,
						PasskeyEncrypted:        ep,
						InitiatorName:           sql.NullString{String: "apitest", Valid: true},
						IsActive:                true,
					}, nil
				}
				m.updateProviderRequestIDFn = func(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error) {
					return db.Payment{}, nil
				}
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
				}
				dc.initiateB2CFn = func(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error) {
					return &daraja.B2CResponse{
						ConversationID: "conv-001",
						ResponseCode:   "0",
					}, nil
				}
			},
			wantErr: false,
			wantID:  "payment-b2c-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{}
			mockClient := &mockClientRepo{}
			mockDC := &mockDarajaClient{}
			tt.mockSetup(mockPay, mockClient, mockDC)

			svc := NewPaymentServiceForTest(mockPay, mockClient, nil, encryptKey, "https://example.com/callback",
				redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
				func(_, _, _, _ string) DarajaClient {
					return mockDC
				}, nil)

			result, err := svc.InitiateB2C(context.Background(), tt.req)

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
				if result.PaymentID != tt.wantID {
					t.Fatalf("expected payment ID %q, got %q", tt.wantID, result.PaymentID)
				}
			}
		})
	}
}

func TestPaymentService_InitiateB2B(t *testing.T) {
	encryptKey := make([]byte, 32)

	tests := []struct {
		name       string
		req        InitiateB2BRequest
		mockSetup  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient)
		wantErr    bool
		errContain string
		wantID     string
	}{
		{
			name: "validation fails",
			req: InitiateB2BRequest{
				AmountCents: 100, ReceiverShortcode: "654321",
				Reference: "ref-001", IdempotencyKey: "key-1",
			},
			mockSetup:  func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {},
			wantErr:    true,
			errContain: "validation",
		},
		{
			name: "idempotency — returns existing",
			req: InitiateB2BRequest{
				ClientID: "client-1", AmountCents: 100, ReceiverShortcode: "654321",
				Reference: "ref-001", IdempotencyKey: "dup-key",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{ID: "existing-b2b", Status: db.PaymentStatusPending}, nil
				}
			},
			wantErr: false,
			wantID:  "existing-b2b",
		},
		{
			name: "successful b2b",
			req: InitiateB2BRequest{
				ClientID: "client-1", AmountCents: 10000, ReceiverShortcode: "654321",
				Reference: "ref-001", IdempotencyKey: "key-001",
			},
			mockSetup: func(m *mockPaymentRepo, mc *mockClientRepo, dc *mockDarajaClient) {
				ek, _ := crypto.Encrypt(encryptKey, "")
				es, _ := crypto.Encrypt(encryptKey, "")
				ep, _ := crypto.Encrypt(encryptKey, "")

				m.getPaymentByIdempotencyKeyFn = func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
				m.createPaymentFn = func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
					return db.Payment{ID: "payment-b2b-001", Status: db.PaymentStatusPending}, nil
				}
				mc.getActiveCredentialsFn = func(ctx context.Context, clientID string) (db.ClientCredential, error) {
					return db.ClientCredential{
						ID: "cred-001", Shortcode: "123456",
						ConsumerKeyEncrypted:    ek,
						ConsumerSecretEncrypted: es,
						PasskeyEncrypted:        ep,
						InitiatorName:           sql.NullString{String: "apitest", Valid: true},
						IsActive:                true,
					}, nil
				}
				m.updateProviderRequestIDFn = func(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error) {
					return db.Payment{}, nil
				}
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
				}
				dc.initiateB2BFn = func(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error) {
					return &daraja.B2BResponse{
						ConversationID: "conv-b2b-001",
						ResponseCode:   "0",
					}, nil
				}
			},
			wantErr: false,
			wantID:  "payment-b2b-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{}
			mockClient := &mockClientRepo{}
			mockDC := &mockDarajaClient{}
			tt.mockSetup(mockPay, mockClient, mockDC)

			svc := NewPaymentServiceForTest(mockPay, mockClient, nil, encryptKey, "https://example.com/callback",
				redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
				func(_, _, _, _ string) DarajaClient {
					return mockDC
				}, nil)

			result, err := svc.InitiateB2B(context.Background(), tt.req)

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
				if result.PaymentID != tt.wantID {
					t.Fatalf("expected payment ID %q, got %q", tt.wantID, result.PaymentID)
				}
			}
		})
	}
}

func TestPaymentService_GetPayment(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		mockFn     func(ctx context.Context, id string) (db.Payment, error)
		wantErr    bool
		errContain string
		wantID     string
	}{
		{
			name: "found",
			id:   "payment-001",
			mockFn: func(ctx context.Context, id string) (db.Payment, error) {
				return db.Payment{ID: id, Status: db.PaymentStatusCompleted}, nil
			},
			wantErr: false,
			wantID:  "payment-001",
		},
		{
			name: "not found",
			id:   "payment-999",
			mockFn: func(ctx context.Context, id string) (db.Payment, error) {
				return db.Payment{}, sql.ErrNoRows
			},
			wantErr:    true,
			errContain: "get payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{
				getPaymentByIDFn: tt.mockFn,
			}
			svc := &PaymentService{
				paymentRepo: mockPay,
				logger:      slog.Default(),
			}

			payment, err := svc.GetPayment(context.Background(), tt.id)

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
				if payment.ID != tt.wantID {
					t.Fatalf("expected ID %q, got %q", tt.wantID, payment.ID)
				}
			}
		})
	}
}

func TestPaymentService_ListPayments(t *testing.T) {
	tests := []struct {
		name      string
		clientID  string
		limit     int32
		offset    int32
		mockFn    func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error)
		wantErr   bool
		wantCount int
	}{
		{
			name: "empty list",
			mockFn: func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
				return []db.Payment{}, nil
			},
			wantCount: 0,
		},
		{
			name: "two payments",
			mockFn: func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
				return []db.Payment{
					{ID: "p1", Status: db.PaymentStatusCompleted},
					{ID: "p2", Status: db.PaymentStatusPending},
				}, nil
			},
			wantCount: 2,
		},
		{
			name: "repo error",
			mockFn: func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
				return nil, context.DeadlineExceeded
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{
				listPaymentsByClientFn: tt.mockFn,
			}
			svc := &PaymentService{
				paymentRepo: mockPay,
				logger:      nil,
			}

			payments, err := svc.ListPayments(context.Background(), tt.clientID, tt.limit, tt.offset)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(payments) != tt.wantCount {
					t.Fatalf("expected %d payments, got %d", tt.wantCount, len(payments))
				}
			}
		})
	}
}

func TestPaymentService_CompletePayment(t *testing.T) {
	tests := []struct {
		name       string
		paymentID  string
		receipt    string
		txID       string
		mockSetup  func(m *mockPaymentRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success",
			mockSetup: func(m *mockPaymentRepo) {
				m.completePaymentFn = func(ctx context.Context, id string, receipt, txID sql.NullString) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusCompleted}, nil
				}
				m.createPaymentEventFn = func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
					return db.PaymentEvent{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "repo error",
			mockSetup: func(m *mockPaymentRepo) {
				m.completePaymentFn = func(ctx context.Context, id string, receipt, txID sql.NullString) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
			},
			wantErr:    true,
			errContain: "complete payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{}
			tt.mockSetup(mockPay)
			svc := &PaymentService{paymentRepo: mockPay, logger: slog.Default()}

			err := svc.CompletePayment(context.Background(), tt.paymentID, tt.receipt, tt.txID)

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

func TestPaymentService_FailPayment(t *testing.T) {
	tests := []struct {
		name       string
		paymentID  string
		reason     string
		mockSetup  func(m *mockPaymentRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success",
			mockSetup: func(m *mockPaymentRepo) {
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
				}
				m.createPaymentEventFn = func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
					return db.PaymentEvent{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "repo error",
			mockSetup: func(m *mockPaymentRepo) {
				m.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
					return db.Payment{}, sql.ErrNoRows
				}
			},
			wantErr:    true,
			errContain: "fail payment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPay := &mockPaymentRepo{}
			tt.mockSetup(mockPay)
			svc := &PaymentService{paymentRepo: mockPay, logger: slog.Default()}

			err := svc.FailPayment(context.Background(), tt.paymentID, tt.reason)

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
