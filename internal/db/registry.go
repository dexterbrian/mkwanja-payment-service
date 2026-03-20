package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Registry holds one pgxpool.Pool per consumer, keyed by consumer ID.
type Registry struct {
	mu    sync.RWMutex
	pools map[string]*pgxpool.Pool
}

func NewRegistry() *Registry {
	return &Registry{pools: make(map[string]*pgxpool.Pool)}
}

// Register creates and pings a connection pool for the given consumer.
func (r *Registry) Register(ctx context.Context, consumerID, databaseURL string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect %s: %w", consumerID, err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping %s: %w", consumerID, err)
	}
	r.mu.Lock()
	r.pools[consumerID] = pool
	r.mu.Unlock()
	return nil
}

// Get returns the pool for a consumer, or an error if not registered.
func (r *Registry) Get(consumerID string) (*pgxpool.Pool, error) {
	r.mu.RLock()
	pool, ok := r.pools[consumerID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no db for consumer: %s", consumerID)
	}
	return pool, nil
}

// Ping checks all registered consumer pools.
func (r *Registry) Ping(ctx context.Context) map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	results := make(map[string]error, len(r.pools))
	for id, pool := range r.pools {
		results[id] = pool.Ping(ctx)
	}
	return results
}
