package postgres

import (
	"context"
	"fmt"

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
	_, err := r.pool.Exec(ctx, `
		INSERT INTO interest_accruals (id, account_id, rule_id, transaction_id, accrual_date, amount_minor, balance_minor, annual_rate_bps, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, accrual.ID, accrual.AccountID, accrual.RuleID, accrual.TransactionID, accrual.AccrualDate, accrual.AmountMinor, accrual.BalanceMinor, accrual.AnnualRateBps, accrual.CreatedAt)
	if err != nil {
		return fmt.Errorf("create interest accrual: %w", err)
	}
	return nil
}

func (r *InterestAccrualRepository) GetByAccountDateRule(ctx context.Context, accountID string, accrualDate string, ruleID string) (*models.InterestAccrual, error) {
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
