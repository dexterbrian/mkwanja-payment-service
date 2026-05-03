-- +goose Up
CREATE TYPE account_type   AS ENUM ('asset', 'liability', 'revenue', 'expense', 'equity');
CREATE TYPE normal_balance AS ENUM ('debit', 'credit');

CREATE TABLE journal_accounts (
    id             TEXT NOT NULL,
    business_id    TEXT NOT NULL REFERENCES businesses(id),
    name           TEXT NOT NULL,
    account_type   account_type NOT NULL,
    normal_balance normal_balance NOT NULL,
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (business_id, id)
);

-- +goose Down
DROP TABLE journal_accounts;
DROP TYPE normal_balance;
DROP TYPE account_type;
