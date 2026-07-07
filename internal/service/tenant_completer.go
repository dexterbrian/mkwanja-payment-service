package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/stdlib"

	db "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
)

// TenantCompleter settles payments on the correct consumer database.
// The asynq worker resolves the consumer ID from the Redis routing key and
// delegates here; this builds a per-consumer PaymentService on demand.
type TenantCompleter struct {
	registry   dbPgRegistry
	encryptKey []byte
	logger     *slog.Logger
}

// NewTenantCompleter creates a TenantCompleter.
func NewTenantCompleter(registry dbPgRegistry, encryptKey []byte, logger *slog.Logger) *TenantCompleter {
	if logger == nil {
		logger = slog.Default()
	}
	return &TenantCompleter{registry: registry, encryptKey: encryptKey, logger: logger}
}

func (t *TenantCompleter) paymentService(consumerID string) (*PaymentService, error) {
	pool, err := t.registry.Get(consumerID)
	if err != nil {
		return nil, fmt.Errorf("resolve consumer db %s: %w", consumerID, err)
	}
	stdlibDB := stdlib.OpenDBFromPool(pool)
	q := db.New(stdlibDB)
	return &PaymentService{
		paymentRepo: repository.NewPgxPaymentRepo(q),
		clientRepo:  repository.NewPgxClientRepo(q),
		journalRepo: repository.NewPgxJournalRepo(q, stdlibDB),
		encryptKey:  t.encryptKey,
		logger:      t.logger.With("consumer_id", consumerID),
	}, nil
}

// CompletePayment settles a payment as completed on the consumer's database.
func (t *TenantCompleter) CompletePayment(ctx context.Context, consumerID, paymentID, receipt, txID string) error {
	svc, err := t.paymentService(consumerID)
	if err != nil {
		return err
	}
	return svc.CompletePayment(ctx, paymentID, receipt, txID)
}

// FailPayment settles a payment as failed on the consumer's database.
func (t *TenantCompleter) FailPayment(ctx context.Context, consumerID, paymentID, reason string) error {
	svc, err := t.paymentService(consumerID)
	if err != nil {
		return err
	}
	return svc.FailPayment(ctx, paymentID, reason)
}
