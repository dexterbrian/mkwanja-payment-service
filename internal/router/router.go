package router

import (
	"mkwanja-payment-svc/internal/handler"
	"mkwanja-payment-svc/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// Setup registers all routes on the given Fiber app.
func Setup(app *fiber.App, health *handler.HealthHandler) {
	// Global middleware
	app.Use(middleware.Recovery())
	app.Use(middleware.Logger())

	// Health probes — no auth
	app.Get("/health", health.Liveness)
	app.Get("/health/ready", health.Readiness)
}
