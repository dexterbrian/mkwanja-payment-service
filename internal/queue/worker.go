package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"mkwanja-payment-svc/internal/daraja"
)

// PaymentCompleter abstracts payment completion for the worker.
type PaymentCompleter interface {
	CompletePayment(ctx context.Context, paymentID, receipt, txID string) error
	FailPayment(ctx context.Context, paymentID, reason string) error
}

// Worker processes webhook tasks from the asynq queue.
type Worker struct {
	server    *asynq.Server
	mux       *asynq.ServeMux
	completer PaymentCompleter
	rdb       *redis.Client
	logger    *slog.Logger
}

// NewWorker creates and starts an asynq worker.
func NewWorker(redisAddr string, completer PaymentCompleter, rdb *redis.Client, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{Concurrency: 10},
	)

	w := &Worker{
		server:    srv,
		mux:       asynq.NewServeMux(),
		completer: completer,
		rdb:       rdb,
		logger:    logger,
	}

	w.mux.HandleFunc(TypeProcessSTKWebhook, w.handleSTKWebhook)
	w.mux.HandleFunc(TypeProcessB2CWebhook, w.handleB2CWebhook)
	w.mux.HandleFunc(TypeProcessB2BWebhook, w.handleB2BWebhook)

	return w
}

// Start begins processing tasks in the background.
func (w *Worker) Start() error {
	return w.server.Start(w.mux)
}

// Shutdown gracefully stops the worker.
func (w *Worker) Shutdown() {
	w.server.Shutdown()
}

func (w *Worker) handleSTKWebhook(ctx context.Context, t *asynq.Task) error {
	var payload STKWebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("parse stk webhook payload: %w", err)
	}

	var cb daraja.STKCallbackBody
	if err := json.Unmarshal([]byte(payload.RawBody), &cb); err != nil {
		return fmt.Errorf("parse stk callback body: %w", err)
	}

	checkoutID := cb.Body.StkCallback.CheckoutRequestID

	routingKey := "stk:" + checkoutID
	route, err := w.rdb.Get(ctx, routingKey).Result()
	if err != nil {
		return fmt.Errorf("lookup routing key %s: %w", routingKey, err)
	}

	parts := strings.SplitN(route, ":", 3)
	if len(parts) < 2 {
		return fmt.Errorf("invalid routing value: %s", route)
	}
	paymentID := parts[len(parts)-1]

	if cb.Body.StkCallback.ResultCode == 0 {
		receipt := extractMpesaReceipt(cb.Body.StkCallback.CallbackMetadata.Item)
		if err := w.completer.CompletePayment(ctx, paymentID, receipt, checkoutID); err != nil {
			return fmt.Errorf("complete payment %s: %w", paymentID, err)
		}
		w.logger.Info("stk payment completed", "payment_id", paymentID, "receipt", receipt)
	} else {
		reason := cb.Body.StkCallback.ResultDesc
		if err := w.completer.FailPayment(ctx, paymentID, reason); err != nil {
			return fmt.Errorf("fail payment %s: %w", paymentID, err)
		}
		w.logger.Info("stk payment failed", "payment_id", paymentID, "reason", reason)
	}

	w.logger.Info("stk webhook processed",
		"checkout_request_id", checkoutID,
		"result_code", cb.Body.StkCallback.ResultCode,
	)

	return nil
}

func (w *Worker) handleB2CWebhook(ctx context.Context, t *asynq.Task) error {
	var payload B2CWebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("parse b2c webhook payload: %w", err)
	}

	w.logger.Info("b2c webhook received", "consumer_id", payload.ConsumerID)

	return nil
}

func (w *Worker) handleB2BWebhook(ctx context.Context, t *asynq.Task) error {
	var payload B2BWebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("parse b2b webhook payload: %w", err)
	}

	w.logger.Info("b2b webhook received", "consumer_id", payload.ConsumerID)

	return nil
}

// extractMpesaReceipt extracts the M-PESA receipt number from callback metadata.
func extractMpesaReceipt(items []daraja.STKCallbackMetadataItem) string {
	for _, item := range items {
		if item.Name == "MpesaReceiptNumber" {
			if v, ok := item.Value.(string); ok {
				return v
			}
		}
	}
	return ""
}
