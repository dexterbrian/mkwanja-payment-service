package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles M-PESA callback endpoints.
type WebhookHandler struct {
	logger *slog.Logger
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(logger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{logger: logger}
}

// HandleSTKCallback handles POST /webhooks/mpesa/stk/:consumer_id.
func (h *WebhookHandler) HandleSTKCallback(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("stk callback received",
		"consumer_id", consumerID,
		"body", string(body),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// HandleB2CCallback handles POST /webhooks/mpesa/b2c/:consumer_id.
func (h *WebhookHandler) HandleB2CCallback(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("b2c callback received",
		"consumer_id", consumerID,
		"body", string(body),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// HandleB2BCallback handles POST /webhooks/mpesa/b2b/:consumer_id.
func (h *WebhookHandler) HandleB2BCallback(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("b2b callback received",
		"consumer_id", consumerID,
		"body", string(body),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

// HandleC2BConfirmation handles POST /webhooks/mpesa/c2b/:consumer_id/confirm.
func (h *WebhookHandler) HandleC2BConfirmation(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("c2b confirmation received",
		"consumer_id", consumerID,
		"body", string(body),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"ResultCode": 0,
		"ResultDesc": "Success",
	})
}

// HandleC2BValidation handles POST /webhooks/mpesa/c2b/:consumer_id/validate.
func (h *WebhookHandler) HandleC2BValidation(c *fiber.Ctx) error {
	consumerID := c.Params("consumer_id")
	body := c.Body()

	h.logger.Info("c2b validation received",
		"consumer_id", consumerID,
		"body", string(body),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"ResultCode": 0,
		"ResultDesc": "Success",
	})
}
