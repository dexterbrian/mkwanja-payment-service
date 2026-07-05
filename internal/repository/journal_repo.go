package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	db "mkwanja-payment-svc/internal/db/generated"
)

// JournalRepo defines journal persistence operations.
type JournalRepo interface {
	CreateJournalEntry(ctx context.Context, params db.CreateJournalEntryParams) (db.Journal, error)
	CreateJournalEntryBulk(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error)
	ListJournalEntriesByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error)
	ListJournalEntriesByPayment(ctx context.Context, paymentID string) ([]db.Journal, error)
	GetAccountBalances(ctx context.Context, clientID string) ([]db.AccountBalance, error)
	GetTrialBalance(ctx context.Context, clientID string) ([]db.AccountBalance, error)
}

// PgxJournalRepo implements JournalRepo using *db.Queries directly (needs WithTx for bulk).
type PgxJournalRepo struct {
	queries   *db.Queries
	stdlibDB  *sql.DB
}

// NewPgxJournalRepo creates a repo backed by *db.Queries and *sql.DB.
func NewPgxJournalRepo(q *db.Queries, stdlibDB *sql.DB) *PgxJournalRepo {
	return &PgxJournalRepo{queries: q, stdlibDB: stdlibDB}
}

// NewPgxJournalRepoFromPool creates a repo from a pgxpool.Pool.
func NewPgxJournalRepoFromPool(pool *pgxpool.Pool) *PgxJournalRepo {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	q := db.New(stdlibDB)
	return &PgxJournalRepo{queries: q, stdlibDB: stdlibDB}
}

func (r *PgxJournalRepo) CreateJournalEntry(ctx context.Context, params db.CreateJournalEntryParams) (db.Journal, error) {
	j, err := r.queries.CreateJournalEntry(ctx, params)
	if err != nil {
		return db.Journal{}, fmt.Errorf("create journal entry: %w", err)
	}
	return j, nil
}

func (r *PgxJournalRepo) CreateJournalEntryBulk(ctx context.Context, entries []db.CreateJournalEntryParams) ([]db.Journal, error) {
	tx, err := r.stdlibDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := r.queries.WithTx(tx)
	results := make([]db.Journal, 0, len(entries))
	for _, entry := range entries {
		j, err := qtx.CreateJournalEntry(ctx, entry)
		if err != nil {
			return nil, fmt.Errorf("create journal entry in bulk: %w", err)
		}
		results = append(results, j)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit journal bulk: %w", err)
	}

	return results, nil
}

func (r *PgxJournalRepo) ListJournalEntriesByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Journal, error) {
	entries, err := r.queries.ListJournalEntriesByClient(ctx, db.ListJournalEntriesByClientParams{
		ClientID: clientID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list journal entries by client: %w", err)
	}
	return entries, nil
}

func (r *PgxJournalRepo) ListJournalEntriesByPayment(ctx context.Context, paymentID string) ([]db.Journal, error) {
	entries, err := r.queries.ListJournalEntriesByPayment(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("list journal entries by payment: %w", err)
	}
	return entries, nil
}

func (r *PgxJournalRepo) GetAccountBalances(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	balances, err := r.queries.GetAccountBalances(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get account balances: %w", err)
	}
	return balances, nil
}

func (r *PgxJournalRepo) GetTrialBalance(ctx context.Context, clientID string) ([]db.AccountBalance, error) {
	balances, err := r.queries.GetTrialBalance(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("get trial balance: %w", err)
	}
	return balances, nil
}

// compile-time interface check
var _ JournalRepo = (*PgxJournalRepo)(nil)

// unused import guard
var _ = sql.ErrNoRows
