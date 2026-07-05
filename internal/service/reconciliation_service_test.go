package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"mkwanja-payment-svc/internal/crypto"
	"mkwanja-payment-svc/internal/daraja"
	db "mkwanja-payment-svc/internal/db/generated"
)

// mockRegistry implements dbPgRegistry for testing.
type mockRegistry struct {
	consumers map[string]struct{}
	err       error
}

func (m *mockRegistry) Get(consumerID string) (*pgxpool.Pool, error) {
	return nil, m.err
}

func (m *mockRegistry) Ping(ctx context.Context) map[string]error {
	result := make(map[string]error, len(m.consumers))
	for id := range m.consumers {
		result[id] = nil
	}
	return result
}

func TestReconciliation_RunOnce_NoConsumers(t *testing.T) {
	reg := &mockRegistry{consumers: map[string]struct{}{}}
	svc := NewReconciliationServiceForTest(reg, nil, "", nil, slog.Default())
	err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestReconciliation_RunOnce_NoPendingPayments(t *testing.T) {
	reg := &mockRegistry{
		consumers: map[string]struct{}{
			"test-consumer": {},
		},
		err: fmt.Errorf("no pool available"),
	}
	svc := NewReconciliationServiceForTest(reg, nil, "", nil, slog.Default())
	err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestReconciliation_reconcilePayment_Completed(t *testing.T) {
	encryptKey := make([]byte, 32)
	ek, _ := crypto.Encrypt(encryptKey, "ck")
	es, _ := crypto.Encrypt(encryptKey, "cs")
	ep, _ := crypto.Encrypt(encryptKey, "pk")

	mockPay := &mockPaymentRepo{}
	mockClient := &mockClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{
				ID: "cred-001", Shortcode: "123456",
				ConsumerKeyEncrypted:    ek,
				ConsumerSecretEncrypted: es,
				PasskeyEncrypted:        ep,
				IsActive:                true,
			}, nil
		},
	}
	mockJournal := &mockJournalRepo{}
	mockDC := &mockDarajaClient{
		queryTransactionStatusFn: func(ctx context.Context, req daraja.TransactionStatusRequest) (*daraja.TransactionStatusResponse, error) {
			return &daraja.TransactionStatusResponse{
				ResultCode:        "0",
				ResultDesc:        "Completed",
				ReceiptNo:         "REC-001",
				ConversationID:    "CONV-001",
				TransactionStatus: "completed",
			}, nil
		},
	}
	mockPay.completePaymentFn = func(ctx context.Context, id string, receipt, txID sql.NullString) (db.Payment, error) {
		return db.Payment{ID: id, Status: db.PaymentStatusCompleted}, nil
	}
	mockPay.createPaymentEventFn = func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
		return db.PaymentEvent{}, nil
	}
	mockJournal.createJournalEntryBulkFn = func(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
		result := make([]db.Journal, len(entries))
		for i := range entries {
			result[i] = db.Journal{ID: int64(i + 1)}
		}
		return result, nil
	}

	svc := NewReconciliationServiceForTest(nil, encryptKey, "https://example.com/callback",
		func(_, _, _, _ string) DarajaClient { return mockDC },
		slog.Default(),
	)

	payment := db.Payment{
		ID:                "pay-001",
		ClientID:          "client-1",
		Status:            db.PaymentStatusPending,
		ProviderRequestID: sql.NullString{String: "req-001", Valid: true},
		AmountCents:       10000,
		Currency:          "KES",
		Direction:         db.PaymentDirectionInbound,
		PaymentType:       db.PaymentTypeStkPush,
	}

	err := svc.reconcilePayment(context.Background(), payment, "test-consumer", mockPay, mockClient, mockJournal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReconciliation_reconcilePayment_Failed(t *testing.T) {
	encryptKey := make([]byte, 32)
	ek, _ := crypto.Encrypt(encryptKey, "ck")
	es, _ := crypto.Encrypt(encryptKey, "cs")
	ep, _ := crypto.Encrypt(encryptKey, "pk")

	mockPay := &mockPaymentRepo{}
	mockClient := &mockClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{
				ID: "cred-001", Shortcode: "123456",
				ConsumerKeyEncrypted:    ek,
				ConsumerSecretEncrypted: es,
				PasskeyEncrypted:        ep,
				IsActive:                true,
			}, nil
		},
	}
	mockJournal := &mockJournalRepo{}
	mockDC := &mockDarajaClient{
		queryTransactionStatusFn: func(ctx context.Context, req daraja.TransactionStatusRequest) (*daraja.TransactionStatusResponse, error) {
			return &daraja.TransactionStatusResponse{
				ResultCode:        "1",
				ResultDesc:        "Failed",
				TransactionStatus: "failed",
			}, nil
		},
	}
	mockPay.failPaymentFn = func(ctx context.Context, id string) (db.Payment, error) {
		return db.Payment{ID: id, Status: db.PaymentStatusFailed}, nil
	}
	mockPay.createPaymentEventFn = func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
		return db.PaymentEvent{}, nil
	}

	svc := NewReconciliationServiceForTest(nil, encryptKey, "https://example.com/callback",
		func(_, _, _, _ string) DarajaClient { return mockDC },
		slog.Default(),
	)

	payment := db.Payment{
		ID:                "pay-002",
		ClientID:          "client-1",
		Status:            db.PaymentStatusPending,
		ProviderRequestID: sql.NullString{String: "req-002", Valid: true},
		AmountCents:       5000,
		Currency:          "KES",
		Direction:         db.PaymentDirectionInbound,
		PaymentType:       db.PaymentTypeStkPush,
	}

	err := svc.reconcilePayment(context.Background(), payment, "test-consumer", mockPay, mockClient, mockJournal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReconciliation_reconcilePayment_MissingCredentials(t *testing.T) {
	mockPay := &mockPaymentRepo{}
	mockClient := &mockClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{}, sql.ErrNoRows
		},
	}
	mockJournal := &mockJournalRepo{}
	mockDC := &mockDarajaClient{}

	svc := NewReconciliationServiceForTest(nil, nil, "", func(_, _, _, _ string) DarajaClient { return mockDC }, slog.Default())

	payment := db.Payment{
		ID:       "pay-003",
		ClientID: "client-missing",
		Status:   db.PaymentStatusPending,
	}

	err := svc.reconcilePayment(context.Background(), payment, "test-consumer", mockPay, mockClient, mockJournal)
	if err == nil {
		t.Fatal("expected error for missing credentials, got nil")
	}
	if !strings.Contains(err.Error(), "get credentials") {
		t.Fatalf("expected error containing 'get credentials', got %q", err.Error())
	}
}

func TestReconciliation_CompletePayment_WithJournal(t *testing.T) {
	encryptKey := make([]byte, 32)
	ek, _ := crypto.Encrypt(encryptKey, "ck")
	es, _ := crypto.Encrypt(encryptKey, "cs")
	ep, _ := crypto.Encrypt(encryptKey, "pk")

	var completedID string
	var capturedJournalEntries []db.CreateJournalEntryParams

	mockPay := &mockPaymentRepo{
		completePaymentFn: func(ctx context.Context, id string, receipt, txID sql.NullString) (db.Payment, error) {
			completedID = id
			return db.Payment{
				ID: id, ClientID: "client-1", Status: db.PaymentStatusCompleted,
				AmountCents: 20000, Currency: "KES",
				Direction: db.PaymentDirectionInbound, PaymentType: db.PaymentTypeStkPush,
			}, nil
		},
		createPaymentEventFn: func(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
			return db.PaymentEvent{}, nil
		},
	}
	mockClient := &mockClientRepo{
		getActiveCredentialsFn: func(ctx context.Context, clientID string) (db.ClientCredential, error) {
			return db.ClientCredential{
				ID: "cred-001", Shortcode: "123456",
				ConsumerKeyEncrypted:    ek,
				ConsumerSecretEncrypted: es,
				PasskeyEncrypted:        ep,
				IsActive:                true,
			}, nil
		},
	}
	mockJournal := &mockJournalRepo{
		createJournalEntryBulkFn: func(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
			capturedJournalEntries = entries
			result := make([]db.Journal, len(entries))
			for i := range entries {
				result[i] = db.Journal{ID: int64(i + 1)}
			}
			return result, nil
		},
	}
	mockDC := &mockDarajaClient{
		queryTransactionStatusFn: func(ctx context.Context, req daraja.TransactionStatusRequest) (*daraja.TransactionStatusResponse, error) {
			return &daraja.TransactionStatusResponse{
				ResultCode:        "0",
				ReceiptNo:         "REC-JRN-001",
				ConversationID:    "CONV-JRN-001",
				TransactionStatus: "completed",
			}, nil
		},
	}

	svc := NewReconciliationServiceForTest(nil, encryptKey, "https://example.com/callback",
		func(_, _, _, _ string) DarajaClient { return mockDC },
		slog.Default(),
	)

	payment := db.Payment{
		ID:                "pay-jrn-001",
		ClientID:          "client-1",
		Status:            db.PaymentStatusPending,
		ProviderRequestID: sql.NullString{String: "req-jrn-001", Valid: true},
		AmountCents:       20000,
		Currency:          "KES",
		Direction:         db.PaymentDirectionInbound,
		PaymentType:       db.PaymentTypeStkPush,
	}

	err := svc.reconcilePayment(context.Background(), payment, "test-consumer", mockPay, mockClient, mockJournal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completedID != "pay-jrn-001" {
		t.Fatalf("expected payment pay-jrn-001 to be completed, got %s", completedID)
	}
	if len(capturedJournalEntries) != 2 {
		t.Fatalf("expected 2 journal entries, got %d", len(capturedJournalEntries))
	}
	var totalDebits, totalCredits int64
	for _, e := range capturedJournalEntries {
		if e.EntryType == db.EntryTypeDebit {
			totalDebits += e.AmountCents
		}
		if e.EntryType == db.EntryTypeCredit {
			totalCredits += e.AmountCents
		}
	}
	if totalDebits != totalCredits {
		t.Fatalf("journal entries not balanced: debits=%d credits=%d", totalDebits, totalCredits)
	}
}
