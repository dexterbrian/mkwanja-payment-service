package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	db "mkwanja-payment-svc/internal/db/generated"
)

// BusinessRepo defines business and credential persistence operations.
type BusinessRepo interface {
	CreateBusiness(ctx context.Context, params db.CreateBusinessParams) (db.Business, error)
	GetBusinessByID(ctx context.Context, id string) (db.Business, error)
	GetBusinessByExternalID(ctx context.Context, externalID string) (db.Business, error)
	ListBusinesses(ctx context.Context) ([]db.Business, error)
	DeactivateBusiness(ctx context.Context, id string) (db.Business, error)

	CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.BusinessCredential, error)
	GetActiveCredentials(ctx context.Context, businessID string) (db.BusinessCredential, error)
	DeactivateCredentials(ctx context.Context, businessID string) error

	CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error)
	ListJournalAccounts(ctx context.Context, businessID string) ([]db.JournalAccount, error)
}

// PgxBusinessRepo implements BusinessRepo using a db.Querier.
type PgxBusinessRepo struct {
	q db.Querier
}

// NewPgxBusinessRepo creates a repo backed by a db.Querier.
func NewPgxBusinessRepo(q db.Querier) *PgxBusinessRepo {
	return &PgxBusinessRepo{q: q}
}

// NewPgxBusinessRepoFromPool creates a repo from a pgxpool.Pool by opening
// a stdlib connection that satisfies the database/sql interface expected by sqlc.
func NewPgxBusinessRepoFromPool(pool *pgxpool.Pool) *PgxBusinessRepo {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	return &PgxBusinessRepo{q: db.New(stdlibDB)}
}

func (r *PgxBusinessRepo) CreateBusiness(ctx context.Context, params db.CreateBusinessParams) (db.Business, error) {
	b, err := r.q.CreateBusiness(ctx, params)
	if err != nil {
		return db.Business{}, fmt.Errorf("create business: %w", err)
	}
	return b, nil
}

func (r *PgxBusinessRepo) GetBusinessByID(ctx context.Context, id string) (db.Business, error) {
	b, err := r.q.GetBusinessByID(ctx, id)
	if err != nil {
		return db.Business{}, fmt.Errorf("get business by id: %w", err)
	}
	return b, nil
}

func (r *PgxBusinessRepo) GetBusinessByExternalID(ctx context.Context, externalID string) (db.Business, error) {
	b, err := r.q.GetBusinessByExternalID(ctx, externalID)
	if err != nil {
		return db.Business{}, fmt.Errorf("get business by external id: %w", err)
	}
	return b, nil
}

func (r *PgxBusinessRepo) ListBusinesses(ctx context.Context) ([]db.Business, error) {
	businesses, err := r.q.ListBusinesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list businesses: %w", err)
	}
	return businesses, nil
}

func (r *PgxBusinessRepo) DeactivateBusiness(ctx context.Context, id string) (db.Business, error) {
	b, err := r.q.DeactivateBusiness(ctx, id)
	if err != nil {
		return db.Business{}, fmt.Errorf("deactivate business: %w", err)
	}
	return b, nil
}

func (r *PgxBusinessRepo) CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.BusinessCredential, error) {
	c, err := r.q.CreateCredentials(ctx, params)
	if err != nil {
		return db.BusinessCredential{}, fmt.Errorf("create credentials: %w", err)
	}
	return c, nil
}

func (r *PgxBusinessRepo) GetActiveCredentials(ctx context.Context, businessID string) (db.BusinessCredential, error) {
	c, err := r.q.GetActiveCredentials(ctx, businessID)
	if err != nil {
		return db.BusinessCredential{}, fmt.Errorf("get active credentials: %w", err)
	}
	return c, nil
}

func (r *PgxBusinessRepo) DeactivateCredentials(ctx context.Context, businessID string) error {
	if err := r.q.DeactivateCredentials(ctx, businessID); err != nil {
		return fmt.Errorf("deactivate credentials: %w", err)
	}
	return nil
}

func (r *PgxBusinessRepo) CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
	a, err := r.q.CreateJournalAccount(ctx, params)
	if err != nil {
		return db.JournalAccount{}, fmt.Errorf("create journal account: %w", err)
	}
	return a, nil
}

func (r *PgxBusinessRepo) ListJournalAccounts(ctx context.Context, businessID string) ([]db.JournalAccount, error) {
	accounts, err := r.q.ListJournalAccounts(ctx, businessID)
	if err != nil {
		return nil, fmt.Errorf("list journal accounts: %w", err)
	}
	return accounts, nil
}

// compile-time interface check
var _ BusinessRepo = (*PgxBusinessRepo)(nil)

// unused import guard
var _ = sql.ErrNoRows
