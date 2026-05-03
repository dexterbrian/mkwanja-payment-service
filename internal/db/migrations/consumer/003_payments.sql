-- +goose Up
CREATE TYPE payment_provider  AS ENUM ('mpesa', 'stripe');
CREATE TYPE payment_direction AS ENUM ('inbound', 'outbound');
CREATE TYPE payment_type      AS ENUM ('stk_push', 'b2c', 'b2b', 'c2b');
CREATE TYPE payment_status    AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled', 'reversed');

CREATE TABLE payments (
    id                  TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    business_id         TEXT NOT NULL REFERENCES businesses(id),
    idempotency_key     TEXT NOT NULL,
    provider            payment_provider NOT NULL DEFAULT 'mpesa',
    payment_type        payment_type NOT NULL,
    direction           payment_direction NOT NULL,
    status              payment_status NOT NULL DEFAULT 'pending',
    amount_cents        BIGINT NOT NULL CHECK (amount_cents > 0),
    currency            TEXT NOT NULL DEFAULT 'KES',
    phone_number        TEXT,
    receiver_shortcode  TEXT,
    reference           TEXT NOT NULL,
    description         TEXT,
    provider_request_id TEXT,
    provider_tx_id      TEXT,
    provider_receipt    TEXT,
    provider_raw        JSONB,
    callback_delivered  BOOLEAN NOT NULL DEFAULT FALSE,
    metadata            JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    UNIQUE(business_id, idempotency_key)
);

CREATE TABLE payment_events (
    id          BIGSERIAL PRIMARY KEY,
    payment_id  TEXT NOT NULL REFERENCES payments(id),
    from_status payment_status,
    to_status   payment_status NOT NULL,
    reason      TEXT,
    raw         JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE RULE no_update_payment_events AS ON UPDATE TO payment_events DO INSTEAD NOTHING;
CREATE RULE no_delete_payment_events AS ON DELETE TO payment_events DO INSTEAD NOTHING;

CREATE INDEX idx_payments_business_id ON payments(business_id);
CREATE INDEX idx_payments_status      ON payments(status);
CREATE INDEX idx_payments_provider_tx ON payments(provider_tx_id);
CREATE INDEX idx_payments_created_at  ON payments(created_at DESC);

-- +goose Down
DROP TABLE payment_events;
DROP TABLE payments;
DROP TYPE payment_status;
DROP TYPE payment_type;
DROP TYPE payment_direction;
DROP TYPE payment_provider;
