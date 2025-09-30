# Mkwanja Payment Service - Product Requirements Document

## Overview

The Mkwanja Payment Service is a GO microservice that integrates payment methods for Manara B2C and B2B apps, enabling users to pay for rides with E-Moti vehicles.

**Development Timeline:**

- **MVP (Phase 1):** 3-4 sprints (3-4 weeks)
- **Post-MVP (Phase 2+):** Iterative enhancements

---

## MVP Scope (Phase 1)

### Objectives

- ✅ **M-PESA Integration** - STK Push payment flow only
- ✅ **Basic Ledger** - Simple double-entry bookkeeping
- ✅ **Security** - JWT authentication, HTTPS, secrets management
- ✅ **Logging** - Basic request/response logging for debugging
- ✅ **Idempotency** - Prevent duplicate payments
- ✅ **Core API** - Payment initiation and callback handling

### MVP Features

### 1. M-PESA STK Push

- Initiate payment via Safaricom M-PESA API
- Handle STK Push callbacks
- Query transaction status
- Support Paybill setup (configurable)

### 2. Simple Ledger System

- Double-entry bookkeeping
- Two core tables: `journal_entries` and `journal_accounts`
- Basic chart of accounts:
    - **Assets:** Customer Wallet, M-PESA Receivable
    - **Liabilities:** Customer Deposits
    - **Revenue:** Ride Revenue
    - **Expenses:** Rider Fees
- Record debits and credits for each transaction

### 3. Authentication & Security

- JWT-based authentication for B2B/B2C apps
- API key storage for M-PESA (environment variables)
- HTTPS enforcement
- Basic rate limiting (in-memory, simple counter)
- Input validation on all endpoints

### 4. Core Data Models

- `transactions` - Payment transaction records
- `journal_entries` - Ledger entries
- `journal_accounts` - Chart of accounts
- `logs` - Activity logs
- `idempotency_keys` - Duplicate payment prevention
- `mpesa_callbacks` - Callback tracking

### 5. Essential Endpoints

```
POST /api/v1/payments/mpesa/stkpush
POST /api/v1/payments/mpesa/callback
GET  /api/v1/payments/:transaction_id/status
GET  /api/v1/accounts/:account_id/balance
GET  /api/v1/health

```

### 6. Basic Observability

- Structured logging (JSON format)
- Log levels: INFO, WARN, ERROR
- Request/response logging with correlation IDs
- Health check endpoint

---

## Post-MVP Scope (Phase 2+)

### Phase 2: Enhanced Reliability

- ⏳ **Kafka/Event Streaming** - Async payment processing
- ⏳ **Advanced Rate Limiting** - Redis-backed distributed rate limiting
- ⏳ **Metrics & Monitoring** - Prometheus/Grafana integration
- ⏳ **Alerting** - PagerDuty/Slack alerts for failures
- ⏳ **Distributed Tracing** - OpenTelemetry integration

### Phase 3: Additional Payment Providers

- ⏳ **Mookh Integration** - Card and Bonga points
- ⏳ **Flutterwave/Paystack** - Multi-country support

### Phase 4: Advanced Features

- ⏳ **Refunds & Reversals** - Handle payment cancellations
- ⏳ **Reconciliation Engine** - Daily settlement verification
- ⏳ **Advanced Ledger** - Multi-currency, financial reporting
- ⏳ **Webhook Management** - Retry logic, dead letter queues

### Phase 5: Quality & Scale

- ⏳ **Comprehensive Testing** - Unit, integration, E2E tests
- ⏳ **Load Testing** - Performance benchmarking
- ⏳ **CI/CD Pipeline** - Automated deployment
- ⏳ **Disaster Recovery** - Backup and restore procedures

---

## Technical Requirements (MVP)

### Backend Stack

- **Language:** Go 1.21+
- **Framework:** [Fiber v2](https://gofiber.io/)
- **ORM:** [GORM](https://gorm.io/)
- **Database:** PostgreSQL 15+
- **Logger:** [Zap](https://github.com/uber-go/zap) or [Zerolog](https://github.com/rs/zerolog)
- **Validation:** [go-playground/validator](https://github.com/go-playground/validator)

### Infrastructure

- **Hosting:** DigitalOcean, AWS, or Heroku
- **Database:** Managed PostgreSQL instance
- **Secrets:** Environment variables (`.env` for dev, cloud secrets manager for prod)
- **SSL/TLS:** Let's Encrypt or cloud provider certificates

### Security (MVP)

- JWT tokens with 1-hour expiration
- Refresh tokens with 7-day expiration
- M-PESA callback signature verification (non-MVP)
- Environment-based API key management
- Basic rate limiting: 100 requests/minute per user

---

## System Architecture (MVP)

```
┌─────────────────────┐
│  Manara B2B/B2C     │
│  Frontend Apps      │
└──────────┬──────────┘
           │ HTTPS + JWT
           ▼
┌─────────────────────────────┐
│  Mkwanja Payment Service    │
│  (Go Fiber)                 │
│                             │
│  ┌──────────────────────┐  │
│  │  Auth Middleware     │  │
│  └──────────────────────┘  │
│  ┌──────────────────────┐  │
│  │  Payment Handlers    │  │
│  └──────────────────────┘  │
│  ┌──────────────────────┐  │
│  │  M-PESA Service      │  │
│  └──────────────────────┘  │
│  ┌──────────────────────┐  │
│  │  Ledger Service      │  │
│  └──────────────────────┘  │
│  ┌──────────────────────┐  │
│  │  Logger              │  │
│  └──────────────────────┘  │
└──────────┬──────────────────┘
           │
           ▼
┌─────────────────────┐         ┌─────────────────┐
│  PostgreSQL DB      │         │  M-PESA API     │
│                     │         │  (Daraja)       │
│  - transactions     │         └─────────────────┘
│  - journal_entries  │
│  - journal_accounts │
│  - logs             │
│  - idempotency_keys │
└─────────────────────┘

```

---

## Payment Flow (MVP)

### STK Push Initiation Flow

1. **B2B/B2C App** sends payment request to `/api/v1/payments/mpesa/stkpush`
    - Headers: `Authorization: Bearer <JWT>`
    - Body: `{ phone_number, amount, account_reference, transaction_desc }`
2. **Mkwanja Service** validates request:
    - Verify JWT token
    - Check idempotency key (prevent duplicates)
    - Validate phone number format and amount
3. **Create Transaction Record:**
    - Status: `PENDING`
    - Store in `transactions` table
    - Log request with correlation ID
4. **Call M-PESA API:**
    - Authenticate with M-PESA (OAuth token)
    - Send STK Push request
    - Log API call
5. **M-PESA Response:**
    - If successful: Return `CheckoutRequestID` to frontend
    - If failed: Log error, update transaction status to `FAILED`
6. **Return Response to App:**
    
    ```json
    {
      "transaction_id": "txn_abc123",
      "checkout_request_id": "ws_CO_12345",
      "status": "PENDING",
      "message": "STK Push sent to customer"
    }
    
    ```
    

### M-PESA Callback Flow

1. **M-PESA** sends callback to `/api/v1/payments/mpesa/callback`
    - Verify callback signature (security)
    - Parse callback payload
2. **Process Callback:**
    - Extract transaction result (success/failure)
    - Update `transactions` table
    - Store callback in `mpesa_callbacks` table
    - Log callback receipt
3. **Update Ledger (if successful):**
    - Create journal entries:
        
        ```
        DEBIT:  M-PESA Receivable    +500 KESCREDIT: Ride Revenue          +500 KES
        
        ```
        
    - Log ledger operation
4. **Notify Frontend (optional for MVP):**
    - B2B/B2C apps poll `/api/v1/payments/:transaction_id/status`
    - Or implement webhook to notify apps (Post-MVP)

---

## Database Schema (MVP)

### 1. transactions

```sql
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) NOT NULL, -- From B2B/B2C app
    payment_provider VARCHAR(50) NOT NULL, -- 'mpesa'
    payment_method VARCHAR(50) NOT NULL, -- 'stk_push'
    phone_number VARCHAR(15) NOT NULL,
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KES',
    status VARCHAR(50) NOT NULL, -- PENDING, SUCCESS, FAILED, EXPIRED
    payment_reference VARCHAR(100), -- M-PESA transaction ID
    checkout_request_id VARCHAR(100), -- M-PESA CheckoutRequestID
    account_reference VARCHAR(100), -- Ride ID or booking reference
    transaction_desc TEXT,
    idempotency_key VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_payment_reference ON transactions(payment_reference);

```

### 2. journal_accounts

```sql
CREATE TABLE journal_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100), -- Links the user id
    account_name VARCHAR(100) NOT NULL, -- e.g., 'Customer Wallet'
    account_type VARCHAR(50) NOT NULL, -- ASSET, LIABILITY, REVENUE, EXPENSE
    balance DECIMAL(15, 2) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'KES',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 3. journal_entries

```sql
CREATE TABLE journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id VARCHAR(100) REFERENCES transactions(transaction_id),
    journal_account_id VARCHAR(20) REFERENCES journal_accounts(id),
    debit DECIMAL(15, 2),
    credit DECIMAL(15, 2),
    currency VARCHAR(3) DEFAULT 'KES',
    description TEXT,
    reference VARCHAR(100), -- Links paired entries
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_journal_entries_transaction_id ON journal_entries(transaction_id);
CREATE INDEX idx_journal_entries_journal_account_id ON journal_entries(journal_account_id);
CREATE INDEX idx_journal_entries_reference ON journal_entries(reference);

```

### 4. idempotency_keys

```sql
CREATE TABLE idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(100) UNIQUE NOT NULL,
    request_hash VARCHAR(64) NOT NULL, -- SHA-256 of request body
    response_body TEXT, -- Store successful response
    status_code INT,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL -- 24 hours from creation
);

CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);

```

### 5. logs

```sql
CREATE TABLE logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    correlation_id VARCHAR(100), -- Track request across services
    level VARCHAR(10) NOT NULL, -- INFO, WARN, ERROR
    message TEXT NOT NULL,
    metadata JSONB, -- Additional context
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_logs_level ON logs(level);
CREATE INDEX idx_logs_correlation_id ON logs(correlation_id);
CREATE INDEX idx_logs_created_at ON logs(created_at);

```

### 6. payments

```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_reference VARCHAR(100), -- The M-Pesa transaction reference, e.g. TSDF90FS1
    transaction_date TIMESTAMP,
    phone_number VARCHAR(15),
    amount DECIMAL(15, 2),
    raw_payload JSONB NOT NULL, -- Store full callback
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_processed ON payments(processed);

```

---

## API Specification (MVP)

### Authentication

All endpoints except `/health` and `/api/v1/payments/mpesa/callback` require JWT authentication.

**Header:**

```
Authorization: Bearer <JWT_TOKEN>

```

### 1. Initiate M-PESA STK Push

**Endpoint:** `POST /api/v1/payments/mpesa/stkpush`

**Request:**

```json
{
  "phone_number": "254712345678",
  "amount": 500,
  "account_reference": "RIDE123",
  "transaction_desc": "Payment for ride",
  "idempotency_key": "unique-key-123"
}

```

**Validation:**

- `phone_number`: Required, format `254XXXXXXXXX`
- `amount`: Required, min 1, max 150000
- `account_reference`: Required, max 100 chars
- `idempotency_key`: Required, max 100 chars

**Response (202 Accepted):**

```json
{
  "success": true,
  "data": {
    "transaction_id": "txn_abc123",
    "checkout_request_id": "ws_CO_12345",
    "status": "PENDING",
    "message": "STK Push sent successfully"
  }
}

```

**Error Response (400 Bad Request):**

```json
{
  "success": false,
  "error": {
    "code": "INVALID_PHONE",
    "message": "Phone number must start with 254"
  }
}

```

**Error Codes:**

- `INVALID_PHONE`: Invalid phone number format
- `INVALID_AMOUNT`: Amount out of range
- `DUPLICATE_REQUEST`: Idempotency key already used
- `MPESA_ERROR`: M-PESA API error

---

### 2. M-PESA Callback (Webhook)

**Endpoint:** `POST /api/v1/payments/mpesa/callback`

**No Authentication Required** (use signature verification instead)

**M-PESA Callback Payload Example:**

```json
{
  "TransactionType":"Customer Merchant Payment",
	"TransID":"TIG83V4802",
	"TransTime":"20250916001111",
	"TransAmount":"1.00",
	"BusinessShortCode":"513042",
	"BillRefNumber":"",
	"InvoiceNumber":"",
	"OrgAccountBalance":"83570.48",
	"ThirdPartyTransID":"",
	"MSISDN":"2200027aae415ab9797a3911a4253e618cb725467d56e1dd1d6308f75734b835","FirstName":"BRIAN",
	"MiddleName":"",
	"LastName":""
}

```

**Response (200 OK):**

```json
{
  "ResultCode": 0,
  "ResultDesc": "Accepted"
}

```

---

### 3. Check Transaction Status

**Endpoint:** `GET /api/v1/payments/:transaction_id/status`

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "transaction_id": "txn_abc123",
    "status": "SUCCESS",
    "amount": 500,
    "currency": "KES",
    "phone_number": "254712345678",
    "payment_reference": "NLJ7RT61SV",
    "completed_at": "2025-09-30T10:30:00Z"
  }
}

```

**Status Values:**

- `PENDING`: Payment initiated, waiting for user action
- `SUCCESS`: Payment completed
- `FAILED`: Payment failed
- `EXPIRED`: STK Push expired (user didn't enter PIN)

---

### 4. Get Account Balance

**Endpoint:** `GET /api/v1/accounts/:account_id/balance`

**Query Params:**

- `account_code` (optional): Filter by specific account (e.g., `ASSET_001`)

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "user_id": "user_123",
    "accounts": [
      {
        "account_id": "ASSET_001",
        "account_name": "Customer Wallet",
        "balance": 1500.00,
        "currency": "KES"
      }
    ]
  }
}

```

---

### 5. Health Check

**Endpoint:** `GET /api/v1/health`

**No Authentication Required**

**Response (200 OK):**

```json
{
  "status": "healthy",
  "timestamp": "2025-09-30T10:30:00Z",
  "database": "connected",
  "version": "1.0.0"
}

```

---

## Project Structure

```
mkwanja-payment-service/
├── main.go                 # Application entry point
├── config/
│   └── config.go                   # Configuration management
│   └── database.go                  
├── internal/
│   ├── handlers/                   # HTTP handlers
│   │   ├── payment_handler.go
│   │   ├── ledger_handler.go
│   │   └── health_handler.go
│   ├── services/                   # Business logic
│   │   ├── mpesa_service.go
│   │   ├── ledger_service.go
│   │   └── idempotency_service.go
│   ├── repositories/               # Database access
│   │   ├── transaction_repo.go
│   │   ├── ledger_repo.go
│   │   └── log_repo.go
│   ├── models/                     # Data models
│   │   ├── transaction.go
│   │   ├── journal.go
│   │   └── mpesa.go
│   ├── middleware/                 # HTTP middleware
│   │   ├── auth.go
│   │   ├── rate_limit.go
│   │   └── logger.go
│   └── routes/                      # Routes
│       ├── api.go
│   └── utils/                      # Utilities
│       ├── jwt.go
│       ├── validator.go
│       └── logger.go
│   └── tests/
├── pkg/                            # Shared/external packages
│   └── mpesa/
│       └── client.go               # M-PESA API client
├── db/migrations/                     # Database migrations
│   ├── 001_create_transactions.sql
│   ├── 002_create_journal.sql
│   └── ...
├── sample.env                    # Environment variables template
├── go.mod
├── go.sum
└── README.md

```

---

## Environment Variables (MVP)

```bash
# Application
APP_ENV=development
APP_PORT=8080
APP_NAME=mkwanja-payment-service

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=mkwanja
DB_PASSWORD=secure_password
DB_NAME=mkwanja_payments
DB_SSL_MODE=disable

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRY=3600                    # 1 hour in seconds
JWT_REFRESH_EXPIRY=604800          # 7 days in seconds

# M-PESA (Daraja API)
MPESA_CONSUMER_KEY=your_consumer_key
MPESA_CONSUMER_SECRET=your_consumer_secret
MPESA_SHORTCODE=174379             # Your paybill/till number
MPESA_PASSKEY=your_passkey
MPESA_CALLBACK_URL=https://yourdomain.com/api/v1/payments/mpesa/callback
MPESA_ENVIRONMENT=sandbox          # sandbox or production

# Logging
LOG_LEVEL=info                     # debug, info, warn, error
LOG_FORMAT=json                    # json or text

# Rate Limiting (MVP - in-memory)
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60               # seconds

```

---

## Error Handling Strategy (MVP)

### Error Response Format

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}  // Optional additional context
  }
}

```

### Standard Error Codes

- `INVALID_REQUEST`: Malformed request body
- `UNAUTHORIZED`: Missing or invalid JWT token
- `FORBIDDEN`: Valid token but insufficient permissions
- `NOT_FOUND`: Resource not found
- `DUPLICATE_REQUEST`: Idempotency key reused
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `MPESA_ERROR`: M-PESA API error
- `LEDGER_ERROR`: Ledger operation failed
- `INTERNAL_ERROR`: Unexpected server error

### Logging Errors

- All errors logged with correlation ID
- ERROR level for 5xx errors
- WARN level for 4xx errors
- Include stack trace for 5xx errors

---

## Security Checklist (MVP)

- [ ]  HTTPS/TLS enforced in production
- [ ]  JWT tokens with secure secret key
- [ ]  M-PESA callback signature verification
- [ ]  Input validation on all endpoints
- [ ]  SQL injection prevention (GORM parameterized queries)
- [ ]  Rate limiting (100 req/min per user)
- [ ]  Environment variables for secrets (no hardcoding)
- [ ]  CORS policy configured
- [ ]  Request timeout (30 seconds)
- [ ]  Database connection pooling with limits

---

## Deployment Checklist (MVP)

- [ ]  Database migrations applied
- [ ]  Environment variables configured
- [ ]  M-PESA sandbox testing completed
- [ ]  Health check endpoint working
- [ ]  Logging configured and working
- [ ]  SSL certificate installed
- [ ]  Callback URL registered with M-PESA
- [ ]  Database backups configured
- [ ]  Monitoring basic logs
- [ ]  Documentation updated

---

## Success Metrics (MVP)

- **Payment Success Rate:** > 95%
- **API Response Time:** p95 < 2 seconds
- **System Uptime:** > 99%
- **Ledger Accuracy:** 100% (all entries balanced)
- **Zero duplicate payments** (idempotency working)

---

## Next Steps After MVP

**Post-MVP Roadmap:**

- Phase 2: Enhanced reliability (Kafka, monitoring)
- Phase 3: Mookh and Flutterwave integration
- Phase 4: Advanced features (refunds, reconciliation)
- Phase 5: Testing and scale improvements