-- name: CreateClient :one
INSERT INTO clients (external_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetClientByID :one
SELECT * FROM clients WHERE id = $1;

-- name: GetClientByExternalID :one
SELECT * FROM clients WHERE external_id = $1;

-- name: ListClients :many
SELECT * FROM clients WHERE active = TRUE ORDER BY created_at DESC;

-- name: DeactivateClient :one
UPDATE clients SET active = FALSE, updated_at = NOW()
WHERE id = $1
RETURNING *;
