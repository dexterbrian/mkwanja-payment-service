package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/crypto"
	"mkwanja-payment-svc/internal/daraja"
	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
	"mkwanja-payment-svc/internal/service"
)

type stubPaymentRepo struct {
	createPaymentFn                func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error)
	getPaymentByIDFn               func(ctx context.Context, id string) (db.Payment, error)
	getPaymentByIdempotencyKeyFn   func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error)
	updatePaymentStatusFn          func(ctx context.Context, id string, status db.PaymentStatus) (db.Payment, error)
	completePaymentFn              func(ctx context.Context, id string, providerReceipt, providerTxID sql.NullString) (db.Payment, error)
	failPaymentFn                  func(ctx context.Context, id string) (db.Payment, error)
	updateProviderRequestIDFn      func(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error)
	listPaymentsByClientFn         func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error)
	listPendingPaymentsOlderThanFn func(ctx context.Context, createdAt time.Time) ([]db.Payment, error)
	createPaymentEventFn           func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error)
}

func (s *stubPaymentRepo) CreatePayment(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
	return s.createPaymentFn(ctx, params)
}
func (s *stubPaymentRepo) GetPaymentByID(ctx context.Context, id string) (db.Payment, error) {
	return s.getPaymentByIDFn(ctx, id)
}
func (s *stubPaymentRepo) GetPaymentByIdempotencyKey(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
	return s.getPaymentByIdempotencyKeyFn(ctx, clientID, idempotencyKey)
}
func (s *stubPaymentRepo) UpdatePaymentStatus(ctx context.Context, id string, status db.PaymentStatus) (db.Payment, error) {
	return s.updatePaymentStatusFn(ctx, id, status)
}
func (s *stubPaymentRepo) CompletePayment(ctx context.Context, id string, providerReceipt, providerTxID sql.NullString) (db.Payment, error) {
	return s.completePaymentFn(ctx, id, providerReceipt, providerTxID)
}
func (s *stubPaymentRepo) FailPayment(ctx context.Context, id string) (db.Payment, error) {
	return s.failPaymentFn(ctx, id)
}
func (s *stubPaymentRepo) UpdateProviderRequestID(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error) {
	return s.updateProviderRequestIDFn(ctx, id, providerRequestID, providerTxID)
}
func (s *stubPaymentRepo) ListPaymentsByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
	return s.listPaymentsByClientFn(ctx, clientID, limit, offset)
}
func (s *stubPaymentRepo) ListPendingPaymentsOlderThan(ctx context.Context, createdAt time.Time) ([]db.Payment, error) {
	return s.listPendingPaymentsOlderThanFn(ctx, createdAt)
}
func (s *stubPaymentRepo) CreatePaymentEvent(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
	return s.createPaymentEventFn(ctx, params)
}

type stubClientRepo struct {
	createClientFn          func(ctx context.Context, params db.CreateClientParams) (db.Client, error)
	getClientByIDFn         func(ctx context.Context, id string) (db.Client, error)
	getClientByExternalIDFn func(ctx context.Context, externalID string) (db.Client, error)
	listClientsFn           func(ctx context.Context) ([]db.Client, error)
	deactivateClientFn      func(ctx context.Context, id string) (db.Client, error)
	createCredentialsFn     func(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error)
	getActiveCredentialsFn  func(ctx context.Context, clientID string) (db.ClientCredential, error)
	deactivateCredentialsFn func(ctx context.Context, clientID string) error
	createJournalAccountFn  func(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error)
	listJournalAccountsFn   func(ctx context.Context, clientID string) ([]db.JournalAccount, error)
}

func (s *stubClientRepo) CreateClient(ctx context.Context, params db.CreateClientParams) (db.Client, error) {
	return s.createClientFn(ctx, params)
}
func (s *stubClientRepo) GetClientByID(ctx context.Context, id string) (db.Client, error) {
	return s.getClientByIDFn(ctx, id)
}
func (s *stubClientRepo) GetClientByExternalID(ctx context.Context, externalID string) (db.Client, error) {
	return s.getClientByExternalIDFn(ctx, externalID)
}
func (s *stubClientRepo) ListClients(ctx context.Context) ([]db.Client, error) {
	return s.listClientsFn(ctx)
}
func (s *stubClientRepo) DeactivateClient(ctx context.Context, id string) (db.Client, error) {
	return s.deactivateClientFn(ctx, id)
}
func (s *stubClientRepo) CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error) {
	return s.createCredentialsFn(ctx, params)
}
func (s *stubClientRepo) GetActiveCredentials(ctx context.Context, clientID string) (db.ClientCredential, error) {
	return s.getActiveCredentialsFn(ctx, clientID)
}
func (s *stubClientRepo) DeactivateCredentials(ctx context.Context, clientID string) error {
	return s.deactivateCredentialsFn(ctx, clientID)
}
func (s *stubClientRepo) CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
	return s.createJournalAccountFn(ctx, params)
}
func (s *stubClientRepo) ListJournalAccounts(ctx context.Context, clientID string) ([]db.JournalAccount, error) {
	return s.listJournalAccountsFn(ctx, clientID)
}

type stubDarajaClient struct {
	initiateSTKPushFn func(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error)
	initiateB2CFn     func(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error)
	initiateB2BFn     func(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error)
}

func (s *stubDarajaClient) InitiateSTKPush(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error) {
	return s.initiateSTKPushFn(ctx, req)
}
func (s *stubDarajaClient) InitiateB2C(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error) {
	return s.initiateB2CFn(ctx, req)
}
func (s *stubDarajaClient) InitiateB2B(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error) {
	return s.initiateB2BFn(ctx, req)
}

func newTestPaymentHandler(payRepo *stubPaymentRepo, clientRepo *stubClientRepo, dc *stubDarajaClient, journalRepo repository.JournalRepo) *PaymentHandler {
	encryptKey := make([]byte, 32)
	svc := service.NewPaymentServiceForTest(payRepo, clientRepo, journalRepo, encryptKey, "https://example.com/callback", nil,
		func(_, _, _, _ string) service.DarajaClient {
			return dc
		}, nil)
	return NewPaymentHandler(svc, nil)
}

func setupFiberTest(handler *PaymentHandler) *fiber.App {
	app := fiber.New()
	api := app.Group("/v1", func(c *fiber.Ctx) error {
		c.Locals("consumer_id", "test-consumer")
		c.Locals("idempotency_key", "test-uuid-key")
		return c.Next()
	})
	api.Post("/payments/stk-push", handler.InitiateSTKPush)
	api.Post("/payments/b2c", handler.InitiateB2C)
	api.Post("/payments/b2b", handler.InitiateB2B)
	api.Get("/payments/:id", handler.GetPayment)
	api.Get("/payments", handler.ListPayments)
	return app
}

func TestPaymentHandler_InitiateSTKPush_InvalidBody(t *testing.T) {
	handler := newTestPaymentHandler(&stubPaymentRepo{}, &stubClientRepo{}, &stubDarajaClient{}, nil)
	app := setupFiberTest(handler)

	req := httptest.NewRequest("POST", "/v1/payments/stk-push", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPaymentHandler_InitiateSTKPush_Success(t *testing.T) {
	encKey := make([]byte, 32)
	ek, _ := crypto.Encrypt(encKey, "")
	es, _ := crypto.Encrypt(encKey, "")
	ep, _ := crypto.Encrypt(encKey, "")

	payRepo := &stubPaymentRepo{
		createPaymentFn: func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
			return db.Payment{ID: "payment-001", Status: db.PaymentStatusPending}, nil
		},
		getPaymentByIdempotencyKeyFn: func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
			return db.Payment{}, sql.ErrNoRows
		},
		updateProviderRequestIDFn: func(ctx context.Context, id string, reqID, txID sql.NullString) (db.Payment, error) {
			return db.Payment{}, nil
		},
		failPaymentFn: func(ctx context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
		},
	}
	clientRepo := &stubClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{
				ID: "cred-001", Shortcode: "123456",
				ConsumerKeyEncrypted:    ek,
				ConsumerSecretEncrypted: es,
				PasskeyEncrypted:        ep,
				IsActive: true,
			}, nil
		},
	}
	dc := &stubDarajaClient{
		initiateSTKPushFn: func(ctx context.Context, req daraja.STKPushRequest) (*daraja.STKPushResponse, error) {
			return &daraja.STKPushResponse{
				CheckoutRequestID: "checkout-001",
				ResponseCode:      "0",
			}, nil
		},
	}

	handler := newTestPaymentHandler(payRepo, clientRepo, dc, nil)
	app := setupFiberTest(handler)

	body := `{"client_id":"client-1","amount_cents":10000,"phone_number":"254712345678","reference":"ref-001"}`
	req := httptest.NewRequest("POST", "/v1/payments/stk-push", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["payment_id"] != "payment-001" {
		t.Fatalf("expected payment_id payment-001, got %v", result["payment_id"])
	}
}

func TestPaymentHandler_InitiateSTKPush_Idempotency(t *testing.T) {
	payRepo := &stubPaymentRepo{
		getPaymentByIdempotencyKeyFn: func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
			return db.Payment{ID: "existing-payment", Status: db.PaymentStatusCompleted}, nil
		},
	}
	handler := newTestPaymentHandler(payRepo, &stubClientRepo{}, &stubDarajaClient{}, nil)
	app := setupFiberTest(handler)

	body := `{"client_id":"client-1","amount_cents":10000,"phone_number":"254712345678","reference":"ref-001"}`
	req := httptest.NewRequest("POST", "/v1/payments/stk-push", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["payment_id"] != "existing-payment" {
		t.Fatalf("expected payment_id existing-payment, got %v", result["payment_id"])
	}
}

func TestPaymentHandler_InitiateB2C_Success(t *testing.T) {
	encKey := make([]byte, 32)
	ek, _ := crypto.Encrypt(encKey, "")
	es, _ := crypto.Encrypt(encKey, "")
	ep, _ := crypto.Encrypt(encKey, "")

	payRepo := &stubPaymentRepo{
		createPaymentFn: func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
			return db.Payment{ID: "payment-b2c-001", Status: db.PaymentStatusPending}, nil
		},
		getPaymentByIdempotencyKeyFn: func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
			return db.Payment{}, sql.ErrNoRows
		},
		updateProviderRequestIDFn: func(ctx context.Context, id string, reqID, txID sql.NullString) (db.Payment, error) {
			return db.Payment{}, nil
		},
		failPaymentFn: func(ctx context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
		},
	}
	clientRepo := &stubClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{
				ID: "cred-001", Shortcode: "123456",
				ConsumerKeyEncrypted:    ek,
				ConsumerSecretEncrypted: es,
				PasskeyEncrypted:        ep,
				InitiatorName:           sql.NullString{String: "apitest", Valid: true},
				IsActive:                true,
			}, nil
		},
	}
	dc := &stubDarajaClient{
		initiateB2CFn: func(ctx context.Context, req daraja.B2CRequest) (*daraja.B2CResponse, error) {
			return &daraja.B2CResponse{
				ConversationID: "conv-001",
				ResponseCode:   "0",
			}, nil
		},
	}

	handler := newTestPaymentHandler(payRepo, clientRepo, dc, nil)
	app := setupFiberTest(handler)

	body := `{"client_id":"client-1","amount_cents":5000,"phone_number":"254712345678","reference":"ref-001"}`
	req := httptest.NewRequest("POST", "/v1/payments/b2c", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["payment_id"] != "payment-b2c-001" {
		t.Fatalf("expected payment_id payment-b2c-001, got %v", result["payment_id"])
	}
}

func TestPaymentHandler_InitiateB2B_Success(t *testing.T) {
	encKey := make([]byte, 32)
	ek, _ := crypto.Encrypt(encKey, "")
	es, _ := crypto.Encrypt(encKey, "")
	ep, _ := crypto.Encrypt(encKey, "")

	payRepo := &stubPaymentRepo{
		createPaymentFn: func(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
			return db.Payment{ID: "payment-b2b-001", Status: db.PaymentStatusPending}, nil
		},
		getPaymentByIdempotencyKeyFn: func(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
			return db.Payment{}, sql.ErrNoRows
		},
		updateProviderRequestIDFn: func(ctx context.Context, id string, reqID, txID sql.NullString) (db.Payment, error) {
			return db.Payment{}, nil
		},
		failPaymentFn: func(ctx context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
		},
	}
	clientRepo := &stubClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{
				ID: "cred-001", Shortcode: "123456",
				ConsumerKeyEncrypted:    ek,
				ConsumerSecretEncrypted: es,
				PasskeyEncrypted:        ep,
				InitiatorName:           sql.NullString{String: "apitest", Valid: true},
				IsActive:                true,
			}, nil
		},
	}
	dc := &stubDarajaClient{
		initiateB2BFn: func(ctx context.Context, req daraja.B2BRequest) (*daraja.B2BResponse, error) {
			return &daraja.B2BResponse{
				ConversationID: "conv-b2b-001",
				ResponseCode:   "0",
			}, nil
		},
	}

	handler := newTestPaymentHandler(payRepo, clientRepo, dc, nil)
	app := setupFiberTest(handler)

	body := `{"client_id":"client-1","amount_cents":10000,"receiver_shortcode":"654321","reference":"ref-001"}`
	req := httptest.NewRequest("POST", "/v1/payments/b2b", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["payment_id"] != "payment-b2b-001" {
		t.Fatalf("expected payment_id payment-b2b-001, got %v", result["payment_id"])
	}
}

func TestPaymentHandler_GetPayment(t *testing.T) {
	payRepo := &stubPaymentRepo{
		getPaymentByIDFn: func(ctx context.Context, id string) (db.Payment, error) {
			return db.Payment{ID: id, Status: db.PaymentStatusCompleted}, nil
		},
	}
	handler := newTestPaymentHandler(payRepo, &stubClientRepo{}, &stubDarajaClient{}, nil)
	app := setupFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/payments/payment-001", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["id"] != "payment-001" {
		t.Fatalf("expected id payment-001, got %v", result["id"])
	}
}

func TestPaymentHandler_ListPayments(t *testing.T) {
	payRepo := &stubPaymentRepo{
		listPaymentsByClientFn: func(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
			return []db.Payment{
				{ID: "p1", Status: db.PaymentStatusCompleted},
				{ID: "p2", Status: db.PaymentStatusPending},
			}, nil
		},
	}
	handler := newTestPaymentHandler(payRepo, &stubClientRepo{}, &stubDarajaClient{}, nil)
	app := setupFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/payments?client_id=client-1&limit=10&offset=0", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	payments, ok := result["payments"].([]any)
	if !ok {
		t.Fatalf("expected payments array, got %T", result["payments"])
	}
	if len(payments) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(payments))
	}
}
