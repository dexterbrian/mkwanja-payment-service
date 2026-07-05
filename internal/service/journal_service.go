package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/domain"
	"mkwanja-payment-svc/internal/repository"
)

// JournalService handles double-entry ledger writes and reads.
type JournalService struct {
	journalRepo repository.JournalRepo
	logger      *slog.Logger
}

// NewJournalService creates a JournalService.
func NewJournalService(journalRepo repository.JournalRepo, logger *slog.Logger) *JournalService {
	if logger == nil {
		logger = slog.Default()
	}
	return &JournalService{journalRepo: journalRepo, logger: logger}
}

// WriteInboundEntries writes balanced journal entries for a confirmed inbound payment
// (STK Push / C2B). Debits mpesa.till, credits revenue.sales.
func (s *JournalService) WriteInboundEntries(ctx context.Context, payment db.Payment) error {
	entries := []db.CreateJournalEntryParams{
		{
			ClientID:    payment.ClientID,
			PaymentID:   payment.ID,
			AccountID:   "mpesa.till",
			EntryType:   db.EntryTypeDebit,
			AmountCents: payment.AmountCents,
			Currency:    payment.Currency,
			Description: "Inbound payment via " + string(payment.PaymentType),
			ReversalOf:  sql.NullInt64{Valid: false},
		},
		{
			ClientID:    payment.ClientID,
			PaymentID:   payment.ID,
			AccountID:   "revenue.sales",
			EntryType:   db.EntryTypeCredit,
			AmountCents: payment.AmountCents,
			Currency:    payment.Currency,
			Description: "Sales revenue for payment " + payment.ID,
			ReversalOf:  sql.NullInt64{Valid: false},
		},
	}
	return s.writeBalancedEntries(ctx, entries)
}

// WriteOutboundEntries writes balanced journal entries for a confirmed outbound payment
// (B2C / B2B). Debits expense.operations, credits mpesa.till.
func (s *JournalService) WriteOutboundEntries(ctx context.Context, payment db.Payment) error {
	entries := []db.CreateJournalEntryParams{
		{
			ClientID:    payment.ClientID,
			PaymentID:   payment.ID,
			AccountID:   "expense.operations",
			EntryType:   db.EntryTypeDebit,
			AmountCents: payment.AmountCents,
			Currency:    payment.Currency,
			Description: "Outbound payment via " + string(payment.PaymentType),
			ReversalOf:  sql.NullInt64{Valid: false},
		},
		{
			ClientID:    payment.ClientID,
			PaymentID:   payment.ID,
			AccountID:   "mpesa.till",
			EntryType:   db.EntryTypeCredit,
			AmountCents: payment.AmountCents,
			Currency:    payment.Currency,
			Description: "Till outbound for payment " + payment.ID,
			ReversalOf:  sql.NullInt64{Valid: false},
		},
	}
	return s.writeBalancedEntries(ctx, entries)
}

// writeBalancedEntries verifies that debit and credit totals are equal, then writes all entries in a single transaction.
func (s *JournalService) writeBalancedEntries(ctx context.Context, params []db.CreateJournalEntryParams) error {
	domainEntries := make([]domain.JournalEntry, 0, len(params))
	for _, p := range params {
		domainEntries = append(domainEntries, domain.JournalEntry{
			ClientID:    p.ClientID,
			PaymentID:   p.PaymentID,
			AccountID:   p.AccountID,
			EntryType:   domain.EntryType(p.EntryType),
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			Description: p.Description,
		})
	}

	if err := domain.VerifyBalance(domainEntries); err != nil {
		return fmt.Errorf("journal balance check failed: %w", err)
	}

	_, err := s.journalRepo.CreateJournalEntryBulk(ctx, params)
	if err != nil {
		return fmt.Errorf("write journal entries: %w", err)
	}

	s.logger.Info("journal entries written",
		"client_id", params[0].ClientID,
		"payment_id", params[0].PaymentID,
		"count", len(params))

	return nil
}

// GetAccountBalances returns account balances for a client.
func (s *JournalService) GetAccountBalances(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	return s.journalRepo.GetAccountBalances(ctx, clientID)
}

// GetTrialBalance returns the trial balance for a client.
func (s *JournalService) GetTrialBalance(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	return s.journalRepo.GetTrialBalance(ctx, clientID)
}

// ListJournalEntriesByClient lists journal entries for a client with pagination.
func (s *JournalService) ListJournalEntriesByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error) {
	return s.journalRepo.ListJournalEntriesByClient(ctx, clientID, limit, offset)
}
