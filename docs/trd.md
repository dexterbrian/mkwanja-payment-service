# TRD: `mkwanja-payment-svc`

**Version:** 0.3.0
**Status:** Ready for build
**Depends on:** PRD-mkwanja-payment-svc.md (v0.3.0)
**Runtime:** Go 1.23+, PostgreSQL 15+ (Supabase, one project per consumer), Redis (Upstash)

---

## 1. Agent Instructions

You are building `mkwanja-payment-svc`, a Go financial microservice. Read the PRD and this entire TRD before writing any code. This is a financial service — correctness is more important than cleverness.

Before writing any code, internalise these facts:
- Each business has its own Daraja credentials stored encrypted in the database
- Each consumer app (eazibiz, mkwanja, kanisa) has its own isolated Postgres database
- All DB writes are routed to the correct consumer database via a registry keyed on `X-Consumer-ID`
- Daraja OAuth tokens are cached in Redis per business: key = `daraja_token:{consumer_id}:{business_id}`
- The `journal` table is append-only — `UPDATE` and `DELETE` are blocked at the Postgres rule level
- Amounts are always integer cents — never float64 for money
- Every journal write must verify debits = credits before committing

Task execution order for any feature:
1. Write SQL migration
2. Write sqlc query file
3. Write domain type
4. Write repository interface then implementation
5. Write service layer
6. Write HTTP handler
7. Register route
8. Write table-driven tests

---

## 2. Project Structure

```
mkwanja-payment-svc/
├── cmd/server/main.go
├── internal/
│   ├── config/config.go
│   ├── db/
│   │   ├── registry.go                  ← one pgx pool per consumer
│   │   ├── migrations/consumer/         ← same migrations applied to each consumer DB
│   │   │   ├── 001_businesses.sql
│   │   │   ├── 002_credentials.sql
│   │   │   ├── 003_payments.sql
│   │   │   ├── 004_journal_accounts.sql
│   │   │   └── 005_journal.sql
│   │   ├── queries/
│   │   │   ├── businesses.sql
│   │   │   ├── credentials.sql
│   │   │   ├── payments.sql
│   │   │   ├── journal_accounts.sql
│   │   │   └── journal.sql
│   │   └── generated/                   ← sqlc output — never edit manually
│   ├── domain/
│   │   ├── business.go
│   │   ├── payment.go
│   │   └── journal.go
│   ├── repository/
│   │   ├── business_repo.go
│   │   ├── payment_repo.go
│   │   └── journal_repo.go
│   ├── service/
│   │   ├── payment_service.go
│   │   ├── journal_service.go
│   │   └── reconciliation_service.go
│   ├── daraja/
│   │   ├── client.go                    ← per-business client, built per request
│   │   ├── auth.go                      ← per-business OAuth token cache
│   │   ├── stk.go
│   │   ├── b2c.go
│   │   ├── b2b.go
│   │   ├── c2b.go
│   │   └── webhook.go
│   ├── crypto/credentials.go            ← AES-256-GCM encrypt/decrypt
│   ├── handler/
│   │   ├── business_handler.go
│   │   ├── payment_handler.go
│   │   ├── ledger_handler.go
│   │   ├── webhook_handler.go
│   │   └── health_handler.go
│   ├── middleware/
│   │   ├── auth.go                      ← X-Service-Secret validation
│   │   ├── consumer.go                  ← resolve consumer, attach DB pool to ctx
│   │   ├── idempotency.go
│   │   ├── logger.go
│   │   └── recovery.go
│   ├── queue/
│   │   ├── worker.go                    ← asynq webhook processor
│   │   └── jobs.go
│   └── router/router.go
├── sqlc.yaml
├── Makefile
├── Dockerfile
└── go.mod
```

---

## 3. Dependencies

```go
require (
    github.com/gofiber/fiber/v2          v2.52.0
    github.com/jackc/pgx/v5              v5.7.0
    github.com/hibiken/asynq             v0.24.0
    github.com/kelseyhightower/envconfig v1.4.0
    github.com/pressly/goose/v3          v3.22.0
    github.com/google/uuid               v1.6.0
    golang.org/x/crypto                  v0.28.0
)
```

No ORMs. No raw `database/sql` queries. `sqlc` only.

---

## 4. Configuration

```go
// internal/config/config.go
package config

type ConsumerConfig struct {
    ID          string
    SecretHash  string // bcrypt hash of the plaintext secret — hash at startup
    CallbackURL string
    DatabaseURL string // Supabase direct connection string for this consumer
}

type Config struct {
    Port        string `envconfig:"PORT" default:"8080"`
    Environment string `envconfig:"ENVIRONMENT" default:"development"`
    RedisURL    string `envconfig:"REDIS_URL" required:"true"`

    // 32-byte hex key: openssl rand -hex 32
    CredentialEncryptionKey string `envconfig:"CREDENTIAL_ENCRYPTION_KEY" required:"true"`

    DarajaBaseURL     string `envconfig:"DARAJA_BASE_URL" required:"true"`
    DarajaCallbackURL string `envconfig:"DARAJA_CALLBACK_URL" required:"true"`

    Consumers []ConsumerConfig // loaded from CONSUMER_* env vars in Load()
}
```

```bash
# .env.example
PORT=8080
ENVIRONMENT=development
REDIS_URL=rediss://default:[token]@[host].upstash.io:6379
CREDENTIAL_ENCRYPTION_KEY=   # openssl rand -hex 32
DARAJA_BASE_URL=https://sandbox.safaricom.co.ke
DARAJA_CALLBACK_URL=https://[domain]/webhooks/mpesa/stk

CONSUMER_EAZIBIZ_ID=eazibiz
CONSUMER_EAZIBIZ_SECRET=     # plaintext, hashed at startup
CONSUMER_EAZIBIZ_CALLBACK_URL=https://[supabase-project].supabase.co/functions/v1/payment-callback
CONSUMER_EAZIBIZ_DATABASE_URL=postgresql://postgres:[pw]@db.[project].supabase.co:5432/postgres

CONSUMER_MKWANJA_ID=mkwanja
CONSUMER_MKWANJA_SECRET=
CONSUMER_MKWANJA_CALLBACK_URL=
CONSUMER_MKWANJA_DATABASE_URL=
```

---

## 5. Database Registry

```go
// internal/db/registry.go
package db

import (
    "context"
    "fmt"
    "sync"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Registry struct {
    mu    sync.RWMutex
    pools map[string]*pgxpool.Pool
}

func NewRegistry() *Registry {
    return &Registry{pools: make(map[string]*pgxpool.Pool)}
}

func (r *Registry) Register(ctx context.Context, consumerID, databaseURL string) error {
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil {
        return fmt.Errorf("connect %s: %w", consumerID, err)
    }
    if err := pool.Ping(ctx); err != nil {
        return fmt.Errorf("ping %s: %w", consumerID, err)
    }
    r.mu.Lock()
    r.pools[consumerID] = pool
    r.mu.Unlock()
    return nil
}

func (r *Registry) Get(consumerID string) (*pgxpool.Pool, error) {
    r.mu.RLock()
    pool, ok := r.pools[consumerID]
    r.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("no db for consumer: %s", consumerID)
    }
    return pool, nil
}
```

Consumer middleware resolves the pool and attaches it to the Fiber context:

```go
// internal/middleware/consumer.go
func ResolveConsumer(registry *db.Registry, consumers ConsumerRegistry) fiber.Handler {
    return func(c *fiber.Ctx) error {
        id := c.Get("X-Consumer-ID")
        secret := c.Get("X-Service-Secret")
        if !consumers.Validate(id, secret) {
            return c.Status(401).JSON(errResp("UNAUTHORIZED", "Invalid credentials"))
        }
        pool, err := registry.Get(id)
        if err != nil {
            return c.Status(500).JSON(errResp("INTERNAL", "Database unavailable"))
        }
        c.Locals("consumer_id", id)
        c.Locals("db_pool", pool)
        return c.Next()
    }
}
```

---

## 6. Database Schema (each consumer database)

### `001_businesses.sql`

```sql
CREATE TABLE businesses (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    external_id TEXT NOT NULL UNIQUE,  -- org/user ID from the consumer app
    name        TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_businesses_external_id ON businesses(external_id);
```

### `002_credentials.sql`

```sql
CREATE TABLE business_credentials (
    id                          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    business_id                 TEXT NOT NULL REFERENCES businesses(id),
    shortcode                   TEXT NOT NULL,
    consumer_key_encrypted      TEXT NOT NULL,
    consumer_secret_encrypted   TEXT NOT NULL,
    passkey_encrypted           TEXT NOT NULL,
    initiator_name              TEXT,                    -- for B2C/B2B
    security_credential_encrypted TEXT,                  -- for B2C/B2B
    is_active                   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(business_id, is_active) WHERE (is_active = TRUE)
);
```

### `003_payments.sql`

```sql
CREATE TYPE payment_provider   AS ENUM ('mpesa', 'stripe');
CREATE TYPE payment_direction  AS ENUM ('inbound', 'outbound');
CREATE TYPE payment_type       AS ENUM ('stk_push', 'b2c', 'b2b', 'c2b');
CREATE TYPE payment_status     AS ENUM (
    'pending', 'processing', 'completed', 'failed', 'cancelled', 'reversed'
);

CREATE TABLE payments (
    id                  TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    business_id         TEXT NOT NULL REFERENCES businesses(id),
    idempotency_key     TEXT NOT NULL,
    provider            payment_provider NOT NULL DEFAULT 'mpesa',
    payment_type        payment_type NOT NULL,
    direction           payment_direction NOT NULL,
    status              payment_status NOT NULL DEFAULT 'pending',
    amount_cents        BIGINT NOT NULL CHECK (amount_cents > 0),
    currency            TEXT NOT NULL DEFAULT 'KES',
    phone_number        TEXT,
    receiver_shortcode  TEXT,              -- B2B: receiving till/paybill
    reference           TEXT NOT NULL,
    description         TEXT,
    provider_request_id TEXT,
    provider_tx_id      TEXT,
    provider_receipt    TEXT,
    provider_raw        JSONB,
    callback_delivered  BOOLEAN NOT NULL DEFAULT FALSE,
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    UNIQUE(business_id, idempotency_key)
);

CREATE TABLE payment_events (
    id          BIGSERIAL PRIMARY KEY,
    payment_id  TEXT NOT NULL REFERENCES payments(id),
    from_status payment_status,
    to_status   payment_status NOT NULL,
    reason      TEXT,
    raw         JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_payment_events AS ON UPDATE TO payment_events DO INSTEAD NOTHING;
CREATE RULE no_delete_payment_events AS ON DELETE TO payment_events DO INSTEAD NOTHING;

CREATE INDEX idx_payments_business_id   ON payments(business_id);
CREATE INDEX idx_payments_status        ON payments(status);
CREATE INDEX idx_payments_provider_tx   ON payments(provider_tx_id);
CREATE INDEX idx_payments_created_at    ON payments(created_at DESC);
```

### `004_journal_accounts.sql`

```sql
CREATE TYPE account_type    AS ENUM ('asset', 'liability', 'revenue', 'expense', 'equity');
CREATE TYPE normal_balance  AS ENUM ('debit', 'credit');

CREATE TABLE journal_accounts (
    id             TEXT NOT NULL,
    business_id    TEXT NOT NULL REFERENCES businesses(id),
    name           TEXT NOT NULL,
    account_type   account_type NOT NULL,
    normal_balance normal_balance NOT NULL,
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (business_id, id)
);
```

### `005_journal.sql`

```sql
CREATE TYPE entry_type AS ENUM ('debit', 'credit');

CREATE TABLE journal (
    id            BIGSERIAL PRIMARY KEY,
    business_id   TEXT NOT NULL,
    payment_id    TEXT NOT NULL REFERENCES payments(id),
    account_id    TEXT NOT NULL,
    entry_type    entry_type NOT NULL,
    amount_cents  BIGINT NOT NULL CHECK (amount_cents > 0),
    currency      TEXT NOT NULL DEFAULT 'KES',
    description   TEXT NOT NULL,
    reversal_of   BIGINT REFERENCES journal(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (business_id, account_id)
        REFERENCES journal_accounts(business_id, id)
);

CREATE RULE no_update_journal AS ON UPDATE TO journal DO INSTEAD NOTHING;
CREATE RULE no_delete_journal AS ON DELETE TO journal DO INSTEAD NOTHING;

CREATE INDEX idx_journal_business_id ON journal(business_id);
CREATE INDEX idx_journal_payment_id  ON journal(payment_id);
CREATE INDEX idx_journal_account_id  ON journal(account_id);
CREATE INDEX idx_journal_created_at  ON journal(created_at DESC);

CREATE VIEW account_balances AS
SELECT
    business_id,
    account_id,
    SUM(CASE WHEN entry_type = 'debit'  THEN amount_cents ELSE 0 END) AS total_debits_cents,
    SUM(CASE WHEN entry_type = 'credit' THEN amount_cents ELSE 0 END) AS total_credits_cents,
    SUM(CASE WHEN entry_type = 'debit'  THEN amount_cents ELSE -amount_cents END) AS net_cents
FROM journal
GROUP BY business_id, account_id;
```

---

## 7. Credential Encryption

```go
// internal/crypto/credentials.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"
)

func Encrypt(key []byte, plaintext string) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("new cipher: %w", err)
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("new gcm: %w", err)
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("generate nonce: %w", err)
    }
    return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

func Decrypt(key []byte, encoded string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", fmt.Errorf("base64 decode: %w", err)
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("new cipher: %w", err)
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("new gcm: %w", err)
    }
    ns := gcm.NonceSize()
    if len(data) < ns {
        return "", fmt.Errorf("ciphertext too short")
    }
    plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
    if err != nil {
        return "", fmt.Errorf("decrypt: %w", err)
    }
    return string(plaintext), nil
}
```

---

## 8. Daraja Client (Per-Business)

```go
// internal/daraja/client.go
package daraja

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

type Credentials struct {
    Shortcode          string
    ConsumerKey        string
    ConsumerSecret     string
    Passkey            string
    InitiatorName      string
    SecurityCredential string
}

type Client struct {
    baseURL     string
    callbackURL string
    creds       Credentials
    tokenCache  TokenCache
    cacheKey    string    // "daraja_token:{consumer_id}:{business_id}"
    http        *http.Client
}

type TokenCache interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string, ttl time.Duration) error
}

func NewClient(baseURL, callbackURL string, creds Credentials, cache TokenCache, cacheKey string) *Client {
    return &Client{
        baseURL:     baseURL,
        callbackURL: callbackURL,
        creds:       creds,
        tokenCache:  cache,
        cacheKey:    fmt.Sprintf("daraja_token:%s", cacheKey),
        http:        &http.Client{Timeout: 30 * time.Second},
    }
}
```

```go
// internal/daraja/b2b.go
type B2BRequest struct {
    Amount           int64  // whole KES
    ReceiverShortcode string
    CommandID        string // "BusinessPayBill" or "BusinessBuyGoods"
    AccountReference string
    Remarks          string
}

type B2BResponse struct {
    ConversationID          string
    OriginatorConversationID string
    ResponseDescription     string
}

func (c *Client) InitiateB2B(ctx context.Context, req B2BRequest) (*B2BResponse, error) {
    // POST /mpesa/b2b/v1/paymentrequest
    // Uses c.creds.InitiatorName + c.creds.SecurityCredential
    // PartyA = c.creds.Shortcode
    // PartyB = req.ReceiverShortcode
}
```

---

## 9. STK Push Service Flow

```go
// internal/service/payment_service.go

func (s *PaymentService) InitiateSTKPush(
    ctx context.Context,
    consumerID string,
    req STKPushRequest,
) (*domain.Payment, error) {

    pool, err := s.registry.Get(consumerID)
    if err != nil {
        return nil, err
    }
    repo := s.newRepo(pool)

    // 1. Idempotency check
    if existing, err := repo.GetByIdempotencyKey(ctx, req.BusinessID, req.IdempotencyKey); err == nil {
        return existing, nil
    }

    // 2. Create payment in 'pending'
    payment, err := repo.CreatePayment(ctx, createPaymentParams(req, "stk_push", "inbound"))
    if err != nil {
        return nil, fmt.Errorf("create payment: %w", err)
    }

    // 3. Load + decrypt credentials
    creds, err := s.loadCredentials(ctx, pool, req.BusinessID)
    if err != nil {
        return nil, fmt.Errorf("load credentials: %w", err)
    }

    // 4. Build per-business Daraja client
    cacheKey := fmt.Sprintf("%s:%s", consumerID, req.BusinessID)
    client := daraja.NewClient(s.cfg.DarajaBaseURL, s.cfg.DarajaCallbackURL, creds, s.tokenCache, cacheKey)

    // 5. Call Daraja
    resp, err := client.InitiateSTKPush(ctx, daraja.STKPushReq{
        Amount:           req.AmountCents / 100,
        PhoneNumber:      normalisePhone(req.PhoneNumber),
        AccountReference: req.Reference,
        TransactionDesc:  req.Description,
    })
    if err != nil {
        _ = repo.UpdateStatus(ctx, payment.ID, "failed")
        s.logger.Error("stk push failed", "payment_id", payment.ID, "error", err)
        return nil, fmt.Errorf("daraja stk push: %w", err)
    }

    // 6. Cache webhook routing key in Redis
    routeKey := fmt.Sprintf("stk:%s", resp.CheckoutRequestID)
    routeVal := fmt.Sprintf("%s:%s:%s", consumerID, req.BusinessID, payment.ID)
    s.redis.Set(ctx, routeKey, routeVal, 30*time.Minute)

    payment, _ = repo.UpdateProviderTxID(ctx, payment.ID, resp.CheckoutRequestID)
    s.logger.Info("stk push initiated", "payment_id", payment.ID, "checkout_id", resp.CheckoutRequestID)
    return payment, nil
}
```

---

## 10. Webhook Handler

```go
// internal/handler/webhook_handler.go

func (h *WebhookHandler) HandleSTKCallback(c *fiber.Ctx) error {
    raw := make([]byte, len(c.Body()))
    copy(raw, c.Body())

    if err := h.queue.Enqueue(c.Context(), "webhook:stk", raw); err != nil {
        h.logger.Error("enqueue webhook failed", "error", err)
        // Return 200 regardless — reconciliation recovers unprocessed webhooks
    }

    return c.Status(200).JSON(fiber.Map{"ResultCode": "00000000", "ResultDesc": "success"})
}
```

```go
// internal/queue/worker.go

func (w *Worker) ProcessSTKWebhook(ctx context.Context, task *asynq.Task) error {
    var body daraja.STKCallbackBody
    if err := json.Unmarshal(task.Payload(), &body); err != nil {
        return fmt.Errorf("unmarshal: %w", err)
    }

    // Route: look up Redis key set at initiation
    routeVal, err := w.redis.Get(ctx, fmt.Sprintf("stk:%s", body.CheckoutRequestID)).Result()
    if err != nil {
        return fmt.Errorf("route not found for %s: %w", body.CheckoutRequestID, err)
    }

    parts := strings.SplitN(routeVal, ":", 3)
    consumerID, businessID, paymentID := parts[0], parts[1], parts[2]

    pool, _ := w.registry.Get(consumerID)
    repo := w.newRepo(pool)

    if body.ResultCode == 0 {
        receipt := body.CallbackMetadata.ExtractReceipt()
        amountCents := body.CallbackMetadata.ExtractAmountCents()

        if err := repo.CompletePayment(ctx, paymentID, receipt); err != nil {
            return fmt.Errorf("complete payment: %w", err)
        }
        if err := w.journalSvc.WriteInboundEntries(ctx, pool, businessID, paymentID, amountCents); err != nil {
            return fmt.Errorf("write journal: %w", err)
        }
    } else {
        if err := repo.FailPayment(ctx, paymentID, body.ResultDesc); err != nil {
            return fmt.Errorf("fail payment: %w", err)
        }
    }

    return w.deliverCallback(ctx, consumerID, paymentID)
}
```

---

## 11. Journal Service

```go
// internal/service/journal_service.go

var defaultAccounts = []SeedAccount{
    {"mpesa.till",             "M-PESA till",          "asset",     "debit"},
    {"revenue.sales",          "Sales revenue",         "revenue",   "credit"},
    {"revenue.other",          "Other revenue",         "revenue",   "credit"},
    {"expense.cogs",           "Cost of goods sold",    "expense",   "debit"},
    {"expense.operations",     "Operating expenses",    "expense",   "debit"},
    {"liability.vat_payable",  "VAT payable",           "liability", "credit"},
    {"liability.pending",      "Pending payments",      "liability", "credit"},
    {"fees.mpesa",             "M-PESA charges",        "expense",   "debit"},
}

// WriteInboundEntries writes balanced journal for a confirmed STK Push.
// amountCents is the VAT-inclusive amount paid by the customer.
func (s *JournalService) WriteInboundEntries(
    ctx context.Context,
    pool *pgxpool.Pool,
    businessID, paymentID string,
    amountCents int64,
) error {
    // VAT extraction from inclusive amount: VAT = amount * 16/116
    vatCents := int64(float64(amountCents) * 16 / 116)
    netCents := amountCents - vatCents

    entries := []JournalEntry{
        {AccountID: "mpesa.till",            EntryType: "debit",  AmountCents: amountCents, Description: "M-PESA inbound"},
        {AccountID: "revenue.sales",         EntryType: "credit", AmountCents: netCents,    Description: "Sales revenue"},
        {AccountID: "liability.vat_payable", EntryType: "credit", AmountCents: vatCents,    Description: "Output VAT 16%"},
    }

    return s.writeBalancedEntries(ctx, pool, businessID, paymentID, entries)
}

// writeBalancedEntries verifies balance then commits in a single transaction.
func (s *JournalService) writeBalancedEntries(
    ctx context.Context,
    pool *pgxpool.Pool,
    businessID, paymentID string,
    entries []JournalEntry,
) error {
    var debits, credits int64
    for _, e := range entries {
        if e.EntryType == "debit" {
            debits += e.AmountCents
        } else {
            credits += e.AmountCents
        }
    }
    if debits != credits {
        return fmt.Errorf("journal does not balance: debits=%d credits=%d", debits, credits)
    }

    conn, err := pool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer conn.Release()

    tx, err := conn.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    repo := s.newTxRepo(tx)
    for _, e := range entries {
        e.BusinessID = businessID
        e.PaymentID = paymentID
        if err := repo.CreateEntry(ctx, e); err != nil {
            return fmt.Errorf("create entry %s: %w", e.AccountID, err)
        }
    }

    return tx.Commit(ctx)
}
```

---

## 12. sqlc Configuration

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/db/queries/"
    schema: "internal/db/migrations/consumer/"
    gen:
      go:
        package: "db"
        out: "internal/db/generated"
        emit_json_tags: true
        emit_interface: true
        emit_empty_slices: true
        emit_pointers_for_null_types: true
```

---

## 13. Makefile

```makefile
.PHONY: run build migrate-eazibiz migrate-mkwanja sqlc test lint

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

migrate-eazibiz:
	goose -dir internal/db/migrations/consumer postgres "$(CONSUMER_EAZIBIZ_DATABASE_URL)" up

migrate-mkwanja:
	goose -dir internal/db/migrations/consumer postgres "$(CONSUMER_MKWANJA_DATABASE_URL)" up

migrate-all: migrate-eazibiz migrate-mkwanja

sqlc:
	sqlc generate

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...
```

---

## 14. Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/server .
EXPOSE 8080
CMD ["./server"]
```

---

## 15. Constraints

- **No ORMs.** sqlc only.
- **No float64 for money.** Integer cents throughout.
- **No global Daraja client.** Build per-business client per request using decrypted credentials.
- **Never log decrypted credentials.** Not even at debug level.
- **Never return credentials in API responses.**
- **Journal is append-only.** Application code never attempts UPDATE or DELETE on journal.
- **Balance check before commit.** Every journal write verifies debits = credits.
- **Redis routing entries have 30-minute TTL.** Reconciliation job handles late-arriving webhooks.
- **Migrations run per consumer database.** Same files, different connection string each time.
- **slog for all logging.** No fmt.Println.
- **Wrap all errors.** `fmt.Errorf("context: %w", err)` throughout.
- **Phone numbers normalised to `2547XXXXXXXX`** before Daraja or DB.
