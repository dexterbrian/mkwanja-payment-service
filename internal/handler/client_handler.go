package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/service"
)

// ClientHandler handles client and credential HTTP endpoints.
type ClientHandler struct {
	svc    *service.ClientService
	logger *slog.Logger
}

// NewClientHandler creates a ClientHandler.
func NewClientHandler(svc *service.ClientService, logger *slog.Logger) *ClientHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ClientHandler{svc: svc, logger: logger}
}

// RegisterClientRequest is the JSON body for POST /v1/clients.
type RegisterClientRequest struct {
	ExternalID         string `json:"external_id"`
	Name               string `json:"name"`
	Shortcode          string `json:"shortcode"`
	ConsumerKey        string `json:"consumer_key"`
	ConsumerSecret     string `json:"consumer_secret"`
	Passkey            string `json:"passkey"`
	InitiatorName      string `json:"initiator_name"`
	SecurityCredential string `json:"security_credential"`
}

// RegisterClient handles POST /v1/clients.
func (h *ClientHandler) RegisterClient(c *fiber.Ctx) error {
	var req RegisterClientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	client, err := h.svc.RegisterClient(c.Context(), service.RegisterClientRequest{
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
		h.logger.Error("register client failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("REGISTRATION_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":          client.ID,
		"external_id": client.ExternalID,
		"name":        client.Name,
		"active":      client.Active,
		"created_at":  client.CreatedAt,
	})
}

// TestCredentialsRequest is the JSON body for POST /v1/clients/test-credentials.
type TestCredentialsRequest struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
	Shortcode      string `json:"shortcode"`
}

// TestCredentials handles POST /v1/clients/test-credentials.
func (h *ClientHandler) TestCredentials(c *fiber.Ctx) error {
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

// UpdateCredentialsRequest is the JSON body for PUT /v1/clients/:client_id/credentials.
type UpdateCredentialsRequest struct {
	Shortcode          string `json:"shortcode"`
	ConsumerKey        string `json:"consumer_key"`
	ConsumerSecret     string `json:"consumer_secret"`
	Passkey            string `json:"passkey"`
	InitiatorName      string `json:"initiator_name"`
	SecurityCredential string `json:"security_credential"`
}

// UpdateCredentials handles PUT /v1/clients/:client_id/credentials.
func (h *ClientHandler) UpdateCredentials(c *fiber.Ctx) error {
	clientID := c.Params("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Client ID is required"))
	}

	var req UpdateCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	if err := h.svc.UpdateCredentials(c.Context(), service.UpdateCredentialsRequest{
		ClientID:           clientID,
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

// UpdateOperatorCredentials handles PUT /v1/admin/operator/credentials.
func (h *ClientHandler) UpdateOperatorCredentials(c *fiber.Ctx, operatorClientID string) error {
	if operatorClientID == "" {
		return c.Status(fiber.StatusNotFound).JSON(errResp("NOT_CONFIGURED", "Operator client is not configured"))
	}

	var req UpdateCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Invalid JSON body"))
	}

	if err := h.svc.UpdateCredentials(c.Context(), service.UpdateCredentialsRequest{
		ClientID:           operatorClientID,
		Shortcode:          req.Shortcode,
		ConsumerKey:        req.ConsumerKey,
		ConsumerSecret:     req.ConsumerSecret,
		Passkey:            req.Passkey,
		InitiatorName:      req.InitiatorName,
		SecurityCredential: req.SecurityCredential,
	}); err != nil {
		h.logger.Error("update operator credentials failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("UPDATE_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "message": "Operator credentials updated"})
}

// DeactivateClient handles DELETE /v1/clients/:client_id.
func (h *ClientHandler) DeactivateClient(c *fiber.Ctx) error {
	clientID := c.Params("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Client ID is required"))
	}

	if err := h.svc.DeactivateClient(c.Context(), clientID); err != nil {
		h.logger.Error("deactivate client failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(errResp("DEACTIVATION_FAILED", err.Error()))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "message": "Client deactivated"})
}

// GetClient handles GET /v1/clients/:client_id.
func (h *ClientHandler) GetClient(c *fiber.Ctx) error {
	id := c.Params("client_id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "Client ID is required"))
	}

	client, err := h.svc.GetClient(c.Context(), id)
	if err != nil {
		h.logger.Error("get client failed", "error", err)
		return c.Status(fiber.StatusNotFound).JSON(errResp("NOT_FOUND", "Client not found"))
	}

	return c.JSON(fiber.Map{
		"id":          client.ID,
		"external_id": client.ExternalID,
		"name":        client.Name,
		"active":      client.Active,
		"created_at":  client.CreatedAt,
		"updated_at":  client.UpdatedAt,
	})
}

// ListClients handles GET /v1/clients.
func (h *ClientHandler) ListClients(c *fiber.Ctx) error {
	clients, err := h.svc.ListClients(c.Context())
	if err != nil {
		h.logger.Error("list clients failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errResp("INTERNAL", "Failed to list clients"))
	}

	items := make([]fiber.Map, 0, len(clients))
	for _, client := range clients {
		items = append(items, fiber.Map{
			"id":          client.ID,
			"external_id": client.ExternalID,
			"name":        client.Name,
			"active":      client.Active,
			"created_at":  client.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"clients": items})
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
