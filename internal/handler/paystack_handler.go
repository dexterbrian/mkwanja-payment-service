package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/service"
)

// PaystackHandler handles Paystack HTTP endpoints.
type PaystackHandler struct {
	svc    *service.PaystackService
	logger *slog.Logger
}

// NewPaystackHandler creates a PaystackHandler.
func NewPaystackHandler(svc *service.PaystackService, logger *slog.Logger) *PaystackHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaystackHandler{svc: svc, logger: logger}
}

// RegisterCredentialsRequest is the JSON body for PUT /v1/clients/:client_id/paystack-credentials.
type RegisterCredentialsRequest struct {
	SecretKey string `json:"secret_key"`
	PublicKey string `json:"public_key"`
}

// RegisterCredentials handles PUT /v1/clients/:client_id/paystack-credentials.
func (h *PaystackHandler) RegisterCredentials(c *fiber.Ctx) error {
	var req RegisterCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	cred, err := h.svc.RegisterCredentials(c.Context(), service.RegisterPaystackCredentialsRequest{
		ClientID:  c.Params("client_id"),
		SecretKey: req.SecretKey,
		PublicKey: req.PublicKey,
	})
	if err != nil {
		h.logger.Error("register paystack credentials failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("REGISTRATION_FAILED", err.Error()))
	}

	// Never return the secret key (even encrypted).
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":         cred.ID,
		"client_id":  cred.ClientID,
		"public_key": cred.PublicKey,
		"active":     cred.IsActive,
		"created_at": cred.CreatedAt,
	})
}

// InitiateChargeRequest is the JSON body for POST /v1/payments/paystack/initialize.
type InitiateChargeRequest struct {
	ClientID    string `json:"client_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Email       string `json:"email"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
	CallbackURL string `json:"callback_url"`
}

// InitiateCharge handles POST /v1/payments/paystack/initialize.
func (h *PaystackHandler) InitiateCharge(c *fiber.Ctx) error {
	var req InitiateChargeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	consumerID, _ := c.Locals("consumer_id").(string)
	idempotencyKey, _ := c.Locals("idempotency_key").(string)

	result, err := h.svc.InitiateCharge(c.Context(), service.InitiatePaystackChargeRequest{
		ClientID:       req.ClientID,
		ConsumerID:     consumerID,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		Email:          req.Email,
		Reference:      req.Reference,
		Description:    req.Description,
		CallbackURL:    req.CallbackURL,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		h.logger.Error("initiate paystack charge failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INITIATION_FAILED", err.Error()))
	}

	if result.IdempotencyReplayed {
		c.Response().Header.Set("Idempotency-Replayed", "true")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"payment_id":        result.PaymentID,
		"authorization_url": result.AuthorizationURL,
		"access_code":       result.AccessCode,
		"status":            result.Status,
	})
}

// HandleWebhook handles POST /webhooks/paystack/:consumer_id.
// Always returns 200 so Paystack doesn't retry forever; failures are
// logged and recovered by verification on the next delivery or manually.
func (h *PaystackHandler) HandleWebhook(c *fiber.Ctx) error {
	signature := c.Get("x-paystack-signature")
	body := c.Body()

	if err := h.svc.HandleWebhook(c.Context(), body, signature); err != nil {
		h.logger.Error("paystack webhook processing failed",
			"consumer_id", c.Params("consumer_id"),
			"error", err,
		)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}
