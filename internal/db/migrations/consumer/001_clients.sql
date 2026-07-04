-- +goose Up
CREATE TABLE clients (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    external_id TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_clients_external_id ON clients(external_id);

-- +goose Down
DROP TABLE clients;
