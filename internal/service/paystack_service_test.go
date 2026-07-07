package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"mkwanja-payment-svc/internal/crypto"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/paystack"
	"mkwanja-payment-svc/internal/repository"
)

// mockPaystackRepo is a test double for repository.PaystackRepo.
type mockPaystackRepo struct {
	createFn     func(ctx context.Context, params db.CreatePaystackCredentialsParams) (db.ClientPaystackCredential, error)
	getActiveFn  func(ctx context.Context, clientID string) (db.ClientPaystackCredential, error)
	deactivateFn func(ctx context.Context, clientID string) error
}

func (m *mockPaystackRepo) CreatePaystackCredentials(ctx context.Context, params db.CreatePaystackCredentialsParams) (db.ClientPaystackCredential, error) {
	return m.createFn(ctx, params)
}
func (m *mockPaystackRepo) GetActivePaystackCredentials(ctx context.Context, clientID string) (db.ClientPaystackCredential, error) {
	return m.getActiveFn(ctx, clientID)
}
func (m *mockPaystackRepo) DeactivatePaystackCredentials(ctx context.Context, clientID string) error {
	return m.deactivateFn(ctx, clientID)
}

var _ repository.PaystackRepo = (*mockPaystackRepo)(nil)

// mockCompleter is a test double for paymentCompleter.
type mockCompleter struct {
	completedID string
	failedID    string
	failReason  string
}

func (m *mockCompleter) CompletePayment(_ context.Context, paymentID, _, _ string) error {
	m.completedID = paymentID
	return nil
}
func (m *mockCompleter) FailPayment(_ context.Context, paymentID, reason string) error {
	m.failedID = paymentID
	m.failReason = reason
	return nil
}

// mockPaystackAPI is a test double for PaystackAPI.
type mockPaystackAPI struct {
	initializeFn func(ctx context.Context, req paystack.InitializeTransactionRequest) (*paystack.InitializeTransactionData, error)
}

func (m *mockPaystackAPI) InitializeTransaction(ctx context.Context, req paystack.InitializeTransactionRequest) (*paystack.InitializeTransactionData, error) {
	return m.initializeFn(ctx, req)
}
func (m *mockPaystackAPI) VerifyTransaction(_ context.Context, _ string) (*paystack.VerifyTransactionData, error) {
	return nil, fmt.Errorf("not implemented")
}

var testEncryptKey = []byte("0123456789abcdef0123456789abcdef") // 32 bytes

func paystackSign(secretKey string, body []byte) string {
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestPaystackRegisterCredentials(t *testing.T) {
	var created db.CreatePaystackCredentialsParams
	deactivated := ""

	repo := &mockPaystackRepo{
		createFn: func(_ context.Context, params db.CreatePaystackCredentialsParams) (db.ClientPaystackCredential, error) {
			created = params
			return db.ClientPaystackCredential{ID: "cred-1", ClientID: params.ClientID, PublicKey: params.PublicKey, IsActive: true}, nil
		},
		deactivateFn: func(_ context.Context, clientID string) error {
			deactivated = clientID
			return nil
		},
	}
	svc := NewPaystackServiceForTest(nil, repo, nil, testEncryptKey, nil, nil)

	cred, err := svc.RegisterCredentials(context.Background(), RegisterPaystackCredentialsRequest{
		ClientID: "client-1", SecretKey: "sk_test_secret", PublicKey: "pk_test_pub",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.ID != "cred-1" {
		t.Fatalf("expected cred-1, got %s", cred.ID)
	}
	if deactivated != "client-1" {
		t.Fatalf("expected old credentials deactivated for client-1, got %q", deactivated)
	}
	if created.SecretKeyEncrypted == "sk_test_secret" {
		t.Fatal("secret key stored in plaintext")
	}
	plain, err := crypto.Decrypt(testEncryptKey, created.SecretKeyEncrypted)
	if err != nil || plain != "sk_test_secret" {
		t.Fatalf("stored secret not decryptable to original: %v", err)
	}
}

func TestPaystackRegisterCredentials_Validation(t *testing.T) {
	svc := NewPaystackServiceForTest(nil, nil, nil, testEncryptKey, nil, nil)

	_, err := svc.RegisterCredentials(context.Background(), RegisterPaystackCredentialsRequest{ClientID: "c1"})
	if err == nil || !strings.Contains(err.Error(), "secret_key is required") {
		t.Fatalf("expected secret_key validation error, got %v", err)
	}
}

func TestPaystackInitiateCharge_Success(t *testing.T) {
	encSecret, err := crypto.Encrypt(testEncryptKey, "sk_test_secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	var initReq paystack.InitializeTransactionRequest
	var providerReqID sql.NullString

	paymentRepo := &mockPaymentRepo{
		getPaymentByIdempotencyKeyFn: func(_ context.Context, _, _ string) (db.Payment, error) {
			return db.Payment{}, sql.ErrNoRows
		},
		createPaymentFn: func(_ context.Context, params db.CreatePaymentParams) (db.Payment, error) {
			return db.Payment{ID: "pay-1", ClientID: params.ClientID, AmountCents: params.AmountCents, Status: db.PaymentStatusPending}, nil
		},
		updateProviderRequestIDFn: func(_ context.Context, id string, requestID, _ sql.NullString) (db.Payment, error) {
			providerReqID = requestID
			return db.Payment{ID: id}, nil
		},
	}
	paystackRepo := &mockPaystackRepo{
		getActiveFn: func(_ context.Context, clientID string) (db.ClientPaystackCredential, error) {
			return db.ClientPaystackCredential{ClientID: clientID, SecretKeyEncrypted: encSecret, IsActive: true}, nil
		},
	}
	api := &mockPaystackAPI{
		initializeFn: func(_ context.Context, req paystack.InitializeTransactionRequest) (*paystack.InitializeTransactionData, error) {
			initReq = req
			return &paystack.InitializeTransactionData{
				AuthorizationURL: "https://checkout.paystack.com/abc123",
				AccessCode:       "abc123",
				Reference:        req.Reference,
			}, nil
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, paystackRepo, nil, testEncryptKey,
		func(secretKey string) PaystackAPI {
			if secretKey != "sk_test_secret" {
				t.Fatalf("client built with wrong secret key")
			}
			return api
		}, nil)

	res, err := svc.InitiateCharge(context.Background(), InitiatePaystackChargeRequest{
		ClientID: "client-1", AmountCents: 150050, Email: "a@b.com",
		Reference: "order-1", IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PaymentID != "pay-1" {
		t.Fatalf("expected pay-1, got %s", res.PaymentID)
	}
	if res.AuthorizationURL != "https://checkout.paystack.com/abc123" {
		t.Fatalf("unexpected authorization url: %s", res.AuthorizationURL)
	}
	// Paystack amounts are subunits — cents must pass through untruncated.
	if initReq.Amount != 150050 {
		t.Fatalf("expected amount 150050 subunits, got %d", initReq.Amount)
	}
	// Paystack reference must be our payment ID so webhooks route back.
	if initReq.Reference != "pay-1" {
		t.Fatalf("expected reference pay-1, got %s", initReq.Reference)
	}
	if !providerReqID.Valid || providerReqID.String != "pay-1" {
		t.Fatalf("expected provider request id pay-1, got %+v", providerReqID)
	}
}

func TestPaystackInitiateCharge_IdempotencyReplay(t *testing.T) {
	paymentRepo := &mockPaymentRepo{
		getPaymentByIdempotencyKeyFn: func(_ context.Context, _, _ string) (db.Payment, error) {
			return db.Payment{ID: "pay-existing", Status: db.PaymentStatusPending}, nil
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, nil, nil, testEncryptKey, nil, nil)

	res, err := svc.InitiateCharge(context.Background(), InitiatePaystackChargeRequest{
		ClientID: "client-1", AmountCents: 100, Email: "a@b.com",
		Reference: "order-1", IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PaymentID != "pay-existing" || !res.IdempotencyReplayed {
		t.Fatalf("expected idempotency replay of pay-existing, got %+v", res)
	}
}

func TestPaystackInitiateCharge_ProviderErrorFailsPayment(t *testing.T) {
	encSecret, _ := crypto.Encrypt(testEncryptKey, "sk_test_secret")
	failedID := ""

	paymentRepo := &mockPaymentRepo{
		getPaymentByIdempotencyKeyFn: func(_ context.Context, _, _ string) (db.Payment, error) {
			return db.Payment{}, sql.ErrNoRows
		},
		createPaymentFn: func(_ context.Context, params db.CreatePaymentParams) (db.Payment, error) {
			return db.Payment{ID: "pay-1", Status: db.PaymentStatusPending}, nil
		},
		failPaymentFn: func(_ context.Context, id string) (db.Payment, error) {
			failedID = id
			return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
		},
	}
	paystackRepo := &mockPaystackRepo{
		getActiveFn: func(_ context.Context, clientID string) (db.ClientPaystackCredential, error) {
			return db.ClientPaystackCredential{SecretKeyEncrypted: encSecret, IsActive: true}, nil
		},
	}
	api := &mockPaystackAPI{
		initializeFn: func(_ context.Context, _ paystack.InitializeTransactionRequest) (*paystack.InitializeTransactionData, error) {
			return nil, fmt.Errorf("paystack error (status 401): Invalid key")
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, paystackRepo, nil, testEncryptKey,
		func(string) PaystackAPI { return api }, nil)

	_, err := svc.InitiateCharge(context.Background(), InitiatePaystackChargeRequest{
		ClientID: "client-1", AmountCents: 100, Email: "a@b.com",
		Reference: "order-1", IdempotencyKey: "idem-1",
	})
	if err == nil || !strings.Contains(err.Error(), "Invalid key") {
		t.Fatalf("expected provider error, got %v", err)
	}
	if failedID != "pay-1" {
		t.Fatalf("expected payment pay-1 marked failed, got %q", failedID)
	}
}

func TestPaystackHandleWebhook_ChargeSuccess(t *testing.T) {
	encSecret, _ := crypto.Encrypt(testEncryptKey, "sk_test_secret")
	completer := &mockCompleter{}

	paymentRepo := &mockPaymentRepo{
		getPaymentByIDFn: func(_ context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, ClientID: "client-1", AmountCents: 150050, Status: db.PaymentStatusPending}, nil
		},
	}
	paystackRepo := &mockPaystackRepo{
		getActiveFn: func(_ context.Context, _ string) (db.ClientPaystackCredential, error) {
			return db.ClientPaystackCredential{SecretKeyEncrypted: encSecret, IsActive: true}, nil
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, paystackRepo, completer, testEncryptKey, nil, nil)

	body := []byte(`{"event":"charge.success","data":{"id":9001,"reference":"pay-1","amount":150050,"status":"success"}}`)
	if err := svc.HandleWebhook(context.Background(), body, paystackSign("sk_test_secret", body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completer.completedID != "pay-1" {
		t.Fatalf("expected pay-1 completed, got %q", completer.completedID)
	}
}

func TestPaystackHandleWebhook_InvalidSignature(t *testing.T) {
	encSecret, _ := crypto.Encrypt(testEncryptKey, "sk_test_secret")
	completer := &mockCompleter{}

	paymentRepo := &mockPaymentRepo{
		getPaymentByIDFn: func(_ context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, ClientID: "client-1", AmountCents: 150050}, nil
		},
	}
	paystackRepo := &mockPaystackRepo{
		getActiveFn: func(_ context.Context, _ string) (db.ClientPaystackCredential, error) {
			return db.ClientPaystackCredential{SecretKeyEncrypted: encSecret, IsActive: true}, nil
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, paystackRepo, completer, testEncryptKey, nil, nil)

	body := []byte(`{"event":"charge.success","data":{"id":9001,"reference":"pay-1","amount":150050}}`)
	err := svc.HandleWebhook(context.Background(), body, "deadbeef")
	if err == nil || !strings.Contains(err.Error(), "invalid webhook signature") {
		t.Fatalf("expected signature error, got %v", err)
	}
	if completer.completedID != "" {
		t.Fatal("payment must not be completed on invalid signature")
	}
}

func TestPaystackHandleWebhook_AmountMismatch(t *testing.T) {
	encSecret, _ := crypto.Encrypt(testEncryptKey, "sk_test_secret")
	completer := &mockCompleter{}

	paymentRepo := &mockPaymentRepo{
		getPaymentByIDFn: func(_ context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, ClientID: "client-1", AmountCents: 150050, Status: db.PaymentStatusPending}, nil
		},
	}
	paystackRepo := &mockPaystackRepo{
		getActiveFn: func(_ context.Context, _ string) (db.ClientPaystackCredential, error) {
			return db.ClientPaystackCredential{SecretKeyEncrypted: encSecret, IsActive: true}, nil
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, paystackRepo, completer, testEncryptKey, nil, nil)

	body := []byte(`{"event":"charge.success","data":{"id":9001,"reference":"pay-1","amount":99999,"status":"success"}}`)
	err := svc.HandleWebhook(context.Background(), body, paystackSign("sk_test_secret", body))
	if err == nil || !strings.Contains(err.Error(), "amount mismatch") {
		t.Fatalf("expected amount mismatch error, got %v", err)
	}
	if completer.completedID != "" {
		t.Fatal("payment must not be completed on amount mismatch")
	}
}

func TestPaystackHandleWebhook_ChargeFailed(t *testing.T) {
	encSecret, _ := crypto.Encrypt(testEncryptKey, "sk_test_secret")
	completer := &mockCompleter{}

	paymentRepo := &mockPaymentRepo{
		getPaymentByIDFn: func(_ context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, ClientID: "client-1", AmountCents: 150050, Status: db.PaymentStatusPending}, nil
		},
	}
	paystackRepo := &mockPaystackRepo{
		getActiveFn: func(_ context.Context, _ string) (db.ClientPaystackCredential, error) {
			return db.ClientPaystackCredential{SecretKeyEncrypted: encSecret, IsActive: true}, nil
		},
	}
	svc := NewPaystackServiceForTest(paymentRepo, paystackRepo, completer, testEncryptKey, nil, nil)

	body := []byte(`{"event":"charge.failed","data":{"id":9001,"reference":"pay-1","amount":150050,"status":"failed"}}`)
	if err := svc.HandleWebhook(context.Background(), body, paystackSign("sk_test_secret", body)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completer.failedID != "pay-1" {
		t.Fatalf("expected pay-1 failed, got %q", completer.failedID)
	}
}
