-- name: CreatePayment :one
INSERT INTO payments (
    business_id, idempotency_key, provider, payment_type, direction,
    amount_cents, currency, phone_number, receiver_shortcode,
    reference, description, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1;

-- name: GetPaymentByIdempotencyKey :one
SELECT * FROM payments WHERE business_id = $1 AND idempotency_key = $2;

-- name: UpdatePaymentStatus :one
UPDATE payments SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CompletePayment :one
UPDATE payments
SET status = 'completed', provider_receipt = $2, provider_tx_id = $3,
    completed_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailPayment :one
UPDATE payments SET status = 'failed', updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProviderRequestID :one
UPDATE payments SET provider_request_id = $2, provider_tx_id = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListPaymentsByBusiness :many
SELECT * FROM payments
WHERE business_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPendingPaymentsOlderThan :many
SELECT * FROM payments
WHERE status = 'pending' AND created_at < $1
ORDER BY created_at ASC;

-- name: CreatePaymentEvent :one
INSERT INTO payment_events (payment_id, from_status, to_status, reason, raw)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
