-- +goose Up
CREATE TABLE transfers (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    to_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    from_transaction_id UUID NOT NULL,
    to_transaction_id UUID NOT NULL,
    from_amount_minor BIGINT NOT NULL CHECK (from_amount_minor > 0),
    to_amount_minor BIGINT NOT NULL CHECK (to_amount_minor > 0),
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    exchange_rate NUMERIC(36, 18) NOT NULL CHECK (exchange_rate > 0),
    exchange_rate_provider TEXT NOT NULL,
    exchange_rate_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transfers_distinct_accounts_check CHECK (from_account_id <> to_account_id),
    CONSTRAINT transfers_distinct_transactions_check CHECK (from_transaction_id <> to_transaction_id),
    CONSTRAINT transfers_from_transaction_fk FOREIGN KEY (from_transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT transfers_to_transaction_fk FOREIGN KEY (to_transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
);

ALTER TABLE transactions
    ADD COLUMN transfer_id UUID REFERENCES transfers(id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX transfers_from_transaction_id_idx ON transfers (from_transaction_id);
CREATE UNIQUE INDEX transfers_to_transaction_id_idx ON transfers (to_transaction_id);
CREATE INDEX transfers_user_id_created_at_idx ON transfers (user_id, created_at);
CREATE INDEX transactions_transfer_id_idx ON transactions (transfer_id);

-- +goose Down
DROP INDEX IF EXISTS transactions_transfer_id_idx;
DROP INDEX IF EXISTS transfers_user_id_created_at_idx;
DROP INDEX IF EXISTS transfers_to_transaction_id_idx;
DROP INDEX IF EXISTS transfers_from_transaction_id_idx;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS transfer_id;

DROP TABLE IF EXISTS transfers;
