-- +goose Up
CREATE TABLE idempotency_keys (
    key TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status_code INTEGER,
    response_body BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (key, user_id, method, path)
);

CREATE INDEX idempotency_keys_expires_at_idx ON idempotency_keys(expires_at);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
