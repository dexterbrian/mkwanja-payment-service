package router

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"mkwanja-payment-svc/internal/config"
	"mkwanja-payment-svc/internal/db"
	dbgen "mkwanja-payment-svc/internal/db/generated"
	"mkwanja-payment-svc/internal/handler"
	"mkwanja-payment-svc/internal/middleware"
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

	// Client handler with lazy service creation per request
	clientHandler := &clientHandlerAdapter{
		encryptKey: deps.EncryptKey,
		logger:     deps.Logger,
	}

	// Client CRUD
	api.Post("/clients", middleware.Idempotency(), clientHandler.registerClient)
	api.Post("/clients/test-credentials", middleware.Idempotency(), clientHandler.testCredentials)
	api.Put("/clients/:client_id/credentials", middleware.Idempotency(), clientHandler.updateCredentials)
	api.Delete("/clients/:client_id", clientHandler.deactivateClient)
	api.Get("/clients/:client_id", clientHandler.getClient)
	api.Get("/clients", clientHandler.listClients)
}

// clientHandlerAdapter creates the service per-request from the pool in context.
type clientHandlerAdapter struct {
	encryptKey []byte
	logger     *slog.Logger
}

func (a *clientHandlerAdapter) serviceFromCtx(c *fiber.Ctx) *service.ClientService {
	pool, ok := c.Locals("db_pool").(*pgxpool.Pool)
	if !ok {
		return nil
	}
	stdlibDB := stdlib.OpenDBFromPool(pool)
	repo := repository.NewPgxClientRepo(dbgen.New(stdlibDB))
	return service.NewClientService(repo, a.encryptKey, a.logger)
}

func (a *clientHandlerAdapter) registerClient(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.RegisterClient(c)
}

func (a *clientHandlerAdapter) testCredentials(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.TestCredentials(c)
}

func (a *clientHandlerAdapter) updateCredentials(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.UpdateCredentials(c)
}

func (a *clientHandlerAdapter) deactivateClient(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.DeactivateClient(c)
}

func (a *clientHandlerAdapter) getClient(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.GetClient(c)
}

func (a *clientHandlerAdapter) listClients(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.ListClients(c)
}
