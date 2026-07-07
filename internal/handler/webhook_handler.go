package handler

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// WebhookEnqueuer queues raw callback bodies for async processing.
type WebhookEnqueuer interface {
	EnqueueSTKWebhook(ctx context.Context, consumerID, rawBody string) error
	EnqueueB2CWebhook(ctx context.Context, consumerID, rawBody string) error
	EnqueueB2BWebhook(ctx context.Context, consumerID, rawBody string) error
}

// WebhookHandler handles M-PESA callback endpoints. Callbacks always get
// HTTP 200 (per docs/rules.md); processing happens on the asynq queue and
// missed webhooks are recovered by the reconciliation job.
type WebhookHandler struct {
	enqueuer WebhookEnqueuer
	logger   *slog.Logger
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(enqueuer WebhookEnqueuer, logger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{enqueuer: enqueuer, logger: logger}
}

// HandleSTKCallback handles POST /webhooks/mpesa/stk/:consumer_id.
func (h *WebhookHandler) HandleSTKCallback(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("stk callback received", "consumer_id", consumerID)

	if h.enqueuer != nil {
		if err := h.enqueuer.EnqueueSTKWebhook(c.Context(), consumerID, string(body)); err != nil {
			h.logger.Error("failed to enqueue stk webhook", "consumer_id", consumerID, "error", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// HandleB2CCallback handles POST /webhooks/mpesa/b2c/:consumer_id.
func (h *WebhookHandler) HandleB2CCallback(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("b2c callback received", "consumer_id", consumerID)

	if h.enqueuer != nil {
		if err := h.enqueuer.EnqueueB2CWebhook(c.Context(), consumerID, string(body)); err != nil {
			h.logger.Error("failed to enqueue b2c webhook", "consumer_id", consumerID, "error", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// HandleB2BCallback handles POST /webhooks/mpesa/b2b/:consumer_id.
func (h *WebhookHandler) HandleB2BCallback(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("b2b callback received", "consumer_id", consumerID)

	if h.enqueuer != nil {
		if err := h.enqueuer.EnqueueB2BWebhook(c.Context(), consumerID, string(body)); err != nil {
			h.logger.Error("failed to enqueue b2b webhook", "consumer_id", consumerID, "error", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// HandleC2BConfirmation handles POST /webhooks/mpesa/c2b/:consumer_id/confirm.
func (h *WebhookHandler) HandleC2BConfirmation(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")

	h.logger.Info("c2b confirmation received", "consumer_id", consumerID, "body", string(c.Body()))

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"ResultCode": 0,
		"ResultDesc": "Success",
	})
}

// HandleC2BValidation handles POST /webhooks/mpesa/c2b/:consumer_id/validate.
func (h *WebhookHandler) HandleC2BValidation(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")

	h.logger.Info("c2b validation received", "consumer_id", consumerID, "body", string(c.Body()))

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"ResultCode": 0,
		"ResultDesc": "Success",
	})
}
