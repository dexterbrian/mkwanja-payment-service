package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/service"
)

// stubJournalRepo is a test double for repository.JournalRepo (used only via JournalService).
type stubJournalRepo struct {
	listJournalEntriesByClientFn func(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error)
	getAccountBalancesFn         func(ctx context.Context, clientID string) ([]db.AccountBalance, error)
	getTrialBalanceFn            func(ctx context.Context, clientID string) ([]db.AccountBalance, error)
}

func (s *stubJournalRepo) CreateJournalEntry(ctx context.Context, params db.CreateJournalEntryParams) (db.Journal, error) {
	return db.Journal{}, nil
}
func (s *stubJournalRepo) CreateJournalEntryBulk(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
	return nil, nil
}
func (s *stubJournalRepo) ListJournalEntriesByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error) {
	return s.listJournalEntriesByClientFn(ctx, clientID, limit, offset)
}
func (s *stubJournalRepo) ListJournalEntriesByPayment(ctx context.Context, paymentID string) ([]db.Journal, error) {
	return nil, nil
}
func (s *stubJournalRepo) GetAccountBalances(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	return s.getAccountBalancesFn(ctx, clientID)
}
func (s *stubJournalRepo) GetTrialBalance(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	return s.getTrialBalanceFn(ctx, clientID)
}

func setupLedgerFiberTest(handler *LedgerHandler) *fiber.App {
	app := fiber.New()
	api := app.Group("/v1")
	api.Get("/ledger", handler.ListEntries)
	api.Get("/ledger/balance", handler.GetBalances)
	api.Get("/ledger/trial-balance", handler.GetTrialBalance)
	return app
}

func newTestLedgerHandler(journalRepo *stubJournalRepo) *LedgerHandler {
	svc := service.NewJournalService(journalRepo, nil)
	return NewLedgerHandler(svc, nil)
}

func TestLedgerHandler_ListEntries_Success(t *testing.T) {
	repo := &stubJournalRepo{
		listJournalEntriesByClientFn: func(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error) {
			return []db.Journal{
				{ID: 1, ClientID: clientID, PaymentID: "pay-001", AccountID: "mpesa.till", EntryType: db.EntryTypeDebit, AmountCents: 50000, Currency: "KES", Description: "test"},
				{ID: 2, ClientID: clientID, PaymentID: "pay-001", AccountID: "revenue.sales", EntryType: db.EntryTypeCredit, AmountCents: 50000, Currency: "KES", Description: "test"},
			}, nil
		},
	}
	handler := newTestLedgerHandler(repo)
	app := setupLedgerFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/ledger?client_id=client-1", nil)
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
	entries, ok := result["entries"].([]any)
	if !ok {
		t.Fatalf("expected entries array, got %T", result["entries"])
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestLedgerHandler_ListEntries_MissingClientID(t *testing.T) {
	handler := newTestLedgerHandler(&stubJournalRepo{})
	app := setupLedgerFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/ledger", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLedgerHandler_GetBalances_Success(t *testing.T) {
	repo := &stubJournalRepo{
		getAccountBalancesFn: func(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
			return []db.AccountBalance{
				{AccountID: "mpesa.till", TotalDebitsCents: 50000, TotalCreditsCents: 0, NetCents: 50000},
				{AccountID: "revenue.sales", TotalDebitsCents: 0, TotalCreditsCents: 50000, NetCents: -50000},
			}, nil
		},
	}
	handler := newTestLedgerHandler(repo)
	app := setupLedgerFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/ledger/balance?client_id=client-1", nil)
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
	balances, ok := result["balances"].([]any)
	if !ok {
		t.Fatalf("expected balances array, got %T", result["balances"])
	}
	if len(balances) != 2 {
		t.Fatalf("expected 2 balances, got %d", len(balances))
	}
}

func TestLedgerHandler_GetBalances_MissingClientID(t *testing.T) {
	handler := newTestLedgerHandler(&stubJournalRepo{})
	app := setupLedgerFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/ledger/balance", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLedgerHandler_GetTrialBalance_Success(t *testing.T) {
	repo := &stubJournalRepo{
		getTrialBalanceFn: func(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
			return []db.AccountBalance{
				{AccountID: "mpesa.till", TotalDebitsCents: 50000, TotalCreditsCents: 0, NetCents: 50000},
				{AccountID: "revenue.sales", TotalDebitsCents: 0, TotalCreditsCents: 50000, NetCents: -50000},
			}, nil
		},
	}
	handler := newTestLedgerHandler(repo)
	app := setupLedgerFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/ledger/trial-balance?client_id=client-1", nil)
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
	tb, ok := result["trial_balance"].([]any)
	if !ok {
		t.Fatalf("expected trial_balance array, got %T", result["trial_balance"])
	}
	if len(tb) != 2 {
		t.Fatalf("expected 2 trial balance entries, got %d", len(tb))
	}
}

func TestLedgerHandler_GetTrialBalance_MissingClientID(t *testing.T) {
	handler := newTestLedgerHandler(&stubJournalRepo{})
	app := setupLedgerFiberTest(handler)

	req := httptest.NewRequest("GET", "/v1/ledger/trial-balance", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
