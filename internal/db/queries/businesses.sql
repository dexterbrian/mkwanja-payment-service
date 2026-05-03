-- name: CreateBusiness :one
INSERT INTO businesses (external_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetBusinessByID :one
SELECT * FROM businesses WHERE id = $1;

-- name: GetBusinessByExternalID :one
SELECT * FROM businesses WHERE external_id = $1;

-- name: ListBusinesses :many
SELECT * FROM businesses WHERE active = TRUE ORDER BY created_at DESC;

-- name: DeactivateBusiness :one
UPDATE businesses SET active = FALSE, updated_at = NOW()
WHERE id = $1
RETURNING *;
