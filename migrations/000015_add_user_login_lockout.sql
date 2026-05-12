-- +goose Up
ALTER TABLE users
    ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN locked_until TIMESTAMPTZ;

CREATE INDEX users_locked_until_idx
    ON users (locked_until)
    WHERE locked_until IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users_locked_until_idx;

ALTER TABLE users
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS failed_login_attempts;
