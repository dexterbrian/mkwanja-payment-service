# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make run                  # go run ./cmd/server
make build                # go build -o bin/server ./cmd/server
make test                 # go test ./... -v -race -count=1
make lint                 # golangci-lint run ./...
make sqlc                 # sqlc generate (regenerate internal/db/generated/)
make migrate-eazibiz      # run goose migrations against eazibiz DB
make migrate-mkwanja      # run goose migrations against mkwanja DB
make migrate-kanisa       # run goose migrations against kanisa DB
make migrate-all          # migrate all consumer DBs

# Single package test
go test ./internal/domain/... -v -run TestPaymentValidate
```

Requires `.env` (copy from `.env.example`) with `REDIS_URL`, `CREDENTIAL_ENCRYPTION_KEY`, `DARAJA_BASE_URL`, `DARAJA_CALLBACK_URL`, and at least one `CONSUMER_*` group set.

## Architecture

This is a **multi-tenant payment microservice** for M-PESA (Daraja API). It never acts as a PSP — every business registers their own Daraja credentials; the service encrypts and uses them on that business's behalf only.

### Multi-tenancy model

- **Consumer apps** (`eazibiz`, `mkwanja`, `kanisa`) each have their own isolated Postgres database (Supabase).
- Every inbound request carries `X-Consumer-ID` + `X-Service-Secret` headers. Middleware validates these and attaches the correct `*pgxpool.Pool` to the Fiber context via `c.Locals("db_pool", pool)`.
- `internal/db/registry.go` — `Registry` holds one pool per consumer, keyed by consumer ID. All DB writes go through `registry.Get(consumerID)`.
- Same SQL schema is applied to every consumer database via goose migrations in `internal/db/migrations/consumer/`.

### Request lifecycle (STK Push example)

1. Handler reads `db_pool` from context → passes to service
2. Service checks idempotency key → creates payment in `pending`
3. Service loads + decrypts credentials from `business_credentials` table
4. Service builds a per-business `daraja.Client` (never reused across businesses)
5. Daraja call → on success, stores routing key `stk:{CheckoutRequestID}` → `{consumerID}:{businessID}:{paymentID}` in Redis (30-min TTL)
6. Safaricom calls back → `WebhookHandler` enqueues raw body to asynq
7. `queue/worker.go` reads Redis routing key → resolves consumer DB → completes payment → writes journal

### Money rules

- **Integer cents only** — `amount_cents BIGINT`, never `float64`
- **Journal is append-only** — Postgres `CREATE RULE no_update_journal / no_delete_journal` blocks all mutations; reversals are new rows with `reversal_of` FK
- **Balance invariant** — `domain.VerifyBalance(entries)` must return nil before any `journal` transaction commits; enforced in `service/journal_service.go`

### Credential security

- AES-256-GCM via `internal/crypto/credentials.go` (`Encrypt` / `Decrypt`)
- Key is `CREDENTIAL_ENCRYPTION_KEY` env var (32-byte hex) — never in DB
- Decrypted credentials must never be logged or returned in API responses

### Code generation

`internal/db/generated/` is sqlc output — **never edit manually**. To change queries: edit files in `internal/db/queries/`, run `make sqlc`.

### Key constraints from docs/rules.md and docs/trd.md

- No ORMs — sqlc only
- No global Daraja client — build per-business per request
- `slog` for all logging — no `fmt.Println`
- Phone numbers normalised to `2547XXXXXXXX` (`domain.NormalisePhone`) before DB or Daraja
- Every `POST` requires `Idempotency-Key` UUID header; middleware enforces this
- Webhook handlers always return HTTP 200 — reconciliation job recovers missed webhooks

### Build phases

Progress is tracked in `docs/phases.md`. Phases 1–2 are complete. Implement features in this order per the TRD: migration → sqlc query → domain type → repository → service → handler → route → tests.
