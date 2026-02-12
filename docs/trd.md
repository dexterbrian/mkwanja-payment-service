# Technical Requirements Document: Mkwanja Payment Service

## 1. System Architecture

The Mkwanja Payment Service is a Go-based microservice designed for transaction orchestration. It lives between the Mkwanja App/Supabase and the M-PESA Daraja ecosystem.

### 1.1 Components
- **API (Fiber):** RESTful endpoints for initiating transactions and receiving callbacks.
- **Worker/Processor:** Internal logic for handling async tasks and retries.
- **Interactions:**
  - **Inbound:** REST API calls with JWT (Bearer).
  - **Outbound:** HTTP calls to M-PESA Daraja.
  - **Storage:** Redis (Idempotency) and PostgreSQL (Shared DB).

---

## 2. Technical Requirements

### 2.1 Framework & Runtime
- **Runtime:** Go 1.21+
- **Web Framework:** Fiber v2 (Fast, Express-like).
- **HTTP Client:** `resty` or standard `net/http` for Daraja API calls. (TODO: determine which one to use based on pros and cons and performance and future scalability and maintainability)

### 2.2 Security
- **JWT Validation:** Use Supabase's public key (retrieved or configured) to validate signatures on all incoming requests.
- **Daraja Auth:** Implement OAuth2 flow for Daraja to obtain and refresh Access Tokens.
- **Signature Verification:** HMAC-SHA256 signature verification for Daraja callbacks where applicable.
- **Rate Limiting:** Implement per-user rate limiting using Fiber's middleware.

### 2.3 Idempotency Logic
- **Key Generation:** Client (Flutter App) generates and sends `X-Idempotency-Key` (UUID).
- **Mechanism:**
  1. Check Redis for the key. If exists, return cached response.
  2. If not, mark key as "In Progress" in Redis.
  3. Execute transaction.
  4. Store result in Redis with a 24-hour TTL.
- **Conflict Handling:** Return `409 Conflict` if a request with the same key is already "In Progress".

---

## 3. API Endpoints

### 3.1 Payment Initiation
- `POST /v1/payments/stk-push`: Initiate deposit.
- `POST /v1/payments/b2c`: Initiate withdrawal.
- `POST /v1/payments/b2b`: Initiate utility/merchant payment.

### 3.2 Callbacks (Public)
- `POST /v1/callbacks/mpesa/stk`: Target for STK callbacks.
- `POST /v1/callbacks/mpesa/b2c`: Target for B2C callbacks.
- `POST /v1/callbacks/mpesa/b2b`: Target for B2B callbacks.

---

## 4. Error Handling
- Use structured JSON error responses containing error (boolean and required), message (string), and data (object).
- Map M-PESA error codes to internal, user-friendly codes.
- Log full error traces for internal monitoring.
