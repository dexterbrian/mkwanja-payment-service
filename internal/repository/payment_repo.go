package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	db "mkwanja-payment-svc/internal/db/generated"
)

// PaymentRepo defines payment persistence operations.
type PaymentRepo interface {
	CreatePayment(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error)
	GetPaymentByID(ctx context.Context, id string) (db.Payment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error)
	UpdatePaymentStatus(ctx context.Context, id string, status db.PaymentStatus) (db.Payment, error)
	CompletePayment(ctx context.Context, id string, providerReceipt, providerTxID sql.NullString) (db.Payment, error)
	FailPayment(ctx context.Context, id string) (db.Payment, error)
	UpdateProviderRequestID(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error)
	ListPaymentsByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error)
	ListPendingPaymentsOlderThan(ctx context.Context, createdAt time.Time) ([]db.Payment, error)
	CreatePaymentEvent(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error)
}

// PgxPaymentRepo implements PaymentRepo using a db.Querier.
type PgxPaymentRepo struct {
	q db.Querier
}

// NewPgxPaymentRepo creates a repo backed by a db.Querier.
func NewPgxPaymentRepo(q db.Querier) *PgxPaymentRepo {
	return &PgxPaymentRepo{q: q}
}

// NewPgxPaymentRepoFromPool creates a repo from a pgxpool.Pool.
func NewPgxPaymentRepoFromPool(pool *pgxpool.Pool) *PgxPaymentRepo {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	return &PgxPaymentRepo{q: db.New(stdlibDB)}
}

func (r *PgxPaymentRepo) CreatePayment(ctx context.Context, params db.CreatePaymentParams) (db.Payment, error) {
	p, err := r.q.CreatePayment(ctx, params)
	if err != nil {
		return db.Payment{}, fmt.Errorf("create payment: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) GetPaymentByID(ctx context.Context, id string) (db.Payment, error) {
	p, err := r.q.GetPaymentByID(ctx, id)
	if err != nil {
		return db.Payment{}, fmt.Errorf("get payment by id: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) GetPaymentByIdempotencyKey(ctx context.Context, clientID, idempotencyKey string) (db.Payment, error) {
	p, err := r.q.GetPaymentByIdempotencyKey(ctx, db.GetPaymentByIdempotencyKeyParams{
		ClientID:       clientID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return db.Payment{}, fmt.Errorf("get payment by idempotency key: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) UpdatePaymentStatus(ctx context.Context, id string, status db.PaymentStatus) (db.Payment, error) {
	p, err := r.q.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return db.Payment{}, fmt.Errorf("update payment status: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) CompletePayment(ctx context.Context, id string, providerReceipt, providerTxID sql.NullString) (db.Payment, error) {
	p, err := r.q.CompletePayment(ctx, db.CompletePaymentParams{
		ID:              id,
		ProviderReceipt: providerReceipt,
		ProviderTxID:    providerTxID,
	})
	if err != nil {
		return db.Payment{}, fmt.Errorf("complete payment: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) FailPayment(ctx context.Context, id string) (db.Payment, error) {
	p, err := r.q.FailPayment(ctx, id)
	if err != nil {
		return db.Payment{}, fmt.Errorf("fail payment: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) UpdateProviderRequestID(ctx context.Context, id string, providerRequestID, providerTxID sql.NullString) (db.Payment, error) {
	p, err := r.q.UpdateProviderRequestID(ctx, db.UpdateProviderRequestIDParams{
		ID:                id,
		ProviderRequestID: providerRequestID,
		ProviderTxID:      providerTxID,
	})
	if err != nil {
		return db.Payment{}, fmt.Errorf("update provider request id: %w", err)
	}
	return p, nil
}

func (r *PgxPaymentRepo) ListPaymentsByClient(ctx context.Context, clientID string, limit, offset int32) ([]db.Payment, error) {
	payments, err := r.q.ListPaymentsByClient(ctx, db.ListPaymentsByClientParams{
		ClientID: clientID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list payments by client: %w", err)
	}
	return payments, nil
}

func (r *PgxPaymentRepo) ListPendingPaymentsOlderThan(ctx context.Context, createdAt time.Time) ([]db.Payment, error) {
	payments, err := r.q.ListPendingPaymentsOlderThan(ctx, createdAt)
	if err != nil {
		return nil, fmt.Errorf("list pending payments older than: %w", err)
	}
	return payments, nil
}

func (r *PgxPaymentRepo) CreatePaymentEvent(ctx context.Context, params db.CreatePaymentEventParams) (db.PaymentEvent, error) {
	e, err := r.q.CreatePaymentEvent(ctx, params)
	if err != nil {
		return db.PaymentEvent{}, fmt.Errorf("create payment event: %w", err)
	}
	return e, nil
}

// compile-time interface check
var _ PaymentRepo = (*PgxPaymentRepo)(nil)

// unused import guard
var _ = sql.ErrNoRows
