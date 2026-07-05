package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/service"
)

// LedgerHandler handles ledger query HTTP endpoints.
type LedgerHandler struct {
	svc    *service.JournalService
	logger *slog.Logger
}

// NewLedgerHandler creates a LedgerHandler.
func NewLedgerHandler(svc *service.JournalService, logger *slog.Logger) *LedgerHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &LedgerHandler{svc: svc, logger: logger}
}

// ListEntries handles GET /v1/ledger.
func (h *LedgerHandler) ListEntries(c *fiber.Ctx) error {
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

	entries, err := h.svc.ListJournalEntriesByClient(c.Context(), clientID, limit, offset)
	if err != nil {
		h.logger.Error("list journal entries failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errResp("INTERNAL", "Failed to list journal entries"))
	}

	items := make([]fiber.Map, 0, len(entries))
	for _, e := range entries {
		items = append(items, fiber.Map{
			"id":           e.ID,
			"client_id":    e.ClientID,
			"payment_id":   e.PaymentID,
			"account_id":   e.AccountID,
			"entry_type":   e.EntryType,
			"amount_cents": e.AmountCents,
			"currency":     e.Currency,
			"description":  e.Description,
			"created_at":   e.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{"entries": items, "limit": limit, "offset": offset})
}

// GetBalances handles GET /v1/ledger/balance.
func (h *LedgerHandler) GetBalances(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "client_id query parameter is required"))
	}

	balances, err := h.svc.GetAccountBalances(c.Context(), clientID)
	if err != nil {
		h.logger.Error("get account balances failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errResp("INTERNAL", "Failed to get account balances"))
	}

	items := make([]fiber.Map, 0, len(balances))
	for _, b := range balances {
		items = append(items, fiber.Map{
			"account_id":          b.AccountID,
			"total_debits_cents":  b.TotalDebitsCents,
			"total_credits_cents": b.TotalCreditsCents,
			"net_cents":           b.NetCents,
		})
	}

	return c.JSON(fiber.Map{"balances": items})
}

// GetTrialBalance handles GET /v1/ledger/trial-balance.
func (h *LedgerHandler) GetTrialBalance(c *fiber.Ctx) error {
	clientID := c.Query("client_id")
	if clientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errResp("INVALID_REQUEST", "client_id query parameter is required"))
	}

	balances, err := h.svc.GetTrialBalance(c.Context(), clientID)
	if err != nil {
		h.logger.Error("get trial balance failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(errResp("INTERNAL", "Failed to get trial balance"))
	}

	items := make([]fiber.Map, 0, len(balances))
	for _, b := range balances {
		items = append(items, fiber.Map{
			"account_id":          b.AccountID,
			"total_debits_cents":  b.TotalDebitsCents,
			"total_credits_cents": b.TotalCreditsCents,
			"net_cents":           b.NetCents,
		})
	}

	return c.JSON(fiber.Map{"trial_balance": items})
}
