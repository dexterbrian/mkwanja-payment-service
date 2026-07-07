package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// Enqueuer pushes webhook processing tasks onto the asynq queue.
type Enqueuer struct {
	client *asynq.Client
}

// NewEnqueuer creates an Enqueuer connected to the given Redis address.
func NewEnqueuer(redisAddr string) *Enqueuer {
	return &Enqueuer{client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

// Close releases the underlying asynq client.
func (e *Enqueuer) Close() error {
	return e.client.Close()
}

// EnqueueSTKWebhook queues a raw STK callback body for processing.
func (e *Enqueuer) EnqueueSTKWebhook(ctx context.Context, consumerID, rawBody string) error {
	return e.enqueue(ctx, TypeProcessSTKWebhook, STKWebhookPayload{ConsumerID: consumerID, RawBody: rawBody})
}

// EnqueueB2CWebhook queues a raw B2C callback body for processing.
func (e *Enqueuer) EnqueueB2CWebhook(ctx context.Context, consumerID, rawBody string) error {
	return e.enqueue(ctx, TypeProcessB2CWebhook, B2CWebhookPayload{ConsumerID: consumerID, RawBody: rawBody})
}

// EnqueueB2BWebhook queues a raw B2B callback body for processing.
func (e *Enqueuer) EnqueueB2BWebhook(ctx context.Context, consumerID, rawBody string) error {
	return e.enqueue(ctx, TypeProcessB2BWebhook, B2BWebhookPayload{ConsumerID: consumerID, RawBody: rawBody})
}

func (e *Enqueuer) enqueue(ctx context.Context, taskType string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", taskType, err)
	}
	if _, err := e.client.EnqueueContext(ctx, asynq.NewTask(taskType, b)); err != nil {
		return fmt.Errorf("enqueue %s: %w", taskType, err)
	}
	return nil
}
