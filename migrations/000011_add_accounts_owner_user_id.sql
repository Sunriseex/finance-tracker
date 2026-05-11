-- +goose Up
ALTER TABLE accounts
ADD COLUMN owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX accounts_owner_user_id_idx ON accounts(owner_user_id);

-- +goose Down
DROP INDEX IF EXISTS accounts_owner_user_id_idx;

ALTER TABLE accounts
DROP COLUMN IF EXISTS owner_user_id;
