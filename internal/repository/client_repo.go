package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	db "mkwanja-payment-svc/internal/db/generated"
)

// ClientRepo defines client and credential persistence operations.
type ClientRepo interface {
	CreateClient(ctx context.Context, params db.CreateClientParams) (db.Client, error)
	GetClientByID(ctx context.Context, id string) (db.Client, error)
	GetClientByExternalID(ctx context.Context, externalID string) (db.Client, error)
	ListClients(ctx context.Context) ([]db.Client, error)
	DeactivateClient(ctx context.Context, id string) (db.Client, error)

	CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error)
	GetActiveCredentials(ctx context.Context, clientID string) (db.ClientCredential, error)
	DeactivateCredentials(ctx context.Context, clientID string) error

	CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error)
	ListJournalAccounts(ctx context.Context, clientID string) ([]db.JournalAccount, error)
}

// PgxClientRepo implements ClientRepo using a db.Querier.
type PgxClientRepo struct {
	q db.Querier
}

// NewPgxClientRepo creates a repo backed by a db.Querier.
func NewPgxClientRepo(q db.Querier) *PgxClientRepo {
	return &PgxClientRepo{q: q}
}

// NewPgxClientRepoFromPool creates a repo from a pgxpool.Pool by opening
// a stdlib connection that satisfies the database/sql interface expected by sqlc.
func NewPgxClientRepoFromPool(pool *pgxpool.Pool) *PgxClientRepo {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	return &PgxClientRepo{q: db.New(stdlibDB)}
}

func (r *PgxClientRepo) CreateClient(ctx context.Context, params db.CreateClientParams) (db.Client, error) {
	c, err := r.q.CreateClient(ctx, params)
	if err != nil {
		return db.Client{}, fmt.Errorf("create client: %w", err)
	}
	return c, nil
}

func (r *PgxClientRepo) GetClientByID(ctx context.Context, id string) (db.Client, error) {
	c, err := r.q.GetClientByID(ctx, id)
	if err != nil {
		return db.Client{}, fmt.Errorf("get client by id: %w", err)
	}
	return c, nil
}

func (r *PgxClientRepo) GetClientByExternalID(ctx context.Context, externalID string) (db.Client, error) {
	c, err := r.q.GetClientByExternalID(ctx, externalID)
	if err != nil {
		return db.Client{}, fmt.Errorf("get client by external id: %w", err)
	}
	return c, nil
}

func (r *PgxClientRepo) ListClients(ctx context.Context) ([]db.Client, error) {
	clients, err := r.q.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	return clients, nil
}

func (r *PgxClientRepo) DeactivateClient(ctx context.Context, id string) (db.Client, error) {
	c, err := r.q.DeactivateClient(ctx, id)
	if err != nil {
		return db.Client{}, fmt.Errorf("deactivate client: %w", err)
	}
	return c, nil
}

func (r *PgxClientRepo) CreateCredentials(ctx context.Context, params db.CreateCredentialsParams) (db.ClientCredential, error) {
	c, err := r.q.CreateCredentials(ctx, params)
	if err != nil {
		return db.ClientCredential{}, fmt.Errorf("create credentials: %w", err)
	}
	return c, nil
}

func (r *PgxClientRepo) GetActiveCredentials(ctx context.Context, clientID string) (db.ClientCredential, error) {
	c, err := r.q.GetActiveCredentials(ctx, clientID)
	if err != nil {
		return db.ClientCredential{}, fmt.Errorf("get active credentials: %w", err)
	}
	return c, nil
}

func (r *PgxClientRepo) DeactivateCredentials(ctx context.Context, clientID string) error {
	if err := r.q.DeactivateCredentials(ctx, clientID); err != nil {
		return fmt.Errorf("deactivate credentials: %w", err)
	}
	return nil
}

func (r *PgxClientRepo) CreateJournalAccount(ctx context.Context, params db.CreateJournalAccountParams) (db.JournalAccount, error) {
	a, err := r.q.CreateJournalAccount(ctx, params)
	if err != nil {
		return db.JournalAccount{}, fmt.Errorf("create journal account: %w", err)
	}
	return a, nil
}

func (r *PgxClientRepo) ListJournalAccounts(ctx context.Context, clientID string) ([]db.JournalAccount, error) {
	accounts, err := r.q.ListJournalAccounts(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("list journal accounts: %w", err)
	}
	return accounts, nil
}

// compile-time interface check
var _ ClientRepo = (*PgxClientRepo)(nil)

// unused import guard
var _ = sql.ErrNoRows
