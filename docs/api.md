# mkwanja-payment-service API Reference

Base URL: `http://localhost:8080`

---

## Authentication

### Headers (required on all `/v1/*` routes)

| Header | Required | Description |
|---|---|---|
| `X-Consumer-ID` | Yes | Consumer app identifier (e.g. `eazibiz`, `mkwanja`, `kanisa`) |
| `X-Service-Secret` | Yes | Plaintext secret configured in `CONSUMER_{NAME}_SECRET` env var |

### Idempotency (required on all mutating POST/PUT endpoints)

| Header | Required | Description |
|---|---|---|
| `Idempotency-Key` | Yes | UUID v4/v7 — ensures safe retry without duplicate processing |

If a request with the same key is replayed, the original result is returned with response header `Idempotency-Replayed: true`.

### Error format

All errors return a consistent envelope:

```json
{
    "error": {
        "code":      "ERROR_CODE",
        "message":   "Human-readable description",
        "retryable": false
    }
}
```

| Status | Meaning |
|---|---|
| 400 | Validation failure or business logic error |
| 401 | Missing or invalid `X-Consumer-ID` / `X-Service-Secret` |
| 404 | Resource not found |
| 500 | Internal server error |

---

## Health

### GET /health

Liveness check — always 200 if the process is running.

**Request:**
```
GET /health
```

**Response 200:**
```json
{"status": "ok"}
```

---

### GET /health/ready

Readiness check — pings all consumer databases and Redis.

**Request:**
```
GET /health/ready
```

**Response 200 (all healthy):**
```json
{"status": "ready"}
```

**Response 503 (degraded):**
```json
{
    "status": "degraded",
    "issues": {
        "redis":      "connection refused",
        "db:eazibiz": "connection refused"
    }
}
```

---

## Clients

All client routes require `X-Consumer-ID`, `X-Service-Secret`.

### POST /v1/clients

Register a new business client with encrypted Daraja credentials. Seeds 8 default journal accounts.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

**Request:**
```json
{
    "external_id":         "shop-001",
    "name":                "Acme Shop",
    "shortcode":           "174379",
    "consumer_key":        "abc123...",
    "consumer_secret":     "xyz789...",
    "passkey":             "bfb2...",
    "initiator_name":      "apitest",
    "security_credential": ""
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `external_id` | string | Yes | Your external identifier for this client |
| `name` | string | Yes | Human-readable business name |
| `shortcode` | string | Yes | M-PESA shortcode (paybill/till) |
| `consumer_key` | string | Yes | Daraja API consumer key |
| `consumer_secret` | string | Yes | Daraja API consumer secret |
| `passkey` | string | Yes | Daraja STK Push passkey (Lipa Na M-PESA Online) |
| `initiator_name` | string | No | M-PESA API initiator name (for B2C/B2B) |
| `security_credential` | string | No | M-PESA security credential (for B2C/B2B) |

**Response 201:**
```json
{
    "id":          "a1b2c3d4-...",
    "external_id": "shop-001",
    "name":        "Acme Shop",
    "active":      true,
    "created_at":  "2026-07-05T18:00:00Z"
}
```

---

### POST /v1/clients/test-credentials

Test Daraja OAuth credentials without persisting them.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

**Request:**
```json
{
    "consumer_key":    "abc123...",
    "consumer_secret": "xyz789...",
    "shortcode":       "174379"
}
```

**Response 200:**
```json
{"status": "ok", "message": "Credentials are valid"}
```

---

### PUT /v1/clients/:client_id/credentials

Update existing client's Daraja credentials. Replaces all credential fields.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

**URL params:** `client_id` (string)

**Request:**
```json
{
    "shortcode":           "174379",
    "consumer_key":        "new-key-...",
    "consumer_secret":     "new-secret-...",
    "passkey":             "new-passkey-...",
    "initiator_name":      "apitest",
    "security_credential": ""
}
```

**Response 200:**
```json
{"status": "ok", "message": "Credentials updated"}
```

---

### PUT /v1/admin/operator/credentials

Update credentials for the operator client (configured via `OPERATOR_CLIENT_ID` env var).

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

Same request body as `PUT /v1/clients/:client_id/credentials`.

**Response 200:**
```json
{"status": "ok", "message": "Operator credentials updated"}
```

**Response 404** (no operator configured):
```json
{"error": {"code": "NOT_CONFIGURED", "message": "Operator client is not configured", "retryable": false}}
```

---

### DELETE /v1/clients/:client_id

Soft-deactivate a client. No idempotency required.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**URL params:** `client_id` (string)

**Response 200:**
```json
{"status": "ok", "message": "Client deactivated"}
```

---

### GET /v1/clients/:client_id

Get a single client by ID.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**Response 200:**
```json
{
    "id":          "a1b2c3d4-...",
    "external_id": "shop-001",
    "name":        "Acme Shop",
    "active":      true,
    "created_at":  "2026-07-05T18:00:00Z",
    "updated_at":  "2026-07-05T18:00:00Z"
}
```

---

### GET /v1/clients

List all clients for the current consumer.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**Response 200:**
```json
{
    "clients": [
        {
            "id":          "a1b2c3d4-...",
            "external_id": "shop-001",
            "name":        "Acme Shop",
            "active":      true,
            "created_at":  "2026-07-05T18:00:00Z"
        }
    ]
}
```

---

## Payments

All payment routes require `X-Consumer-ID`, `X-Service-Secret`.

### POST /v1/payments/stk-push

Initiate an M-PESA STK Push (Lipa Na M-PESA Online) to a customer's phone. The customer receives a push notification to enter their PIN.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

**Request:**
```json
{
    "client_id":    "a1b2c3d4-...",
    "amount_cents": 10000,
    "currency":     "KES",
    "phone_number": "254712345678",
    "reference":    "INV-001",
    "description":  "Payment for invoice INV-001"
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `client_id` | string | Yes | — | Registered client UUID |
| `amount_cents` | integer | Yes | — | Amount in cents (e.g., 10000 = KES 100) |
| `currency` | string | No | `"KES"` | ISO 4217 currency code |
| `phone_number` | string | Yes | — | Customer phone (`2547XXXXXXXX` or `07XXXXXXXX`) |
| `reference` | string | Yes | — | Account reference shown to customer |
| `description` | string | No | `""` | Transaction description |

**Response 201:**
```json
{
    "payment_id":          "pay_abc123...",
    "checkout_request_id": "ws_CO_...",
    "status":              "pending"
}
```

On idempotency replay, response header `Idempotency-Replayed: true` is added.

---

### POST /v1/payments/b2c

Initiate a Business-to-Customer (B2C) payment — send money from the business till to a customer's M-PESA wallet.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

**Request:**
```json
{
    "client_id":    "a1b2c3d4-...",
    "amount_cents": 5000,
    "currency":     "KES",
    "phone_number": "254712345678",
    "reference":    "PAYOUT-001",
    "description":  "Commission payout",
    "command_id":   "BusinessPayment",
    "remarks":      "Monthly commission",
    "occasion":     ""
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `client_id` | string | Yes | — | Registered client UUID |
| `amount_cents` | integer | Yes | — | Amount in cents (e.g., 5000 = KES 50) |
| `currency` | string | No | `"KES"` | ISO 4217 currency code |
| `phone_number` | string | Yes | — | Recipient phone (`2547XXXXXXXX` or `07XXXXXXXX`) |
| `reference` | string | Yes | — | Payment reference |
| `description` | string | No | `""` | Transaction description |
| `command_id` | string | No | `""` | B2C command (`BusinessPayment`, `SalaryPayment`, `PromotionPayment`) |
| `remarks` | string | No | `""` | Remarks sent to M-PESA |
| `occasion` | string | No | `""` | Occasion sent to M-PESA |

**Response 201:**
```json
{
    "payment_id":                 "pay_def456...",
    "conversation_id":            "AG_2026...",
    "originator_conversation_id": "",
    "status":                     "pending"
}
```

---

### POST /v1/payments/b2b

Initiate a Business-to-Business (B2B) payment — transfer funds between business shortcodes.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`, `Idempotency-Key`

**Request:**
```json
{
    "client_id":             "a1b2c3d4-...",
    "amount_cents":          200000,
    "currency":              "KES",
    "receiver_shortcode":    "654321",
    "reference":             "SUPPLIER-001",
    "description":           "Supplier payment",
    "command_id":            "BusinessPayBill",
    "sender_identifier_type":  "4",
    "receiver_identifier_type": "4",
    "remarks":               "Monthly supplier payment"
}
```

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `client_id` | string | Yes | — | Registered client UUID |
| `amount_cents` | integer | Yes | — | Amount in cents (e.g., 200000 = KES 2,000) |
| `currency` | string | No | `"KES"` | ISO 4217 currency code |
| `receiver_shortcode` | string | Yes | — | Recipient business shortcode |
| `reference` | string | Yes | — | Payment reference |
| `description` | string | No | `""` | Transaction description |
| `command_id` | string | No | `"BusinessPayBill"` | B2B command (`BusinessPayBill`, `BusinessBuyGoods`) |
| `sender_identifier_type` | string | No | `"4"` | Sender identifier type (4=shortcode) |
| `receiver_identifier_type` | string | No | `"4"` | Receiver identifier type (4=shortcode) |
| `remarks` | string | No | `""` | Remarks sent to M-PESA |

**Response 201:**
```json
{
    "payment_id":                 "pay_ghi789...",
    "conversation_id":            "AG_2026...",
    "originator_conversation_id": "",
    "status":                     "pending"
}
```

---

### GET /v1/payments/:id

Get a single payment by ID.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**URL params:** `id` (payment UUID)

**Response 200:**
```json
{
    "id":                  "pay_abc123...",
    "client_id":           "a1b2c3d4-...",
    "idempotency_key":     "550e8400-...",
    "provider":            "mpesa",
    "payment_type":        "stk_push",
    "direction":           "inbound",
    "status":              "completed",
    "amount_cents":        10000,
    "currency":            "KES",
    "phone_number":        "254712345678",
    "reference":           "INV-001",
    "description":         "Payment for invoice INV-001",
    "provider_request_id": "ws_CO_...",
    "provider_tx_id":      "MPS_...",
    "provider_receipt":    "NFC12345",
    "created_at":          "2026-07-05T18:00:00Z",
    "updated_at":          "2026-07-05T18:01:00Z",
    "completed_at":        "2026-07-05T18:01:05Z"
}
```

**Payment types:** `stk_push`, `b2c`, `b2b`, `c2b`

**Directions:** `inbound` (STK Push, C2B), `outbound` (B2C, B2B)

**Statuses:** `pending` → `processing` → `completed` | `failed` | `cancelled` | `reversed`

---

### GET /v1/payments

List payments for a client with pagination.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**Query params:**

| Param | Required | Default | Max | Description |
|---|---|---|---|---|
| `client_id` | Yes | — | — | Filter by client UUID |
| `limit` | No | 20 | 100 | Max results per page |
| `offset` | No | 0 | — | Pagination offset |

**Response 200:**
```json
{
    "payments": [
        {
            "id":           "pay_abc123...",
            "client_id":    "a1b2c3d4-...",
            "provider":     "mpesa",
            "payment_type": "stk_push",
            "status":       "completed",
            "amount_cents": 10000,
            "currency":     "KES",
            "reference":    "INV-001",
            "created_at":   "2026-07-05T18:00:00Z"
        }
    ],
    "limit":  20,
    "offset": 0
}
```

---

## Ledger

All ledger routes require `X-Consumer-ID`, `X-Service-Secret`.

### GET /v1/ledger

List journal entries for a client. Returns double-entry ledger records ordered by creation date (newest first).

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**Query params:**

| Param | Required | Default | Max | Description |
|---|---|---|---|---|
| `client_id` | Yes | — | — | Filter by client UUID |
| `limit` | No | 20 | 100 | Max results per page |
| `offset` | No | 0 | — | Pagination offset |

**Response 200:**
```json
{
    "entries": [
        {
            "id":            1,
            "client_id":     "a1b2c3d4-...",
            "payment_id":    "pay_abc123...",
            "account_id":    "mpesa.till",
            "entry_type":    "debit",
            "amount_cents":  10000,
            "currency":      "KES",
            "description":   "Inbound payment via stk_push",
            "reversal_of":   null,
            "created_at":    "2026-07-05T18:01:05Z"
        },
        {
            "id":            2,
            "client_id":     "a1b2c3d4-...",
            "payment_id":    "pay_abc123...",
            "account_id":    "revenue.sales",
            "entry_type":    "credit",
            "amount_cents":  10000,
            "currency":      "KES",
            "description":   "Revenue from stk_push",
            "reversal_of":   null,
            "created_at":    "2026-07-05T18:01:05Z"
        }
    ],
    "limit":  20,
    "offset": 0
}
```

**Entry types:** `debit`, `credit`
- **Inbound payments** (STK Push, C2B): DEBIT `mpesa.till`, CREDIT `revenue.sales`
- **Outbound payments** (B2C, B2B): DEBIT `expense.operations`, CREDIT `mpesa.till`

The balance invariant (debits == credits) is enforced before every journal transaction.

---

### GET /v1/ledger/balance

Get account balances for a client — aggregated debits, credits, and net position per account.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**Query params:**

| Param | Required |
|---|---|
| `client_id` | Yes |

**Response 200:**
```json
{
    "balances": [
        {
            "account_id":          "mpesa.till",
            "total_debits_cents":  100000,
            "total_credits_cents": 50000,
            "net_cents":           50000
        },
        {
            "account_id":          "revenue.sales",
            "total_debits_cents":  0,
            "total_credits_cents": 100000,
            "net_cents":           -100000
        }
    ]
}
```

---

### GET /v1/ledger/trial-balance

Get the trial balance for a client — same data as account balances but presented as a trial balance for accounting verification.

**Headers:** `X-Consumer-ID`, `X-Service-Secret`

**Query params:**

| Param | Required |
|---|---|
| `client_id` | Yes |

**Response 200:**
```json
{
    "trial_balance": [
        {
            "account_id":          "mpesa.till",
            "total_debits_cents":  100000,
            "total_credits_cents": 50000,
            "net_cents":           50000
        },
        {
            "account_id":          "revenue.sales",
            "total_debits_cents":  0,
            "total_credits_cents": 100000,
            "net_cents":           -100000
        }
    ]
}
```

---

### Default Chart of Accounts

Accounts are automatically seeded when a client is registered.

| Account ID | Name | Type | Normal Balance |
|---|---|---|---|
| `mpesa.till` | M-PESA till | asset | debit |
| `revenue.sales` | Sales revenue | revenue | credit |
| `revenue.other` | Other revenue | revenue | credit |
| `expense.cogs` | Cost of goods sold | expense | debit |
| `expense.operations` | Operating expenses | expense | debit |
| `liability.vat_payable` | VAT payable | liability | credit |
| `liability.pending` | Pending payments | liability | credit |
| `fees.mpesa` | M-PESA charges | expense | debit |

---

## Webhooks

Webhook routes are called by Safaricom — no authentication required. All handlers return HTTP 200 immediately and process payment callbacks asynchronously via the background job queue.

### POST /webhooks/mpesa/stk/:consumer_id

STK Push callback from M-PESA after a customer completes (or cancels) the STK Push prompt.

```json
{
    "Body": {
        "stkCallback": {
            "MerchantRequestID": "29115-...",
            "CheckoutRequestID": "ws_CO_...",
            "ResultCode": 0,
            "ResultDesc": "The service request has been completed. Payment has been received successfully.",
            "CallbackMetadata": {
                "Item": [
                    {"Name": "Amount", "Value": 100.00},
                    {"Name": "MpesaReceiptNumber", "Value": "NFC12345"},
                    {"Name": "TransactionDate", "Value": 20260705180000},
                    {"Name": "PhoneNumber", "Value": 254712345678}
                ]
            }
        }
    }
}
```

| ResultCode | Meaning |
|---|---|
| `0` | Success — payment was received |
| `1032` | Request cancelled by user |
| `1037` | Timeout — STK Push not responded to |
| `2001` | Insufficient balance |

**Response 200:**
```json
{"status": "ok"}
```

---

### POST /webhooks/mpesa/b2c/:consumer_id

B2C payment result callback.

**Response 200:**
```json
{"status": "ok"}
```

---

### POST /webhooks/mpesa/b2b/:consumer_id

B2B payment result callback.

**Response 200:**
```json
{"status": "ok"}
```

---

### POST /webhooks/mpesa/c2b/:consumer_id/confirm

C2B confirmation callback — sent when a customer sends money to the business paybill/till.

**Response 200:**
```json
{"ResultCode": 0, "ResultDesc": "Success"}
```

---

### POST /webhooks/mpesa/c2b/:consumer_id/validate

C2B validation callback — sent before processing a C2B transaction to allow the system to accept or reject it.

**Response 200:**
```json
{"ResultCode": 0, "ResultDesc": "Success"}
```

---

## Reconciliation

The reconciliation service runs automatically every 5 minutes as a background goroutine. It:

1. Queries all `pending` payments older than 2 minutes
2. For each payment, calls the Daraja Transaction Status API using the client's own credentials
3. If confirmed successful: completes the payment and writes balanced journal entries
4. If confirmed failed: marks the payment as failed
5. Logs all outcomes via structured slog

No manual API endpoint is needed — the reconciliation runs in-process on each service instance.

---

## Headers Reference

| Header | Routes | Description |
|---|---|---|
| `X-Consumer-ID` | All `/v1/*` | Consumer app identifier |
| `X-Service-Secret` | All `/v1/*` | Consumer app secret |
| `Idempotency-Key` | POST `/v1/*`, PUT `/v1/*` | UUID for idempotent retry |
| `Idempotency-Replayed`| Response header | Set to `true` when a duplicate key is detected |

---

## Status Codes

| Code | Meaning |
|---|---|
| 200 | Success |
| 201 | Created |
| 400 | Bad request (validation error) |
| 401 | Unauthorized (missing or invalid credentials) |
| 404 | Resource not found |
| 500 | Internal server error |
| 503 | Service unavailable (dependency down) |

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | No | `8080` | HTTP listen port |
| `ENVIRONMENT` | No | `development` | Runtime environment |
| `REDIS_URL` | Yes | — | Redis connection string |
| `CREDENTIAL_ENCRYPTION_KEY` | Yes | — | 32-byte hex key for AES-256-GCM |
| `DARAJA_BASE_URL` | Yes | — | Daraja API base URL (sandbox/production) |
| `DARAJA_CALLBACK_URL` | Yes | — | Publicly reachable callback URL for webhooks |
| `OPERATOR_CLIENT_ID` | No | `""` | Operator client UUID |
| `OPERATOR_CONSUMER_ID` | No | `mkwanja` | Consumer app for operator |
| `OPERATOR_CLIENT_NAME` | No | `Dexter Operator` | Operator display name |
| `CONSUMER_{NAME}_ID` | Yes* | — | Consumer app ID |
| `CONSUMER_{NAME}_SECRET` | Yes* | — | Consumer app plaintext secret |
| `CONSUMER_{NAME}_CALLBACK_URL` | Yes* | — | Per-consumer webhook base URL |
| `CONSUMER_{NAME}_DATABASE_URL` | Yes* | — | Per-consumer Postgres URL |

\* At least one consumer group (set of 4 `CONSUMER_{NAME}_*` vars) is required. Each group defines one tenant app with its own database.
