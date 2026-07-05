## Phase 1 Learnings — Project Foundation
- **Go Multi-Module Build Issues:** When moving `main.go` to a subdirectory (like `cmd/server/main.go`), Go's linker will fail if the root directory still contains `package main` files but lacks a `main()` function. Deleting or using `//go:build ignore` is necessary.
- **Strict Dependency Management:** The TRD forbade GORM, and standardizing on `pgx/v5` and `sqlc` forces a focus on performance and raw SQL control, which is better for financial applications.
- **Environment Variance:** Loading config for multiple consumers dynamically from `CONSUMER_*` env vars allows the service to scale to new apps without code changes.
## Architecture Decisions

- **Per-consumer databases**: Each consumer app gets its own PostgreSQL database. The service connects to all of them via a DB registry.
- **Per-business Daraja clients**: A new Daraja client is built per-request using the business's decrypted credentials. No shared/global client.
- **Integer cents**: All monetary values stored as `BIGINT` cents. No floating-point money.
- **Append-only journal**: The journal table has PostgreSQL rules blocking UPDATE/DELETE. Reversals are new rows.
- **Balance invariant**: `domain.VerifyBalance()` enforces debits == credits before every journal transaction commits.
- **Idempotency keys**: All payment mutations require an `Idempotency-Key` UUID header. Replays return the existing result.
- **Webhook routing**: Per-consumer webhook URLs (e.g., `/webhooks/mpesa/stk/eazibiz`). Redis maps `CheckoutRequestID` â†’ `clientID:paymentID` for routing.
- **Reconciliation**: A background job runs every 5 minutes querying pending payments older than 2 minutes, calling Daraja Transaction Status to resolve them.

## M-PESA (Daraja) Notes

- Sandbox base URL: `https://sandbox.safaricom.co.ke`
- Production base URL: `https://api.safaricom.co.ke`
- OAuth tokens expire after 3600 seconds; cached in Redis with key prefix `daraja_token:`
- STK Push callback format: nested JSON under `Body.stkCallback`
- All phone numbers normalised to `2547XXXXXXXX` format

## Build Notes

- Local development uses Air for live reload (configured in `.air.toml`)
- Fresh was initially tried but failed with Go module subdirectory entry points on Windows
- PostgreSQL accessed via pgx pool â†’ stdlib bridge (since sqlc uses `database/sql` interface)

