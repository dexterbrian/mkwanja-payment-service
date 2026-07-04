-- +goose Up
CREATE TABLE client_credentials (
    id                            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    client_id                     TEXT NOT NULL REFERENCES clients(id),
    shortcode                     TEXT NOT NULL,
    consumer_key_encrypted        TEXT NOT NULL,
    consumer_secret_encrypted     TEXT NOT NULL,
    passkey_encrypted             TEXT NOT NULL,
    initiator_name                TEXT,
    security_credential_encrypted TEXT,
    is_active                     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_client_credentials_one_active
    ON client_credentials(client_id) WHERE (is_active = TRUE);

-- +goose Down
DROP TABLE client_credentials;
