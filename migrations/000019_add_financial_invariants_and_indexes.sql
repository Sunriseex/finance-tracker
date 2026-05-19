-- +goose Up
ALTER TABLE transactions
    ADD CONSTRAINT transactions_transfer_type_id_check CHECK (
        (type IN ('transfer_in', 'transfer_out') AND transfer_id IS NOT NULL)
        OR
        (type NOT IN ('transfer_in', 'transfer_out') AND transfer_id IS NULL)
    );

ALTER TABLE interest_accruals
    ADD CONSTRAINT interest_accruals_transaction_id_unique UNIQUE (transaction_id);

CREATE INDEX transactions_related_account_id_idx ON transactions (related_account_id);
CREATE INDEX transactions_category_id_idx ON transactions (category_id);
CREATE INDEX interest_accruals_rule_id_idx ON interest_accruals (rule_id);
CREATE INDEX categories_parent_id_idx ON categories (parent_id);

-- +goose StatementBegin
CREATE FUNCTION validate_transfer_integrity(p_transfer_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    invalid_count INTEGER;
BEGIN
    IF p_transfer_id IS NULL THEN
        RETURN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM transfers WHERE id = p_transfer_id) THEN
        RETURN;
    END IF;

    SELECT COUNT(*)
    INTO invalid_count
    FROM transfers tr
    WHERE tr.id = p_transfer_id
      AND (
          (SELECT COUNT(*) FROM transactions tx WHERE tx.transfer_id = tr.id) <> 2
          OR NOT EXISTS (
              SELECT 1
              FROM transactions out_tx
              JOIN transactions in_tx ON in_tx.id = tr.to_transaction_id
              WHERE out_tx.id = tr.from_transaction_id
                AND out_tx.transfer_id = tr.id
                AND in_tx.transfer_id = tr.id
                AND out_tx.type = 'transfer_out'
                AND in_tx.type = 'transfer_in'
                AND out_tx.account_id = tr.from_account_id
                AND in_tx.account_id = tr.to_account_id
                AND out_tx.related_account_id = tr.to_account_id
                AND in_tx.related_account_id = tr.from_account_id
                AND out_tx.amount_minor = tr.from_amount_minor
                AND in_tx.amount_minor = tr.to_amount_minor
          )
      );

    IF invalid_count > 0 THEN
        RAISE EXCEPTION 'invalid transfer invariant for transfer %', p_transfer_id
            USING ERRCODE = '23514';
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION validate_transfer_integrity_from_transfer()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM validate_transfer_integrity(NEW.id);
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION validate_transfer_integrity_from_transaction()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM validate_transfer_integrity(OLD.transfer_id);
        RETURN OLD;
    END IF;

    PERFORM validate_transfer_integrity(NEW.transfer_id);
    IF TG_OP = 'UPDATE' AND OLD.transfer_id IS DISTINCT FROM NEW.transfer_id THEN
        PERFORM validate_transfer_integrity(OLD.transfer_id);
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER transfers_integrity_check
AFTER INSERT OR UPDATE ON transfers
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_transfer_integrity_from_transfer();

CREATE CONSTRAINT TRIGGER transactions_transfer_integrity_check
AFTER INSERT OR UPDATE OR DELETE ON transactions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION validate_transfer_integrity_from_transaction();

-- +goose Down
DROP TRIGGER IF EXISTS transactions_transfer_integrity_check ON transactions;
DROP TRIGGER IF EXISTS transfers_integrity_check ON transfers;
DROP FUNCTION IF EXISTS validate_transfer_integrity_from_transaction();
DROP FUNCTION IF EXISTS validate_transfer_integrity_from_transfer();
DROP FUNCTION IF EXISTS validate_transfer_integrity(UUID);

DROP INDEX IF EXISTS categories_parent_id_idx;
DROP INDEX IF EXISTS interest_accruals_rule_id_idx;
DROP INDEX IF EXISTS transactions_category_id_idx;
DROP INDEX IF EXISTS transactions_related_account_id_idx;

ALTER TABLE interest_accruals
    DROP CONSTRAINT IF EXISTS interest_accruals_transaction_id_unique;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_transfer_type_id_check;
