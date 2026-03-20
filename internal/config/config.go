package config

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// ConsumerConfig holds per-consumer-app settings loaded from env vars.
type ConsumerConfig struct {
	ID          string
	SecretHash  string // bcrypt hash derived from plaintext secret at startup
	CallbackURL string
	DatabaseURL string
}

// Config is the top-level application configuration.
type Config struct {
	Port        string `envconfig:"PORT" default:"8080"`
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	RedisURL    string `envconfig:"REDIS_URL" required:"true"`

	// 32-byte hex key: openssl rand -hex 32
	CredentialEncryptionKey string `envconfig:"CREDENTIAL_ENCRYPTION_KEY" required:"true"`

	DarajaBaseURL     string `envconfig:"DARAJA_BASE_URL" required:"true"`
	DarajaCallbackURL string `envconfig:"DARAJA_CALLBACK_URL" required:"true"`

	Consumers []ConsumerConfig
}

// Load reads configuration from environment variables.
// Consumer configs are loaded from CONSUMER_<ID>_* env vars.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                    getEnv("PORT", "8080"),
		Environment:             getEnv("ENVIRONMENT", "development"),
		RedisURL:                mustEnv("REDIS_URL"),
		CredentialEncryptionKey: mustEnv("CREDENTIAL_ENCRYPTION_KEY"),
		DarajaBaseURL:           mustEnv("DARAJA_BASE_URL"),
		DarajaCallbackURL:       mustEnv("DARAJA_CALLBACK_URL"),
	}

	cfg.Consumers = loadConsumers()
	if len(cfg.Consumers) == 0 {
		return nil, fmt.Errorf("no consumers configured — set CONSUMER_<ID>_* env vars")
	}

	return cfg, nil
}

// loadConsumers scans env for CONSUMER_*_ID vars and builds ConsumerConfig slices.
func loadConsumers() []ConsumerConfig {
	seen := make(map[string]bool)
	var consumers []ConsumerConfig

	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "CONSUMER_") {
			continue
		}
		parts := strings.SplitN(e, "_", 3) // CONSUMER_<NAME>_KEY
		if len(parts) < 3 {
			continue
		}
		name := parts[1] // e.g. "EAZIBIZ"
		if seen[name] {
			continue
		}
		seen[name] = true

		id := os.Getenv(fmt.Sprintf("CONSUMER_%s_ID", name))
		secret := os.Getenv(fmt.Sprintf("CONSUMER_%s_SECRET", name))
		callbackURL := os.Getenv(fmt.Sprintf("CONSUMER_%s_CALLBACK_URL", name))
		databaseURL := os.Getenv(fmt.Sprintf("CONSUMER_%s_DATABASE_URL", name))

		if id == "" || secret == "" || databaseURL == "" {
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			continue
		}

		consumers = append(consumers, ConsumerConfig{
			ID:          id,
			SecretHash:  string(hash),
			CallbackURL: callbackURL,
			DatabaseURL: databaseURL,
		})
	}

	return consumers
}

// ConsumerRegistry validates consumer ID + secret pairs.
type ConsumerRegistry struct {
	consumers map[string]ConsumerConfig
}

func NewConsumerRegistry(consumers []ConsumerConfig) *ConsumerRegistry {
	m := make(map[string]ConsumerConfig, len(consumers))
	for _, c := range consumers {
		m[c.ID] = c
	}
	return &ConsumerRegistry{consumers: m}
}

// Validate returns true if the ID and plaintext secret match a registered consumer.
func (r *ConsumerRegistry) Validate(id, secret string) bool {
	c, ok := r.consumers[id]
	if !ok {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(c.SecretHash), []byte(secret)) == nil
}

// Get returns the ConsumerConfig for the given ID.
func (r *ConsumerRegistry) Get(id string) (ConsumerConfig, bool) {
	c, ok := r.consumers[id]
	return c, ok
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}
