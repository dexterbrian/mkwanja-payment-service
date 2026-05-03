-- +goose Up
CREATE TYPE entry_type AS ENUM ('debit', 'credit');

CREATE TABLE journal (
    id           BIGSERIAL PRIMARY KEY,
    business_id  TEXT NOT NULL,
    payment_id   TEXT NOT NULL REFERENCES payments(id),
    account_id   TEXT NOT NULL,
    entry_type   entry_type NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    currency     TEXT NOT NULL DEFAULT 'KES',
    description  TEXT NOT NULL,
    reversal_of  BIGINT REFERENCES journal(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (business_id, account_id) REFERENCES journal_accounts(business_id, id)
);

CREATE RULE no_update_journal AS ON UPDATE TO journal DO INSTEAD NOTHING;
CREATE RULE no_delete_journal AS ON DELETE TO journal DO INSTEAD NOTHING;

CREATE INDEX idx_journal_business_id ON journal(business_id);
CREATE INDEX idx_journal_payment_id  ON journal(payment_id);
CREATE INDEX idx_journal_account_id  ON journal(account_id);
CREATE INDEX idx_journal_created_at  ON journal(created_at DESC);

CREATE VIEW account_balances AS
SELECT
    business_id,
    account_id,
    SUM(CASE WHEN entry_type = 'debit'  THEN amount_cents ELSE 0 END) AS total_debits_cents,
    SUM(CASE WHEN entry_type = 'credit' THEN amount_cents ELSE 0 END) AS total_credits_cents,
    SUM(CASE WHEN entry_type = 'debit'  THEN amount_cents ELSE -amount_cents END) AS net_cents
FROM journal
GROUP BY business_id, account_id;

-- +goose Down
DROP VIEW account_balances;
DROP TABLE journal;
DROP TYPE entry_type;
