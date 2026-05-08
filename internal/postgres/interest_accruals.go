package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/finance-manager/internal/models"
)

type InterestAccrualRepository struct {
	pool *pgxpool.Pool
}

func NewInterestAccrualRepository(pool *pgxpool.Pool) *InterestAccrualRepository {
	return &InterestAccrualRepository{pool: pool}
}

func (r *InterestAccrualRepository) Create(ctx context.Context, accrual *models.InterestAccrual) error {
	if err := insertInterestAccrual(ctx, r.pool, accrual); err != nil {
		return fmt.Errorf("create interest accrual: %w", err)
	}
	return nil
}

func (r *InterestAccrualRepository) CreateWithTransaction(ctx context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin interest accrual transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := insertTransaction(ctx, tx, transaction); err != nil {
		return fmt.Errorf("create interest transaction: %w", err)
	}
	if err := insertInterestAccrual(ctx, tx, accrual); err != nil {
		return fmt.Errorf("create interest accrual: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit interest accrual transaction: %w", err)
	}
	return nil
}

func (r *InterestAccrualRepository) ReplaceRangeWithTransactions(ctx context.Context, accountID, ruleID string, fromDate, toDate time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin replace interest accruals: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `
		DELETE FROM interest_accruals
		WHERE account_id = $1 AND rule_id = $2 AND accrual_date BETWEEN $3 AND $4
		RETURNING transaction_id
	`, accountID, ruleID, fromDate, toDate)
	if err != nil {
		return 0, fmt.Errorf("delete interest accruals: %w", err)
	}

	var transactionIDs []string
	for rows.Next() {
		var transactionID string
		if err := rows.Scan(&transactionID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan deleted interest transaction id: %w", err)
		}
		transactionIDs = append(transactionIDs, transactionID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("delete interest accruals rows: %w", err)
	}
	rows.Close()

	for _, transactionID := range transactionIDs {
		if _, err := tx.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND type = $2`, transactionID, models.TransactionTypeInterestIncome); err != nil {
			return 0, fmt.Errorf("delete interest transaction %s: %w", transactionID, err)
		}
	}

	if len(transactions) != len(accruals) {
		return 0, fmt.Errorf("replace interest accruals: transactions and accruals length mismatch")
	}

	for i := range transactions {
		if err := insertTransaction(ctx, tx, &transactions[i]); err != nil {
			return 0, fmt.Errorf("create recalculated interest transaction: %w", err)
		}
		if err := insertInterestAccrual(ctx, tx, &accruals[i]); err != nil {
			return 0, fmt.Errorf("create recalculated interest accrual: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit replace interest accruals: %w", err)
	}

	return int64(len(transactionIDs)), nil
}

func (r *InterestAccrualRepository) GetByAccountDateRule(ctx context.Context, accountID, accrualDate, ruleID string) (*models.InterestAccrual, error) {
	accrual, err := scanInterestAccrual(r.pool.QueryRow(ctx, selectInterestAccrualSQL+` WHERE account_id = $1 AND accrual_date = $2 AND rule_id = $3`, accountID, accrualDate, ruleID))
	if err != nil {
		return nil, fmt.Errorf("get interest accrual: %w", mapNotFound(err))
	}
	return accrual, nil
}

func (r *InterestAccrualRepository) ListByAccount(ctx context.Context, accountID string) ([]models.InterestAccrual, error) {
	rows, err := r.pool.Query(ctx, selectInterestAccrualSQL+` WHERE account_id = $1 ORDER BY accrual_date`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list interest accruals: %w", err)
	}
	defer rows.Close()

	var accruals []models.InterestAccrual
	for rows.Next() {
		accrual, err := scanInterestAccrual(rows)
		if err != nil {
			return nil, err
		}
		accruals = append(accruals, *accrual)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list interest accruals rows: %w", err)
	}
	return accruals, nil
}

const selectInterestAccrualSQL = `
	SELECT id, account_id, rule_id, transaction_id, accrual_date, amount_minor, balance_minor, annual_rate_bps, created_at
	FROM interest_accruals
`

type interestAccrualScanner interface {
	Scan(dest ...any) error
}

func scanInterestAccrual(row interestAccrualScanner) (*models.InterestAccrual, error) {
	var accrual models.InterestAccrual
	if err := row.Scan(
		&accrual.ID,
		&accrual.AccountID,
		&accrual.RuleID,
		&accrual.TransactionID,
		&accrual.AccrualDate,
		&accrual.AmountMinor,
		&accrual.BalanceMinor,
		&accrual.AnnualRateBps,
		&accrual.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan interest accrual: %w", mapNotFound(err))
	}
	return &accrual, nil
}

func insertInterestAccrual(ctx context.Context, execer sqlExecer, accrual *models.InterestAccrual) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO interest_accruals (id, account_id, rule_id, transaction_id, accrual_date, amount_minor, balance_minor, annual_rate_bps, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, accrual.ID, accrual.AccountID, accrual.RuleID, accrual.TransactionID, accrual.AccrualDate, accrual.AmountMinor, accrual.BalanceMinor, accrual.AnnualRateBps, accrual.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert interest accrual: %w", err)
	}
	return nil
}
