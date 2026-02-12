# Code Design: Mkwanja Payment Service

## 1. Project Structure

The service follows a standard Go project layout:

- `cmd/api/`: Entry point for the Fiber application.
- `internal/`:
  - `handler/`: HTTP request handlers (parsing input, calling service layer).
  - `service/`: Business logic (idempotency checks, Daraja orchestration).
  - `repository/`: Data access (PostgreSQL for logs, Redis for caching).
  - `integration/`: External API clients (Daraja, Supabase Auth).
  - `middleware/`: JWT verification, Logging, Recover.
- `pkg/`: Reusable utilities (config, logger, signature helpers).

## 2. Transaction Flow (Safe Execution)

```mermaid
sequenceDiagram
    participant App as Flutter App
    participant SVC as Payment Service
    participant DB as PostgreSQL
    participant R as Redis
    participant MP as M-PESA Daraja

    App->>SVC: POST /payment (JWT + X-Idempotency-Key)
    SVC->>R: GET idempotency_key
    R-->>SVC: Not Found (or Cached Response)
    SVC->>DB: INSERT transaction (Pending)
    SVC->>MP: Initiate Request (STK/B2C)
    MP-->>SVC: ACK (RequestID)
    SVC->>DB: UPDATE transaction (external_ref=RequestID)
    SVC->>App: 202 Accepted
    Note over SVC,MP: Wait for Callback
    MP->>SVC: POST /callback
    SVC->>SVC: Verify Signature
    SVC->>DB: UPDATE transaction (Success/Failed)
    SVC->>R: SET idempotency_key (Result)
```

## 3. Reliability Patterns

- **Panics:** Use Fiber's `Recover` middleware to prevent service crashes on unexpected errors.
- **Timeouts:** All external calls (Daraja, DB) must have strict context timeouts.
- **Logging:** Structured logging (using `zerolog` or `zap`) including RequestID for tracing. (TODO: determine which one to use based on pros and cons and performance and future scalability and maintainability)

## 4. Idempotency Implementation
- Hash the request body + user ID + idempotency key to create a unique cache key.
- Use atomic operations (`SETNX`) in Redis if marking a key as "In Progress".
