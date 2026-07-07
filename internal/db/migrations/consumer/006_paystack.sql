-- +goose Up
-- Paystack support: businesses bring their own Paystack API keys
-- (same BYO-credentials model as Daraja).
ALTER TYPE payment_provider ADD VALUE IF NOT EXISTS 'paystack';
ALTER TYPE payment_type ADD VALUE IF NOT EXISTS 'paystack_charge';

CREATE TABLE client_paystack_credentials (
    id                   TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    client_id            TEXT NOT NULL REFERENCES clients(id),
    secret_key_encrypted TEXT NOT NULL,
    public_key           TEXT NOT NULL,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_client_paystack_credentials_one_active
    ON client_paystack_credentials(client_id) WHERE (is_active = TRUE);

-- +goose Down
DROP TABLE client_paystack_credentials;
