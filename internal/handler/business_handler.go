package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/service"
)

// BusinessHandler handles business and credential HTTP endpoints.
type BusinessHandler struct {
	svc    *service.BusinessService
	logger *slog.Logger
}

// NewBusinessHandler creates a BusinessHandler.
func NewBusinessHandler(svc *service.BusinessService, logger *slog.Logger) *BusinessHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BusinessHandler{svc: svc, logger: logger}
}

// RegisterBusinessRequest is the JSON body for POST /v1/businesses.
type RegisterBusinessRequest struct {
	ExternalID         string `json:"external_id"`
	Name               string `json:"name"`
	Shortcode          string `json:"shortcode"`
	ConsumerKey        string `json:"consumer_key"`
	ConsumerSecret     string `json:"consumer_secret"`
	Passkey            string `json:"passkey"`
	InitiatorName      string `json:"initiator_name"`
	SecurityCredential string `json:"security_credential"`
}

// RegisterBusiness handles POST /v1/businesses.
func (h *BusinessHandler) RegisterBusiness(c *fiber.Ctx) error {
	var req RegisterBusinessRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	b, err := h.svc.RegisterBusiness(c.Context(), service.RegisterBusinessRequest{
		ExternalID:         req.ExternalID,
		Name:               req.Name,
		Shortcode:          req.Shortcode,
		ConsumerKey:        req.ConsumerKey,
		ConsumerSecret:     req.ConsumerSecret,
		Passkey:            req.Passkey,
		InitiatorName:      req.InitiatorName,
		SecurityCredential: req.SecurityCredential,
	})
	if err != nil {
		h.logger.Error("register business failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("REGISTRATION_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":          b.ID,
		"external_id": b.ExternalID,
		"name":        b.Name,
		"active":      b.Active,
		"created_at":  b.CreatedAt,
	})
}

// TestCredentialsRequest is the JSON body for POST /v1/businesses/test-credentials.
type TestCredentialsRequest struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
	Shortcode      string `json:"shortcode"`
}

// TestCredentials handles POST /v1/businesses/test-credentials.
func (h *BusinessHandler) TestCredentials(c *fiber.Ctx) error {
	var req TestCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	if err := h.svc.TestCredentials(c.Context(), service.TestCredentialsRequest{
		ConsumerKey:    req.ConsumerKey,
		ConsumerSecret: req.ConsumerSecret,
		Shortcode:      req.Shortcode,
	}); err != nil {
		h.logger.Error("test credentials failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("TEST_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "message": "Credentials are valid"})
}

// UpdateCredentialsRequest is the JSON body for PUT /v1/businesses/:id/credentials.
type UpdateCredentialsRequest struct {
	Shortcode          string `json:"shortcode"`
	ConsumerKey        string `json:"consumer_key"`
	ConsumerSecret     string `json:"consumer_secret"`
	Passkey            string `json:"passkey"`
	InitiatorName      string `json:"initiator_name"`
	SecurityCredential string `json:"security_credential"`
}

// UpdateCredentials handles PUT /v1/businesses/:id/credentials.
func (h *BusinessHandler) UpdateCredentials(c *fiber.Ctx) error {
	businessID := c.Params("id")
	if businessID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Business ID is required"))
	}

	var req UpdateCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	if err := h.svc.UpdateCredentials(c.Context(), service.UpdateCredentialsRequest{
		BusinessID:         businessID,
		Shortcode:          req.Shortcode,
		ConsumerKey:        req.ConsumerKey,
		ConsumerSecret:     req.ConsumerSecret,
		Passkey:            req.Passkey,
		InitiatorName:      req.InitiatorName,
		SecurityCredential: req.SecurityCredential,
	}); err != nil {
		h.logger.Error("update credentials failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("UPDATE_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "message": "Credentials updated"})
}

// DeactivateBusiness handles DELETE /v1/businesses/:id.
func (h *BusinessHandler) DeactivateBusiness(c *fiber.Ctx) error {
	businessID := c.Params("id")
	if businessID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Business ID is required"))
	}

	if err := h.svc.DeactivateBusiness(c.Context(), businessID); err != nil {
		h.logger.Error("deactivate business failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("DEACTIVATION_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "message": "Business deactivated"})
}

// GetBusiness handles GET /v1/businesses/:id.
func (h *BusinessHandler) GetBusiness(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Business ID is required"))
	}

	b, err := h.svc.GetBusiness(c.Context(), id)
	if err != nil {
		h.logger.Error("get business failed", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(errResp("NOT_FOUND", "Business not found"))
	}

	return c.JSON(fiber.Map{
		"id":          b.ID,
		"external_id": b.ExternalID,
		"name":        b.Name,
		"active":      b.Active,
		"created_at":  b.CreatedAt,
		"updated_at":  b.UpdatedAt,
	})
}

// ListBusinesses handles GET /v1/businesses.
func (h *BusinessHandler) ListBusinesses(c *fiber.Ctx) error {
	businesses, err := h.svc.ListBusinesses(c.Context())
	if err != nil {
		h.logger.Error("list businesses failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errResp("INTERNAL", "Failed to list businesses"))
	}

	items := make([]fiber.Map, 0, len(businesses))
	for _, b := range businesses {
		items = append(items, fiber.Map{
			"id":          b.ID,
			"external_id": b.ExternalID,
			"name":        b.Name,
			"active":      b.Active,
			"created_at":  b.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"businesses": items})
}

func errResp(code, message string) fiber.Map {
	return fiber.Map{
		"error": fiber.Map{
			"code":      code,
			"message":   message,
			"retryable": false,
		},
	}
}
