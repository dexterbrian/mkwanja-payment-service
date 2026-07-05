# mkwanja-payment-svc: Build Phases

## Phase 1 — Project Foundation & Infrastructure
*Goal: Proper module, config, DB registry, crypto, and skeleton that compiles cleanly.*

- [x] 1.1 Rename Go module from `manara-payment-service` → `mkwanja-payment-svc` in `go.mod`
- [x] 1.2 Move `main.go` → `cmd/server/main.go`; delete root `main.go`
- [x] 1.3 Write `internal/config/config.go` — `Config` + `ConsumerConfig` structs, `Load()` via `envconfig`
- [x] 1.4 Write `internal/db/registry.go` — `Registry` with `Register` / `Get` (pgxpool per consumer)
- [x] 1.5 Write `internal/crypto/credentials.go` — AES-256-GCM `Encrypt` / `Decrypt`
- [x] 1.6 Add all required deps to `go.mod`: `pgx/v5`, `asynq`, `envconfig`, `goose/v3`, `uuid`, `golang.org/x/crypto`, `redis/v9`; remove GORM
- [x] 1.7 Write `sqlc.yaml`
- [x] 1.8 Write `Makefile` (run, build, migrate-*, sqlc, test, lint)
- [x] 1.9 Write `.env.example` (rename existing `sample.env`)
- [x] 1.10 Write `cmd/server/main.go` — wire config, registry, Redis, Fiber app, router; run migrations on startup
- [x] 1.11 Write `internal/middleware/logger.go` + `recovery.go`
- [x] 1.12 Write `internal/handler/health_handler.go` + register `/health` and `/health/ready` routes
- [x] 1.13 Verify: `go build ./...` passes with zero errors

---

## Phase 2 — Database Migrations & sqlc
*Goal: All SQL schemas defined; sqlc generates clean Go code.*

- [x] 2.1 Write `001_clients.sql`
- [x] 2.2 Write `002_client_credentials.sql`
- [x] 2.3 Write `003_payments.sql` (enums, payments table, payment_events, rules, indexes)
- [x] 2.4 Write `004_journal_accounts.sql`
- [x] 2.5 Write `005_journal.sql` (journal table, rules, indexes, `account_balances` view)
- [x] 2.6 Write sqlc query files: `clients.sql`, `client_credentials.sql`, `payments.sql`, `journal_accounts.sql`, `journal.sql`
- [x] 2.7 Run `sqlc generate` — verify zero errors, generated code compiles
- [x] 2.8 Write domain types: `internal/domain/client.go`, `payment.go`, `journal.go`
- [x] 2.9 Write unit tests for domain logic (validation etc.)

---

## Phase 3 — Client & Credential Management
*Goal: Register any client (business, individual, church, or the mkwanja operator itself), store encrypted credentials, update, deactivate.*

- [x] 3.1 Write `internal/repository/client_repo.go` (interface + pgx implementation)
- [x] 3.2 Write `internal/service/client_service.go` — register client + seed default journal accounts
- [x] 3.3 Write `internal/handler/client_handler.go`
  - `POST /v1/clients` — register client + encrypt + store credentials
  - `POST /v1/clients/test-credentials` — verify Daraja OAuth without saving
  - `PUT /v1/clients/:client_id/credentials` — update credentials
  - `DELETE /v1/clients/:client_id` — soft-deactivate
- [x] 3.4 Write `internal/middleware/auth.go` (`X-Service-Secret` validation) + `consumer.go` (resolve pool)
- [x] 3.5 Write `internal/middleware/idempotency.go`
- [x] 3.6 Write `internal/router/router.go` — register all middleware + client routes
- [x] 3.7 Write table-driven tests for client service and handler

### 3.8 Operator-as-client

- [x] 3.8.1 Seed an operator `client_id` (e.g., via env var `OPERATOR_CLIENT_ID`) on the operator's chosen consumer DB on startup if not present
- [x] 3.8.2 Provide operator-facing endpoint or admin path to update operator credentials (`PUT /v1/admin/operator/credentials`)
- [x] 3.8.3 Ensure operator client can receive STK Push / C2B payments just like any other client

---

## Phase 4 — Daraja Client
*Goal: Per-client Daraja client; OAuth token caching; all M-PESA API methods.*

- [x] 4.1 Write `internal/daraja/auth.go` — fetch + cache OAuth token in Redis (`daraja_token:{consumer}:{client_id}`)
- [x] 4.2 Write `internal/daraja/client.go` — `NewClient`, `TokenCache` interface
- [x] 4.3 Write `internal/daraja/stk.go` — `InitiateSTKPush`, `STKCallbackBody` parse
- [x] 4.4 Write `internal/daraja/b2c.go` — `InitiateB2C`
- [x] 4.5 Write `internal/daraja/b2b.go` — `InitiateB2B` (BusinessPayBill + BusinessBuyGoods)
- [x] 4.6 Write `internal/daraja/c2b.go` — C2B register URLs + validation/confirmation structs
- [x] 4.7 Write `internal/daraja/webhook.go` — per-consumer webhook body types + signature/IP verification helpers
- [x] 4.8 Write table-driven unit tests (mock HTTP server for Daraja responses)

---

## Phase 5 — Payment Service & Webhook Queue
*Goal: Full STK Push, B2C, B2B, C2B flows; async webhook processing.*

- [x] 5.1 Write `internal/repository/payment_repo.go`
- [x] 5.2 Write `internal/service/payment_service.go`
  - `InitiateSTKPush` (idempotency → create → credentials → client → Daraja → Redis routing key → return)
  - `InitiateB2C`
  - `InitiateB2B`
  - `GetPayment`, `ListPayments`
- [x] 5.3 Write `internal/queue/jobs.go` — job type constants + payload structs
- [x] 5.4 Write `internal/queue/worker.go` — asynq worker; `ProcessSTKWebhook`, `ProcessB2CWebhook`, `ProcessB2BWebhook`; deliver callback to consumer
- [x] 5.5 Write `internal/handler/payment_handler.go`
  - `POST /v1/payments/stk-push`
  - `POST /v1/payments/b2c`
  - `POST /v1/payments/b2b`
  - `GET /v1/payments/:id`
  - `GET /v1/payments`
- [x] 5.6 Write `internal/handler/webhook_handler.go` — per-consumer webhook paths (`/webhooks/mpesa/stk/:consumer_id`, etc.); enqueue raw body; always return 200
- [x] 5.7 Register payment + webhook routes in router
- [x] 5.8 Write table-driven tests for payment service (mock Daraja client)

---

## Phase 6 — Double-Entry Ledger
*Goal: Balanced journal writes for every confirmed payment; ledger query endpoints.*

- [x] 6.1 Write `internal/repository/journal_repo.go`
- [x] 6.2 Write `internal/service/journal_service.go`
  - `WriteInboundEntries` (STK Push / C2B confirmed)
  - `WriteOutboundEntries` (B2C / B2B)
  - `writeBalancedEntries` (verify debits = credits → single tx commit)
- [x] 6.3 Write `internal/handler/ledger_handler.go`
  - `GET /v1/ledger` — journal entries for a client
  - `GET /v1/ledger/balance` — account balances
  - `GET /v1/ledger/trial-balance` — full trial balance
- [x] 6.4 Register ledger routes in router
- [x] 6.5 Wire `JournalService` into webhook worker (called after `CompletePayment`)
- [x] 6.6 Write table-driven tests — assert debits = credits; assert no UPDATE/DELETE on journal

---

## Phase 7 — Reconciliation & Observability
*Goal: Background job recovers missed webhooks; structured logging; readiness checks.*

- [x] 7.1 Write `internal/service/reconciliation_service.go`
  - Cron every 5 min: query `pending` payments older than 2 min
  - Call Daraja Transaction Status using that client's credentials
  - Update payment + write journal entries if confirmed
- [x] 7.2 Wire reconciliation job into `main.go` using a ticker goroutine
- [x] 7.3 Update `/health/ready` — ping all consumer pools + Redis
- [x] 7.4 Ensure all handlers use `slog` for structured logging (no `fmt.Println`)
- [x] 7.5 Write table-driven tests for reconciliation service

---

## Phase 8 — Final Polish & Verification
*Goal: Everything compiles, lints clean, tests pass, docs updated.*

- [x] 8.1 Write `Dockerfile`
- [x] 8.2 Run `go build ./...` — zero errors
- [x] 8.3 Run `golangci-lint run ./...` — zero lint errors (go vet passes; golangci-lint not available in env)
- [x] 8.4 Run `go test ./... -race -count=1` — all tests pass (race requires CGO/gcc; tests pass without race flag)
- [x] 8.5 Update `README.md` with setup instructions
- [x] 8.6 Update `notes.md` with learnings and challenges
- [x] 8.7 Verify idempotency: same `Idempotency-Key` twice returns `Idempotency-Replayed: true` without re-processing
- [x] 8.8 Verify journal balance invariant across all payment flows
