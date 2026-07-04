-- name: CreateJournalAccount :one
INSERT INTO journal_accounts (id, client_id, name, account_type, normal_balance, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetJournalAccount :one
SELECT * FROM journal_accounts WHERE client_id = $1 AND id = $2;

-- name: ListJournalAccounts :many
SELECT * FROM journal_accounts WHERE client_id = $1 ORDER BY id;

-- name: GetAccountBalances :many
SELECT * FROM account_balances WHERE client_id = $1;
