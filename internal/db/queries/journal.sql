-- name: CreateJournalEntry :one
INSERT INTO journal (business_id, payment_id, account_id, entry_type, amount_cents, currency, description, reversal_of)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListJournalEntriesByBusiness :many
SELECT * FROM journal
WHERE business_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListJournalEntriesByPayment :many
SELECT * FROM journal WHERE payment_id = $1 ORDER BY id;

-- name: GetTrialBalance :many
SELECT * FROM account_balances WHERE business_id = $1;
