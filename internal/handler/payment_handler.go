package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/service"
)

// PaymentHandler handles payment HTTP endpoints.
type PaymentHandler struct {
	svc    *service.PaymentService
	logger *slog.Logger
}

// NewPaymentHandler creates a PaymentHandler.
func NewPaymentHandler(svc *service.PaymentService, logger *slog.Logger) *PaymentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &PaymentHandler{svc: svc, logger: logger}
}

// InitiateSTKPushRequest is the JSON body for POST /v1/payments/stk-push.
type InitiateSTKPushRequest struct {
	ClientID    string `json:"client_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	PhoneNumber string `json:"phone_number"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
}

// InitiateSTKPush handles POST /v1/payments/stk-push.
func (h *PaymentHandler) InitiateSTKPush(c *fiber.Ctx) error {
	var req InitiateSTKPushRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	consumerID, _ := c.Locals("consumer_id").(string)
	idempotencyKey, _ := c.Locals("idempotency_key").(string)

	result, err := h.svc.InitiateSTKPush(c.Context(), service.InitiateSTKPushRequest{
		ClientID:       req.ClientID,
		ConsumerID:     consumerID,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		PhoneNumber:    req.PhoneNumber,
		Reference:      req.Reference,
		Description:    req.Description,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		h.logger.Error("initiate stk push failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INITIATION_FAILED", err.Error()))
	}

	if result.IdempotencyReplayed {
		c.Response().Header.Set("Idempotency-Replayed", "true")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"payment_id":          result.PaymentID,
		"checkout_request_id": result.CheckoutRequestID,
		"status":              result.Status,
	})
}

// InitiateB2CRequest is the JSON body for POST /v1/payments/b2c.
type InitiateB2CRequest struct {
	ClientID    string `json:"client_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	PhoneNumber string `json:"phone_number"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
	CommandID   string `json:"command_id"`
	Remarks     string `json:"remarks"`
	Occasion    string `json:"occasion"`
}

// InitiateB2C handles POST /v1/payments/b2c.
func (h *PaymentHandler) InitiateB2C(c *fiber.Ctx) error {
	var req InitiateB2CRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	consumerID, _ := c.Locals("consumer_id").(string)
	idempotencyKey, _ := c.Locals("idempotency_key").(string)

	result, err := h.svc.InitiateB2C(c.Context(), service.InitiateB2CRequest{
		ClientID:       req.ClientID,
		ConsumerID:     consumerID,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		PhoneNumber:    req.PhoneNumber,
		Reference:      req.Reference,
		Description:    req.Description,
		IdempotencyKey: idempotencyKey,
		CommandID:      req.CommandID,
		Remarks:        req.Remarks,
		Occasion:       req.Occasion,
	})
	if err != nil {
		h.logger.Error("initiate b2c failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INITIATION_FAILED", err.Error()))
	}

	if result.IdempotencyReplayed {
		c.Response().Header.Set("Idempotency-Replayed", "true")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"payment_id":               result.PaymentID,
		"conversation_id":          result.ConversationID,
		"originator_conversation_id": result.OriginatorConvID,
		"status":                   result.Status,
	})
}

// InitiateB2BRequest is the JSON body for POST /v1/payments/b2b.
type InitiateB2BRequest struct {
	ClientID          string `json:"client_id"`
	AmountCents       int64  `json:"amount_cents"`
	Currency          string `json:"currency"`
	ReceiverShortcode string `json:"receiver_shortcode"`
	Reference         string `json:"reference"`
	Description       string `json:"description"`
	CommandID         string `json:"command_id"`
	SenderIDType      string `json:"sender_identifier_type"`
	ReceiverIDType    string `json:"receiver_identifier_type"`
	Remarks           string `json:"remarks"`
}

// InitiateB2B handles POST /v1/payments/b2b.
func (h *PaymentHandler) InitiateB2B(c *fiber.Ctx) error {
	var req InitiateB2BRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	consumerID, _ := c.Locals("consumer_id").(string)
	idempotencyKey, _ := c.Locals("idempotency_key").(string)

	result, err := h.svc.InitiateB2B(c.Context(), service.InitiateB2BRequest{
		ClientID:          req.ClientID,
		ConsumerID:        consumerID,
		AmountCents:       req.AmountCents,
		Currency:          req.Currency,
		ReceiverShortcode: req.ReceiverShortcode,
		Reference:         req.Reference,
		Description:       req.Description,
		IdempotencyKey:    idempotencyKey,
		CommandID:         req.CommandID,
		SenderIDType:      req.SenderIDType,
		ReceiverIDType:    req.ReceiverIDType,
		Remarks:           req.Remarks,
	})
	if err != nil {
		h.logger.Error("initiate b2b failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INITIATION_FAILED", err.Error()))
	}

	if result.IdempotencyReplayed {
		c.Response().Header.Set("Idempotency-Replayed", "true")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"payment_id":               result.PaymentID,
		"conversation_id":          result.ConversationID,
		"originator_conversation_id": result.OriginatorConvID,
		"status":                   result.Status,
	})
}

// GetPayment handles GET /v1/payments/:id.
func (h *PaymentHandler) GetPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Payment ID is required"))
	}

	payment, err := h.svc.GetPayment(c.Context(), id)
	if err != nil {
		h.logger.Error("get payment failed", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(errResp("NOT_FOUND", "Payment not found"))
	}

	return c.JSON(fiber.Map{
		"id":                payment.ID,
		"client_id":         payment.ClientID,
		"idempotency_key":   payment.IdempotencyKey,
		"provider":          payment.Provider,
		"payment_type":      payment.PaymentType,
		"direction":         payment.Direction,
		"status":            payment.Status,
		"amount_cents":      payment.AmountCents,
		"currency":          payment.Currency,
		"phone_number":      payment.PhoneNumber,
		"reference":         payment.Reference,
		"description":       payment.Description,
		"provider_request_id": payment.ProviderRequestID,
		"provider_tx_id":    payment.ProviderTxID,
		"provider_receipt":  payment.ProviderReceipt,
		"created_at":        payment.CreatedAt,
		"updated_at":        payment.UpdatedAt,
		"completed_at":      payment.CompletedAt,
	})
}

// ListPayments handles GET /v1/payments.
func (h *PaymentHandler) ListPayments(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "client_id query parameter is required"))
	}

	limit := int32(c.QueryInt("limit", 20))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := int32(c.QueryInt("offset", 0))
	if offset < 0 {
		offset = 0
	}

	payments, err := h.svc.ListPayments(c.Context(), clientID, limit, offset)
	if err != nil {
		h.logger.Error("list payments failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errResp("INTERNAL", "Failed to list payments"))
	}

	items := make([]fiber.Map, 0, len(payments))
	for _, p := range payments {
		items = append(items, fiber.Map{
			"id":           p.ID,
			"client_id":    p.ClientID,
			"provider":     p.Provider,
			"payment_type": p.PaymentType,
			"status":       p.Status,
			"amount_cents": p.AmountCents,
			"currency":     p.Currency,
			"reference":    p.Reference,
			"created_at":   p.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"payments": items, "limit": limit, "offset": offset})
}
