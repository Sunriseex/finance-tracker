-- +goose Up
ALTER TABLE users
ADD COLUMN primary_currency CHAR(3) NOT NULL DEFAULT 'RUB' CHECK (primary_currency ~ '^[A-Z]{3}$');

-- +goose Down
ALTER TABLE users
DROP COLUMN IF EXISTS primary_currency;
