# mkwanja-payment-service

Multi-tenant M-PESA payment service (Daraja API) for consumer apps.

## Architecture

See [CLAUDE.md](./CLAUDE.md) for full architecture docs.

## Quick Start

1. Copy `.env.example` → `.env` and fill in your config
2. Ensure local infra is running (see below)
3. `make run`

## Dependencies

- Go 1.23+
- PostgreSQL (via Supabase)
- Redis
- Safaricom Daraja API credentials

## Local Infra

Start local Supabase and Redis:

```bash
# Start Supabase (Docker)
npx supabase start

# Start Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

## API Endpoints

See [docs/api.md](./docs/api.md) for full API reference.

### Client Management
- `POST /v1/clients` — Register a new client with Daraja credentials
- `PUT /v1/clients/:client_id/credentials` — Update credentials
- `DELETE /v1/clients/:client_id` — Deactivate a client
- `POST /v1/clients/test-credentials` — Test Daraja OAuth

### Payments
- `POST /v1/payments/stk-push` — Initiate STK Push
- `POST /v1/payments/b2c` — Initiate B2C payment
- `POST /v1/payments/b2b` — Initiate B2B payment
- `GET /v1/payments/:id` — Get payment by ID
- `GET /v1/payments` — List payments

### Ledger
- `GET /v1/ledger` — List journal entries
- `GET /v1/ledger/balance` — Account balances
- `GET /v1/ledger/trial-balance` — Trial balance

### Webhooks (Safaricom → Service)
- `POST /webhooks/mpesa/stk/:consumer_id`
- `POST /webhooks/mpesa/b2c/:consumer_id`
- `POST /webhooks/mpesa/b2b/:consumer_id`
- `POST /webhooks/mpesa/c2b/:consumer_id/confirm`
- `POST /webhooks/mpesa/c2b/:consumer_id/validate`

### Health
- `GET /health` — Liveness check
- `GET /health/ready` — Readiness check (pings all DBs + Redis)

## Project Structure

```
cmd/server/            — Entry point
internal/
  config/              — Environment config
  crypto/              — AES-256-GCM encryption
  daraja/              — M-PESA Daraja API client
  db/                  — DB registry, migrations, sqlc generated code
  domain/              — Domain types and validation
  handler/             — HTTP handlers (Fiber)
  middleware/          — Auth, consumer, idempotency
  queue/               — Async job queue (asynq)
  repository/          — Data access layer
  router/              — Route registration
  service/             — Business logic
docs/                  — Documentation
```

## Testing

```bash
make test          # go test ./... -v -race -count=1
make lint          # golangci-lint run ./...
```
