package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"mkwanja-payment-svc/internal/db"
)

// HealthHandler handles liveness and readiness probes.
type HealthHandler struct {
	registry *db.Registry
	redis    interface {
		Ping(ctx context.Context) error
	}
}

func NewHealthHandler(registry *db.Registry, redis interface {
	Ping(ctx context.Context) error
}) *HealthHandler {
	return &HealthHandler{registry: registry, redis: redis}
}

// Liveness — always 200 if the process is running.
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Readiness — pings all consumer DBs and Redis.
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
	defer cancel()

	issues := fiber.Map{}

	// Ping Redis
	if err := h.redis.Ping(ctx); err != nil {
		issues["redis"] = err.Error()
	}

	// Ping all consumer DB pools
	for id, err := range h.registry.Ping(ctx) {
		if err != nil {
			issues["db:"+id] = err.Error()
		}
	}

	if len(issues) > 0 {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "degraded",
			"issues": issues,
		})
	}
	return c.JSON(fiber.Map{"status": "ready"})
}
