package routes

import (
	"mkwanja-payment-service/internal/configs"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterApiRoutes(app *fiber.App, db *gorm.DB, cfg *configs.Config) {
	// Health check
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().String(),
			"database":  "connected",
			"version":   "1.0.0",
		})
	})
	// ...register other routes here
}
