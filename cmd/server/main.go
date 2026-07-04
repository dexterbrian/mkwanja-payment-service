package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"mkwanja-payment-svc/internal/config"
	"mkwanja-payment-svc/internal/crypto"
	"mkwanja-payment-svc/internal/db"
	"mkwanja-payment-svc/internal/handler"
	"mkwanja-payment-svc/internal/repository"
	"mkwanja-payment-svc/internal/router"
	"mkwanja-payment-svc/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	// Redis client
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	// Parse encryption key
	encryptKey, err := crypto.ParseHexKey(cfg.CredentialEncryptionKey)
	if err != nil {
		slog.Error("invalid CREDENTIAL_ENCRYPTION_KEY", "error", err)
		os.Exit(1)
	}

	// DB registry — register one pool per consumer, then migrate and optionally seed operator
	registry := db.NewRegistry()
	ctx := context.Background()
	for _, c := range cfg.Consumers {
		if err := registry.Register(ctx, c.ID, c.DatabaseURL); err != nil {
			slog.Error("db registry failed", "consumer", c.ID, "error", err)
			os.Exit(1)
		}
		slog.Info("consumer db registered", "consumer", c.ID)

		pool, err := registry.Get(c.ID)
		if err != nil {
			slog.Error("db pool unavailable after registration", "consumer", c.ID, "error", err)
			os.Exit(1)
		}

		if err := db.MigrateConsumerDB(ctx, pool); err != nil {
			slog.Error("db migration failed", "consumer", c.ID, "error", err)
			os.Exit(1)
		}
		slog.Info("consumer db migrated", "consumer", c.ID)
	}

	// Seed operator client if configured
	if cfg.OperatorClientID != "" {
		operatorPool, err := registry.Get(cfg.OperatorConsumerID)
		if err != nil {
			slog.Error("operator consumer db not found", "consumer", cfg.OperatorConsumerID, "error", err)
			os.Exit(1)
		}
		repo := repository.NewPgxClientRepoFromPool(operatorPool)
		svc := service.NewClientService(repo, encryptKey, slog.Default())

		if _, err := svc.EnsureOperatorClient(ctx, cfg.OperatorClientID, cfg.OperatorClientName); err != nil {
			slog.Error("operator client seeding failed", "error", err)
			os.Exit(1)
		}
	}

	// Consumer registry
	consumerRegistry := config.NewConsumerRegistry(cfg.Consumers)

	// Build app
	app := fiber.New(fiber.Config{
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          30 * time.Second,
		DisableStartupMessage: false,
	})

	// Handlers
	redisWrapper := &redisPingWrapper{rdb}
	healthHandler := handler.NewHealthHandler(registry, redisWrapper)

	// Routes
	router.Setup(app, router.Dependencies{
		HealthHandler:    healthHandler,
		ConsumerRegistry: consumerRegistry,
		DBRegistry:       registry,
		EncryptKey:       encryptKey,
		DarajaBaseURL:    cfg.DarajaBaseURL,
		CallbackURL:      cfg.DarajaCallbackURL,
		RedisClient:      rdb,
		Logger:           slog.Default(),
		OperatorClientID: cfg.OperatorClientID,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("shutting down")
		_ = app.ShutdownWithTimeout(5 * time.Second)
	}()

	addr := ":" + cfg.Port
	slog.Info("starting server", "addr", addr, "env", cfg.Environment)
	if err := app.Listen(addr); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// redisPingWrapper adapts *redis.Client to the interface expected by HealthHandler.
type redisPingWrapper struct{ c *redis.Client }

func (r *redisPingWrapper) Ping(ctx context.Context) error {
	return r.c.Ping(ctx).Err()
}
