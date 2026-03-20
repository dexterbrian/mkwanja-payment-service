# PRD: `mkwanja-payment-svc`

**Version:** 0.3.0
**Status:** Ready for build
**Repo:** `msingi-dev/mkwanja-payment-svc`
**Consumed by:** `eazibiz`, `kanisa-digital`, `mkwanja` (Flutter), `ubuntu-coop`

---

## 1. Purpose

`mkwanja-payment-svc` is the central payment processing and double-entry ledger service for all `@msingi` applications. It is the single point of contact for any money movement — no consumer application ever talks to Safaricom's Daraja API directly.

The service operates on a **bring-your-own-till model**: each business registers their own Safaricom M-PESA till number, obtains their own Daraja API credentials, and provides those to the payment service. The service encrypts and stores them, then uses them to make API calls on that business's behalf. This model means msingi does not act as a Payment Service Provider and is not required to hold a PSP licence from the Central Bank of Kenya (CBK).

---

## 2. Regulatory Context

Under the CBK's National Payment System Act and PSP Regulations, a licence is required to operate as a payment intermediary. By requiring each business to hold their own Safaricom till and Daraja credentials, msingi acts as a software integration layer rather than a payment intermediary. Each business's M-PESA transactions flow between that business's registered till and Safaricom — msingi facilitates the API calls but never holds, settles, or intermediates funds.

This model must be maintained consistently:
- msingi must never operate a shared till used across multiple businesses
- Each business's Daraja credentials are used exclusively for that business's transactions
- The service must not commingle funds between businesses in any way

---

## 3. Goals

- **Correctness over speed.** Every transaction must be recorded correctly or not at all. Partial writes are never acceptable.
- **Idempotency.** Every mutating endpoint accepts an idempotency key. Duplicate requests return the original response without re-processing.
- **Auditability.** Every state change is logged. The journal is append-only. Nothing is deleted.
- **Isolation.** Consumer apps never deal with Daraja directly. Each business's credentials and ledger are fully siloed.
- **Reliability.** Webhook delivery from Safaricom is unreliable — the service handles retries, deduplication, and out-of-order delivery.
- **Data isolation.** Each consumer app has its own isolated Postgres database. Eazibiz ledger data never shares a database with Mkwanja data.

---

## 4. Non-Goals

- No end-user authentication — consumer apps authenticate their users; this service authenticates calling *services*
- No SMS, push notifications, or emails — the service emits events that consumers act on
- No Stripe subscriptions — Stripe is handled directly by consumer apps where applicable
- No business logic for any individual app — the service is payment-agnostic
- No UI
- No msingi-owned shared till — all Daraja calls use individual business credentials

---

## 5. Bring-Your-Own-Till Model

### 5.1 Business onboarding

When a business is onboarded onto a consumer app, they provide:
- Their M-PESA till number or paybill number
- Their Daraja API consumer key and consumer secret
- Their Daraja passkey (for STK Push)
- Their Safaricom shortcode
- Optionally: their Daraja initiator name and security credential (required for B2C and B2B outbound payments)

The consumer app collects these during its onboarding wizard and calls the payment service credentials registration endpoint. The payment service encrypts credentials at rest and associates them with that business's `business_id`.

### 5.2 Credential security

- Consumer key, consumer secret, passkey, and security credential are encrypted at rest using AES-256-GCM
- The encryption key is an environment variable on the payment service — never stored in the database
- Credentials are never logged and never returned in API responses after registration
- Each business's Daraja OAuth token is cached in Redis under a key scoped to their `business_id` — never shared across businesses

### 5.3 Per-request credential loading

Every payment request:
1. Resolves the consumer database from the `X-Consumer-ID` header
2. Loads and decrypts the business's credentials from that database
3. Fetches (or uses cached) OAuth token for that specific business
4. Calls Daraja using that business's shortcode and credentials only

### 5.4 Webhook routing

All consumer apps share a single Safaricom-registered webhook URL. The service routes incoming webhooks to the correct business by:
1. At initiation time: storing `stk:{CheckoutRequestID}` → `{consumerID}:{businessID}:{paymentID}` in Redis (30-minute TTL)
2. At webhook receipt: reading the Redis key to identify which consumer database and business to update
3. The reconciliation job handles payments whose webhooks arrive after the Redis TTL expires

---

## 6. Database Architecture

Each consumer app has its own isolated Postgres database hosted as a separate Supabase project. The payment service maintains a connection registry and routes all writes to the appropriate database.

### 6.1 Consumer databases

| Consumer | Database | Notes |
|---|---|---|
| `eazibiz` | `payment_eazibiz` | One Supabase project shared across all Eazibiz deployments |
| `mkwanja` | `payment_mkwanja` | Mkwanja Flutter app users |
| `kanisa` | `payment_kanisa` | Kanisa Digital churches |

### 6.2 Schema per consumer database

Every consumer database contains the same schema: `businesses`, `business_credentials`, `payments`, `payment_events`, `journal_accounts`, `journal`.

---

## 7. Supported M-PESA APIs

| API | Direction | Use case |
|---|---|---|
| STK Push | Customer → Business | Customer pays at checkout, donations, auto-payments |
| B2C | Business → Person | Refunds, supplier payments, withdrawal payouts |
| B2B | Business → Business | Inter-business payments, bulk supplier payments |
| C2B | Customer → Business | Customer-initiated paybill/till payments |
| Transaction Status Query | — | Reconciliation of unconfirmed transactions |
| Account Balance | — | Check a business's M-PESA till balance |

### 7.1 B2B specifics

B2B uses `CommandID`:
- `BusinessPayBill` — paying a supplier on paybill
- `BusinessBuyGoods` — paying a till number directly

B2B requires an API initiator configured on the business's Daraja portal. Businesses must provide initiator name and security credential during M-PESA setup (optional step — required only if they want outbound B2B payments).

---

## 8. Double-Entry Ledger

Every money movement creates exactly two journal entries — one debit and one credit — that sum to zero. Enforced at application and database level.

### 8.1 Standard journal accounts (seeded per business on registration)

| Account | Type | Normal balance |
|---|---|---|
| `mpesa.till` | Asset | Debit |
| `revenue.sales` | Revenue | Credit |
| `revenue.other` | Revenue | Credit |
| `expense.cogs` | Expense | Debit |
| `expense.operations` | Expense | Debit |
| `liability.vat_payable` | Liability | Credit |
| `liability.pending` | Liability | Credit |
| `fees.mpesa` | Expense | Debit |

### 8.2 Example — confirmed inbound STK Push (KES 1,160 VAT-inclusive)

```
DEBIT  mpesa.till             KES 1,160   ← till balance increases
CREDIT revenue.sales          KES 1,000   ← net revenue ex-VAT
CREDIT liability.vat_payable  KES   160   ← output VAT collected (16%)
```

### 8.3 Ledger rules

- Both entries written in a single database transaction — both or neither
- No `UPDATE` or `DELETE` ever on the journal table (enforced by Postgres rules)
- Reversals are new entries with `reversal_of` referencing the original entry ID
- Application layer verifies debits = credits before committing every transaction

---

## 9. API Surface

All consumer-facing endpoints require `X-Service-Secret` and `X-Consumer-ID` headers.

### Credential management
| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/businesses` | Register business + store encrypted credentials |
| `POST` | `/v1/businesses/test-credentials` | Verify credentials work without saving |
| `PUT` | `/v1/businesses/:id/credentials` | Update credentials |
| `DELETE` | `/v1/businesses/:id` | Deactivate business |

### Payments
| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/payments/stk-push` | Initiate STK Push |
| `POST` | `/v1/payments/b2c` | Initiate B2C |
| `POST` | `/v1/payments/b2b` | Initiate B2B |
| `GET` | `/v1/payments/:id` | Get payment status |
| `GET` | `/v1/payments` | List payments |

### Ledger
| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/ledger` | Query journal entries for a business |
| `GET` | `/v1/ledger/balance` | Account balances for a business |
| `GET` | `/v1/ledger/trial-balance` | Full trial balance |

### Webhooks (no consumer auth — Safaricom calls these)
| Method | Path | Description |
|---|---|---|
| `POST` | `/webhooks/mpesa/stk` | STK Push result |
| `POST` | `/webhooks/mpesa/b2c` | B2C result |
| `POST` | `/webhooks/mpesa/b2b` | B2B result |
| `POST` | `/webhooks/mpesa/c2b/confirm` | C2B confirmation |
| `POST` | `/webhooks/mpesa/c2b/validate` | C2B validation |

### Health
| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness |
| `GET` | `/health/ready` | Readiness — all consumer DBs + Redis |

---

## 10. Idempotency

Every `POST` requires an `Idempotency-Key` header (UUID). Same key within 24 hours returns the original response with `Idempotency-Replayed: true`. No re-processing.

---

## 11. Error Model

```json
{
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "The M-PESA till has insufficient funds.",
    "payment_id": "pay_01J...",
    "retryable": false
  }
}
```

`retryable: true` for network timeouts and Daraja 5xx. Consumer apps implement exponential backoff with jitter for retryable errors.

---

## 12. Reconciliation

Background job every 5 minutes: queries any payment in `pending` state older than 2 minutes, calls Daraja Transaction Status API using that business's credentials, updates accordingly. Recovers payments where Safaricom's webhook was never delivered.

---

## 13. Success Criteria

- STK Push initiated within 3 seconds of API call
- Webhook processed and journal written within 5 seconds of Safaricom delivery
- Zero double-charges across any volume of duplicate webhook deliveries
- Reconciliation catches 100% of missed webhooks within 10 minutes
- Debits always equal credits across all journal entries
- One business's credentials can never affect another business's transactions

---

## 14. Future

- **Stripe:** `provider` field on payments (`mpesa` | `stripe`) accommodates this without schema change
- **Additional consumer apps:** adding a new consumer requires adding env vars and running migrations against a new database — no code changes to the service
