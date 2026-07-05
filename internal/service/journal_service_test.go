package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// mockJournalRepo is a test double for repository.JournalRepo.
type mockJournalRepo struct {
	createJournalEntryFn         func(ctx context.Context, params db.CreateJournalEntryParams) (db.Journal, error)
	createJournalEntryBulkFn     func(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error)
	listJournalEntriesByClientFn func(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error)
	listJournalEntriesByPaymentFn func(ctx context.Context, paymentID string) ([]db.Journal, error)
	getAccountBalancesFn         func(ctx context.Context, clientID string) ([]db.AccountBalance, error)
	getTrialBalanceFn            func(ctx context.Context, clientID string) ([]db.AccountBalance, error)
}

func (m *mockJournalRepo) CreateJournalEntry(ctx context.Context, params db.CreateJournalEntryParams) (db.Journal, error) {
	return m.createJournalEntryFn(ctx, params)
}
func (m *mockJournalRepo) CreateJournalEntryBulk(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
	return m.createJournalEntryBulkFn(ctx, entries)
}
func (m *mockJournalRepo) ListJournalEntriesByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error) {
	return m.listJournalEntriesByClientFn(ctx, clientID, limit, offset)
}
func (m *mockJournalRepo) ListJournalEntriesByPayment(ctx context.Context, paymentID string) ([]db.Journal, error) {
	return m.listJournalEntriesByPaymentFn(ctx, paymentID)
}
func (m *mockJournalRepo) GetAccountBalances(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	return m.getAccountBalancesFn(ctx, clientID)
}
func (m *mockJournalRepo) GetTrialBalance(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	return m.getTrialBalanceFn(ctx, clientID)
}

var _ repository.JournalRepo = (*mockJournalRepo)(nil)

func TestJournalService_WriteInboundEntries_Success(t *testing.T) {
	var captured []db.CreateJournalEntryParams
	repo := &mockJournalRepo{
		createJournalEntryBulkFn: func(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
			captured = entries
			result := make([]db.Journal, len(entries))
			for i := range entries {
				result[i] = db.Journal{ID: int64(i + 1)}
			}
			return result, nil
		},
	}

	svc := NewJournalService(repo, nil)
	payment := db.Payment{
		ID:          "pay-001",
		ClientID:    "client-1",
		AmountCents: 50000,
		Currency:    "KES",
		PaymentType: db.PaymentTypeStkPush,
		Direction:   db.PaymentDirectionInbound,
	}

	err := svc.WriteInboundEntries(context.Background(), payment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(captured) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(captured))
	}

	if captured[0].AccountID != "mpesa.till" {
		t.Fatalf("expected first entry account mpesa.till, got %s", captured[0].AccountID)
	}
	if captured[0].EntryType != db.EntryTypeDebit {
		t.Fatalf("expected first entry type debit, got %s", captured[0].EntryType)
	}
	if captured[0].AmountCents != 50000 {
		t.Fatalf("expected amount_cents 50000, got %d", captured[0].AmountCents)
	}

	if captured[1].AccountID != "revenue.sales" {
		t.Fatalf("expected second entry account revenue.sales, got %s", captured[1].AccountID)
	}
	if captured[1].EntryType != db.EntryTypeCredit {
		t.Fatalf("expected second entry type credit, got %s", captured[1].EntryType)
	}
	if captured[1].AmountCents != 50000 {
		t.Fatalf("expected amount_cents 50000, got %d", captured[1].AmountCents)
	}

	if captured[0].ClientID != "client-1" {
		t.Fatalf("expected client_id client-1, got %s", captured[0].ClientID)
	}
	if captured[0].PaymentID != "pay-001" {
		t.Fatalf("expected payment_id pay-001, got %s", captured[0].PaymentID)
	}
}

func TestJournalService_WriteOutboundEntries_Success(t *testing.T) {
	var captured []db.CreateJournalEntryParams
	repo := &mockJournalRepo{
		createJournalEntryBulkFn: func(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
			captured = entries
			result := make([]db.Journal, len(entries))
			for i := range entries {
				result[i] = db.Journal{ID: int64(i + 1)}
			}
			return result, nil
		},
	}

	svc := NewJournalService(repo, nil)
	payment := db.Payment{
		ID:          "pay-002",
		ClientID:    "client-1",
		AmountCents: 25000,
		Currency:    "KES",
		PaymentType: db.PaymentTypeB2c,
		Direction:   db.PaymentDirectionOutbound,
	}

	err := svc.WriteOutboundEntries(context.Background(), payment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(captured) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(captured))
	}

	if captured[0].AccountID != "expense.operations" {
		t.Fatalf("expected first entry account expense.operations, got %s", captured[0].AccountID)
	}
	if captured[0].EntryType != db.EntryTypeDebit {
		t.Fatalf("expected first entry type debit, got %s", captured[0].EntryType)
	}
	if captured[0].AmountCents != 25000 {
		t.Fatalf("expected amount_cents 25000, got %d", captured[0].AmountCents)
	}

	if captured[1].AccountID != "mpesa.till" {
		t.Fatalf("expected second entry account mpesa.till, got %s", captured[1].AccountID)
	}
	if captured[1].EntryType != db.EntryTypeCredit {
		t.Fatalf("expected second entry type credit, got %s", captured[1].EntryType)
	}
	if captured[1].AmountCents != 25000 {
		t.Fatalf("expected amount_cents 25000, got %d", captured[1].AmountCents)
	}
}

func TestJournalService_ListJournalEntriesByClient(t *testing.T) {
	expected := []db.Journal{
		{ID: 1, ClientID: "client-1", AmountCents: 50000},
		{ID: 2, ClientID: "client-1", AmountCents: 25000},
	}
	repo := &mockJournalRepo{
		listJournalEntriesByClientFn: func(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error) {
			return expected, nil
		},
	}

	svc := NewJournalService(repo, nil)
	entries, err := svc.ListJournalEntriesByClient(context.Background(), "client-1", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestJournalService_GetAccountBalances(t *testing.T) {
	expected := []db.AccountBalance{
		{AccountID: "mpesa.till", TotalDebitsCents: 50000, TotalCreditsCents: 0, NetCents: 50000},
	}
	repo := &mockJournalRepo{
		getAccountBalancesFn: func(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
			return expected, nil
		},
	}

	svc := NewJournalService(repo, nil)
	balances, err := svc.GetAccountBalances(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(balances))
	}
	if balances[0].AccountID != "mpesa.till" {
		t.Fatalf("expected mpesa.till, got %s", balances[0].AccountID)
	}
}

func TestJournalService_GetTrialBalance(t *testing.T) {
	expected := []db.AccountBalance{
		{AccountID: "mpesa.till", TotalDebitsCents: 50000, TotalCreditsCents: 0, NetCents: 50000},
		{AccountID: "revenue.sales", TotalDebitsCents: 0, TotalCreditsCents: 50000, NetCents: -50000},
	}
	repo := &mockJournalRepo{
		getTrialBalanceFn: func(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
			return expected, nil
		},
	}

	svc := NewJournalService(repo, nil)
	balances, err := svc.GetTrialBalance(context.Background(), "client-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 2 {
		t.Fatalf("expected 2 trial balance entries, got %d", len(balances))
	}
}

func TestJournalService_WriteInboundEntries_RepoError(t *testing.T) {
	repo := &mockJournalRepo{
		createJournalEntryBulkFn: func(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
			return nil, sql.ErrConnDone
		},
	}

	svc := NewJournalService(repo, nil)
	payment := db.Payment{
		ID:          "pay-003",
		ClientID:    "client-1",
		AmountCents: 10000,
		Currency:    "KES",
		PaymentType: db.PaymentTypeStkPush,
		Direction:   db.PaymentDirectionInbound,
	}

	err := svc.WriteInboundEntries(context.Background(), payment)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write journal entries") {
		t.Fatalf("expected error containing 'write journal entries', got %q", err.Error())
	}
}
