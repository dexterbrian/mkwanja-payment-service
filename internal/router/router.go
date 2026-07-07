package router

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"mkwanja-payment-svc/internal/config"
	"mkwanja-payment-svc/internal/daraja"
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
	DarajaBaseURL    string
	CallbackURL      string
	PaystackBaseURL  string
	WebhookEnqueuer  handler.WebhookEnqueuer
	RedisClient      *redis.Client
	Logger           *slog.Logger
	OperatorClientID string
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
		encryptKey:       deps.EncryptKey,
		logger:           deps.Logger,
		operatorClientID: deps.OperatorClientID,
	}

	// Client CRUD
	api.Post("/clients", middleware.Idempotency(), clientHandler.registerClient)
	api.Post("/clients/test-credentials", middleware.Idempotency(), clientHandler.testCredentials)
	api.Put("/clients/:client_id/credentials", middleware.Idempotency(), clientHandler.updateCredentials)
	api.Delete("/clients/:client_id", clientHandler.deactivateClient)
	api.Get("/clients/:client_id", clientHandler.getClient)
	api.Get("/clients", clientHandler.listClients)

	// Operator admin endpoint
	api.Put("/admin/operator/credentials", middleware.Idempotency(), clientHandler.updateOperatorCredentials)

	// Payment handler with lazy service creation per request
	paymentHandler := &paymentHandlerAdapter{
		encryptKey:    deps.EncryptKey,
		darajaBaseURL: deps.DarajaBaseURL,
		callbackURL:   deps.CallbackURL,
		rdb:           deps.RedisClient,
		logger:        deps.Logger,
	}

	// Payment routes
	api.Post("/payments/stk-push", middleware.Idempotency(), paymentHandler.initiateSTKPush)
	api.Post("/payments/b2c", middleware.Idempotency(), paymentHandler.initiateB2C)
	api.Post("/payments/b2b", middleware.Idempotency(), paymentHandler.initiateB2B)
	api.Get("/payments/:id", paymentHandler.getPayment)
	api.Get("/payments", paymentHandler.listPayments)

	// Paystack routes (bring-your-own-credentials, like Daraja)
	paystackAdapter := &paystackHandlerAdapter{
		encryptKey:      deps.EncryptKey,
		paystackBaseURL: deps.PaystackBaseURL,
		registry:        deps.DBRegistry,
		logger:          deps.Logger,
	}
	api.Put("/clients/:client_id/paystack-credentials", middleware.Idempotency(), paystackAdapter.registerCredentials)
	api.Post("/payments/paystack/initialize", middleware.Idempotency(), paystackAdapter.initiateCharge)

	// Ledger routes
	ledgerHandler := &ledgerHandlerAdapter{logger: deps.Logger}
	api.Get("/ledger", ledgerHandler.listEntries)
	api.Get("/ledger/balance", ledgerHandler.getBalances)
	api.Get("/ledger/trial-balance", ledgerHandler.getTrialBalance)

	// Webhook routes (no consumer auth — outside /v1 group)
	webhookHandler := handler.NewWebhookHandler(deps.WebhookEnqueuer, deps.Logger)
	webhooks := app.Group("/webhooks/mpesa")
	webhooks.Post("/stk/:consumer_id", webhookHandler.HandleSTKCallback)
	webhooks.Post("/b2c/:consumer_id", webhookHandler.HandleB2CCallback)
	webhooks.Post("/b2b/:consumer_id", webhookHandler.HandleB2BCallback)
	webhooks.Post("/c2b/:consumer_id/confirm", webhookHandler.HandleC2BConfirmation)
	webhooks.Post("/c2b/:consumer_id/validate", webhookHandler.HandleC2BValidation)

	// Paystack webhook — signature-verified per business inside the service
	app.Post("/webhooks/paystack/:consumer_id", paystackAdapter.handleWebhook)
}

// clientHandlerAdapter creates the service per-request from the pool in context.
type clientHandlerAdapter struct {
	encryptKey       []byte
	logger           *slog.Logger
	operatorClientID string
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

func (a *clientHandlerAdapter) updateOperatorCredentials(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewClientHandler(svc, a.logger)
	return h.UpdateOperatorCredentials(c, a.operatorClientID)
}

// paymentHandlerAdapter creates the payment service per-request from the pool in context.
type paymentHandlerAdapter struct {
	encryptKey    []byte
	darajaBaseURL string
	callbackURL   string
	rdb           *redis.Client
	logger        *slog.Logger
}

func (a *paymentHandlerAdapter) serviceFromCtx(c *fiber.Ctx) *service.PaymentService {
	pool, ok := c.Locals("db_pool").(*pgxpool.Pool)
	if !ok {
		return nil
	}
	stdlibDB := stdlib.OpenDBFromPool(pool)
	q := dbgen.New(stdlibDB)
	paymentRepo := repository.NewPgxPaymentRepo(q)
	clientRepo := repository.NewPgxClientRepo(q)
	journalRepo := repository.NewPgxJournalRepo(q, stdlibDB)
	tokenCache := daraja.NewRedisTokenCache(a.rdb)
	return service.NewPaymentService(paymentRepo, clientRepo, journalRepo, a.encryptKey, a.darajaBaseURL, a.callbackURL, tokenCache, a.rdb, a.logger)
}

func (a *paymentHandlerAdapter) initiateSTKPush(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaymentHandler(svc, a.logger)
	return h.InitiateSTKPush(c)
}

func (a *paymentHandlerAdapter) initiateB2C(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaymentHandler(svc, a.logger)
	return h.InitiateB2C(c)
}

func (a *paymentHandlerAdapter) initiateB2B(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaymentHandler(svc, a.logger)
	return h.InitiateB2B(c)
}

func (a *paymentHandlerAdapter) getPayment(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaymentHandler(svc, a.logger)
	return h.GetPayment(c)
}

func (a *paymentHandlerAdapter) listPayments(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaymentHandler(svc, a.logger)
	return h.ListPayments(c)
}

// paystackHandlerAdapter creates the Paystack service per-request. Consumer
// API routes take the pool from context; the webhook route (no consumer
// auth) resolves it from the registry by the :consumer_id path param.
type paystackHandlerAdapter struct {
	encryptKey      []byte
	paystackBaseURL string
	registry        *db.Registry
	logger          *slog.Logger
}

func (a *paystackHandlerAdapter) serviceFromPool(pool *pgxpool.Pool) *service.PaystackService {
	stdlibDB := stdlib.OpenDBFromPool(pool)
	q := dbgen.New(stdlibDB)
	paymentRepo := repository.NewPgxPaymentRepo(q)
	paystackRepo := repository.NewPgxPaystackRepo(q)
	journalRepo := repository.NewPgxJournalRepo(q, stdlibDB)
	clientRepo := repository.NewPgxClientRepo(q)
	// The completer only needs the repos — no Daraja config.
	completer := service.NewPaymentCompleter(paymentRepo, clientRepo, journalRepo, a.encryptKey, a.logger)
	return service.NewPaystackService(paymentRepo, paystackRepo, completer, a.encryptKey, a.paystackBaseURL, a.logger)
}

func (a *paystackHandlerAdapter) serviceFromCtx(c *fiber.Ctx) *service.PaystackService {
	pool, ok := c.Locals("db_pool").(*pgxpool.Pool)
	if !ok {
		return nil
	}
	return a.serviceFromPool(pool)
}

func (a *paystackHandlerAdapter) registerCredentials(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaystackHandler(svc, a.logger)
	return h.RegisterCredentials(c)
}

func (a *paystackHandlerAdapter) initiateCharge(c *fiber.Ctx) error {
	svc := a.serviceFromCtx(c)
	h := handler.NewPaystackHandler(svc, a.logger)
	return h.InitiateCharge(c)
}

func (a *paystackHandlerAdapter) handleWebhook(c *fiber.Ctx) error {
	pool, err := a.registry.Get(c.Params("consumer_id"))
	if err != nil {
		a.logger.Error("paystack webhook for unknown consumer", "consumer_id", c.Params("consumer_id"))
		// Still 200 — never invite provider retries against a bad URL.
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	}
	svc := a.serviceFromPool(pool)
	h := handler.NewPaystackHandler(svc, a.logger)
	return h.HandleWebhook(c)
}

// ledgerHandlerAdapter creates the journal service per-request from the pool in context.
type ledgerHandlerAdapter struct {
	logger *slog.Logger
}

func (a *ledgerHandlerAdapter) handlerFromCtx(c *fiber.Ctx) *handler.LedgerHandler {
	pool, ok := c.Locals("db_pool").(*pgxpool.Pool)
	if !ok {
		return nil
	}
	stdlibDB := stdlib.OpenDBFromPool(pool)
	q := dbgen.New(stdlibDB)
	journalRepo := repository.NewPgxJournalRepo(q, stdlibDB)
	svc := service.NewJournalService(journalRepo, a.logger)
	return handler.NewLedgerHandler(svc, a.logger)
}

func (a *ledgerHandlerAdapter) listEntries(c *fiber.Ctx) error {
	h := a.handlerFromCtx(c)
	return h.ListEntries(c)
}

func (a *ledgerHandlerAdapter) getBalances(c *fiber.Ctx) error {
	h := a.handlerFromCtx(c)
	return h.GetBalances(c)
}

func (a *ledgerHandlerAdapter) getTrialBalance(c *fiber.Ctx) error {
	h := a.handlerFromCtx(c)
	return h.GetTrialBalance(c)
}
