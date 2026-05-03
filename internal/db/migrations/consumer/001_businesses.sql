-- +goose Up
CREATE TABLE businesses (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    external_id TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_businesses_external_id ON businesses(external_id);

-- +goose Down
DROP TABLE businesses;
