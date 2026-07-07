-- name: CreatePaystackCredentials :one
INSERT INTO client_paystack_credentials (
    client_id, secret_key_encrypted, public_key
) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetActivePaystackCredentials :one
SELECT * FROM client_paystack_credentials
WHERE client_id = $1 AND is_active = TRUE;

-- name: DeactivatePaystackCredentials :exec
UPDATE client_paystack_credentials SET is_active = FALSE
WHERE client_id = $1 AND is_active = TRUE;
