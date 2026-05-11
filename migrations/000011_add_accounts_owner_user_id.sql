-- +goose Up
ALTER TABLE accounts
ADD COLUMN owner_user_id UUID REFERENCES users(id) ON DELETE CASCADE;

UPDATE accounts
SET owner_user_id = (SELECT id FROM users LIMIT 1)
WHERE owner_user_id IS NULL
  AND (SELECT count(*) FROM users) = 1;

CREATE INDEX accounts_owner_user_id_idx ON accounts(owner_user_id);

-- +goose Down
DROP INDEX IF EXISTS accounts_owner_user_id_idx;

ALTER TABLE accounts
DROP COLUMN IF EXISTS owner_user_id;
