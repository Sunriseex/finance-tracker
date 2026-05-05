package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/repository"
)

type InterestRuleRepository struct {
	pool *pgxpool.Pool
}

func NewInterestRuleRepository(pool *pgxpool.Pool) *InterestRuleRepository {
	return &InterestRuleRepository{pool: pool}
}

func (r *InterestRuleRepository) Create(ctx context.Context, rule *models.InterestRule) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO interest_rules (
			id, account_id, annual_rate_bps, promo_rate_bps, promo_end_date,
			accrual_frequency, capitalization_frequency, day_count_convention,
			is_active, start_date, end_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, rule.ID, rule.AccountID, rule.AnnualRateBps, rule.PromoRateBps, rule.PromoEndDate, rule.AccrualFrequency, rule.CapitalizationFrequency, rule.DayCountConvention, rule.IsActive, rule.StartDate, rule.EndDate)
	if err != nil {
		return fmt.Errorf("create interest rule: %w", err)
	}
	return nil
}

func (r *InterestRuleRepository) GetByID(ctx context.Context, id string) (*models.InterestRule, error) {
	rule, err := scanInterestRule(r.pool.QueryRow(ctx, selectInterestRuleSQL+` WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get interest rule: %w", mapNotFound(err))
	}
	return rule, nil
}

func (r *InterestRuleRepository) ListByAccount(ctx context.Context, accountID string) ([]models.InterestRule, error) {
	rows, err := r.pool.Query(ctx, selectInterestRuleSQL+` WHERE account_id = $1 ORDER BY start_date, created_at`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list interest rules: %w", err)
	}
	defer rows.Close()

	var rules []models.InterestRule
	for rows.Next() {
		rule, err := scanInterestRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list interest rules rows: %w", err)
	}
	return rules, nil
}

func (r *InterestRuleRepository) Update(ctx context.Context, rule *models.InterestRule) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE interest_rules
		SET annual_rate_bps = $2, promo_rate_bps = $3, promo_end_date = $4,
			accrual_frequency = $5, capitalization_frequency = $6, day_count_convention = $7,
			is_active = $8, start_date = $9, end_date = $10, updated_at = now()
		WHERE id = $1
	`, rule.ID, rule.AnnualRateBps, rule.PromoRateBps, rule.PromoEndDate, rule.AccrualFrequency, rule.CapitalizationFrequency, rule.DayCountConvention, rule.IsActive, rule.StartDate, rule.EndDate)
	if err != nil {
		return fmt.Errorf("update interest rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update interest rule: %w", repository.ErrNotFound)
	}
	return nil
}

const selectInterestRuleSQL = `
	SELECT id, account_id, annual_rate_bps, promo_rate_bps, promo_end_date,
		accrual_frequency, capitalization_frequency, day_count_convention,
		is_active, start_date, end_date
	FROM interest_rules
`

type interestRuleScanner interface {
	Scan(dest ...any) error
}

func scanInterestRule(row interestRuleScanner) (*models.InterestRule, error) {
	var rule models.InterestRule
	if err := row.Scan(
		&rule.ID,
		&rule.AccountID,
		&rule.AnnualRateBps,
		&rule.PromoRateBps,
		&rule.PromoEndDate,
		&rule.AccrualFrequency,
		&rule.CapitalizationFrequency,
		&rule.DayCountConvention,
		&rule.IsActive,
		&rule.StartDate,
		&rule.EndDate,
	); err != nil {
		return nil, fmt.Errorf("scan interest rule: %w", mapNotFound(err))
	}
	return &rule, nil
}
