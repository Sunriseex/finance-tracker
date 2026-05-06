package postgres

import (
	"context"
	"fmt"

	"github.com/sunriseex/finance-manager/internal/models"
)

func (s *Store) CreateMigratedDeposit(ctx context.Context, account *models.Account, rule *models.InterestRule, transaction *models.Transaction) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migrated deposit transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := insertAccount(ctx, tx, account); err != nil {
		return fmt.Errorf("create migrated account: %w", err)
	}
	if err := insertInterestRule(ctx, tx, rule); err != nil {
		return fmt.Errorf("create migrated interest rule: %w", err)
	}
	if err := insertTransaction(ctx, tx, transaction); err != nil {
		return fmt.Errorf("create migrated initial balance: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrated deposit transaction: %w", err)
	}
	return nil
}
