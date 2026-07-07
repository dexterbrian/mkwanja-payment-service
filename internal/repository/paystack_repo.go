package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	db "mkwanja-payment-svc/internal/db/generated"
)

// PaystackRepo defines Paystack credential persistence operations.
type PaystackRepo interface {
	CreatePaystackCredentials(ctx context.Context, params db.CreatePaystackCredentialsParams) (db.ClientPaystackCredential, error)
	GetActivePaystackCredentials(ctx context.Context, clientID string) (db.ClientPaystackCredential, error)
	DeactivatePaystackCredentials(ctx context.Context, clientID string) error
}

// PgxPaystackRepo implements PaystackRepo using a db.Querier.
type PgxPaystackRepo struct {
	q db.Querier
}

// NewPgxPaystackRepo creates a repo backed by a db.Querier.
func NewPgxPaystackRepo(q db.Querier) *PgxPaystackRepo {
	return &PgxPaystackRepo{q: q}
}

// NewPgxPaystackRepoFromPool creates a repo from a pgxpool.Pool.
func NewPgxPaystackRepoFromPool(pool *pgxpool.Pool) *PgxPaystackRepo {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	return &PgxPaystackRepo{q: db.New(stdlibDB)}
}

func (r *PgxPaystackRepo) CreatePaystackCredentials(ctx context.Context, params db.CreatePaystackCredentialsParams) (db.ClientPaystackCredential, error) {
	c, err := r.q.CreatePaystackCredentials(ctx, params)
	if err != nil {
		return db.ClientPaystackCredential{}, fmt.Errorf("create paystack credentials: %w", err)
	}
	return c, nil
}

func (r *PgxPaystackRepo) GetActivePaystackCredentials(ctx context.Context, clientID string) (db.ClientPaystackCredential, error) {
	c, err := r.q.GetActivePaystackCredentials(ctx, clientID)
	if err != nil {
		return db.ClientPaystackCredential{}, fmt.Errorf("get active paystack credentials: %w", err)
	}
	return c, nil
}

func (r *PgxPaystackRepo) DeactivatePaystackCredentials(ctx context.Context, clientID string) error {
	if err := r.q.DeactivatePaystackCredentials(ctx, clientID); err != nil {
		return fmt.Errorf("deactivate paystack credentials: %w", err)
	}
	return nil
}

var _ PaystackRepo = (*PgxPaystackRepo)(nil)
