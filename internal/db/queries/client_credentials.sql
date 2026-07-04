-- name: CreateCredentials :one
INSERT INTO client_credentials (
    client_id, shortcode, consumer_key_encrypted, consumer_secret_encrypted,
    passkey_encrypted, initiator_name, security_credential_encrypted
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetActiveCredentials :one
SELECT * FROM client_credentials
WHERE client_id = $1 AND is_active = TRUE;

-- name: DeactivateCredentials :exec
UPDATE client_credentials SET is_active = FALSE
WHERE client_id = $1 AND is_active = TRUE;
