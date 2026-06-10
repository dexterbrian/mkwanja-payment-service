package router

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"mkwanja-payment-svc/internal/config"
	"mkwanja-payment-svc/internal/db"
	"mkwanja-payment-svc/internal/handler"
	"mkwanja-payment-svc/internal/middleware"
	dbgen "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/repository"
	"mkwanja-payment-svc/internal/service"
)

// Dependencies holds all router dependencies.
type Dependencies struct {
	HealthHandler    *handler.HealthHandler
	ConsumerRegistry *config.ConsumerRegistry
	DBRegistry       *db.Registry
	EncryptKey       []byte
	Logger           *slog.Logger
}

// Setup registers all routes and middleware on the given Fiber app.
func Setup(app *fiber.App, deps Dependencies) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	// Global middleware (order matters — first registered = outermost)
	app.Use(middleware.Recovery())
	app.Use(middleware.Logger())

	// Health probes — no auth required
	app.Get("/health", deps.HealthHandler.Liveness)
	app.Get("/health/ready", deps.HealthHandler.Readiness)

	// Consumer API — requires consumer auth + DB resolution
	api := app.Group("/v1", middleware.Consumer(deps.DBRegistry, deps.ConsumerRegistry))

	// Business handler with lazy service creation per request
	bizHandler := &businessHandlerAdapter{
		encryptKey: deps.EncryptKey,
		logger:     deps.Logger,
	}

	// Business CRUD
	api.Post("/businesses", middleware.Idempotency(), bizHandler.registerBusiness)
	api.Post("/businesses/test-credentials", middleware.Idempotency(), bizHandler.testCredentials)
	api.Put("/businesses/:id/credentials", middleware.Idempotency(), bizHandler.updateCredentials)
	api.Delete("/businesses/:id", bizHandler.deactivateBusiness)
	api.Get("/businesses/:id", bizHandler.getBusiness)
	api.Get("/businesses", bizHandler.listBusinesses)
}

// businessHandlerAdapter creates the service per-request from the pool in context.
type businessHandlerAdapter struct {
	encryptKey []byte
	logger     *slog.Logger
}

func (a *businessHandlerAdapter) serviceFromCtx(c *fiber.Ctx) *service.BusinessService {
	pool, ok := c.Locals("db_pool").(*pgxpool.Pool)
	if !ok {
		return nil
	}
	stdlibDB := stdlib.OpenDBFromPool(pool)
	repo := repository.NewPgxBusinessRepo(dbgen.New(stdlibDB))
	return service.NewBusinessService(repo, a.encryptKey, a.logger)
}

func (a *businessHandlerAdapter) registerBusiness(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewBusinessHandler(svc, a.logger)
	return h.RegisterBusiness(c)
}

func (a *businessHandlerAdapter) testCredentials(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewBusinessHandler(svc, a.logger)
	return h.TestCredentials(c)
}

func (a *businessHandlerAdapter) updateCredentials(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewBusinessHandler(svc, a.logger)
	return h.UpdateCredentials(c)
}

func (a *businessHandlerAdapter) deactivateBusiness(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewBusinessHandler(svc, a.logger)
	return h.DeactivateBusiness(c)
}

func (a *businessHandlerAdapter) getBusiness(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewBusinessHandler(svc, a.logger)
	return h.GetBusiness(c)
}

func (a *businessHandlerAdapter) listBusinesses(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewBusinessHandler(svc, a.logger)
	return h.ListBusinesses(c)
}
