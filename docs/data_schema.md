# Data Schema: Mkwanja Payment Service

The payment service shares a PostgreSQL database with the Mkwanja App ecosystem (Supabase) but maintains its own caching layer for idempotency.

## 1. Shared PostgreSQL Entities (Relevant to Payments)

The payment service primarily writes to the `transactions` table.

### 1.1 transactions
- `id` (uuid, PK)
- `user_id` (uuid, FK)
- `wallet_id` (uuid, FK)
- `budget_item_id` (uuid, FK, nullable)
- `amount` (numeric)
- `debit` (numeric)
- `credit` (numeric)
- `type` (enum: 'deposit', 'withdrawal', 'made_payment', ...)
- `status` (enum: 'pending', 'success', 'failed', 'cancelled')
- `payment_tx_id` (varchar) - M-PESA Receipt (filled after callback).
- `external_reference` (varchar) - Internal MerchantRequestID/CheckoutRequestID.
- `execution_meta` (jsonb) - Full payload from Daraja.
- `idempotency_key` (varchar, unique) - The key provided by the client.

## 2. Internal Caching (Redis)

Used for high-speed idempotency and rate limiting.

### 2.1 Idempotency Keys
- **Key:** `idempotency:{request_hash}`
- **Value:** JSON blob containing the original response status and body.
- **TTL:** 24 Hours.

### 2.2 Auth Cache (Optional)
- Caching of Daraja Access Tokens.
- **TTL:** Based on Daraja token expiry (usually 3600s).
