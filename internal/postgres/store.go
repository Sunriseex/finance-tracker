package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/finance-manager/internal/repository"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Accounts() repository.AccountRepository {
	return NewAccountRepository(s.pool)
}

func (s *Store) Transactions() repository.TransactionRepository {
	return NewTransactionRepository(s.pool)
}

func (s *Store) Categories() repository.CategoryRepository {
	return NewCategoryRepository(s.pool)
}

func (s *Store) InterestRules() repository.InterestRuleRepository {
	return NewInterestRuleRepository(s.pool)
}

func (s *Store) InterestAccruals() repository.InterestAccrualRepository {
	return NewInterestAccrualRepository(s.pool)
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrNotFound
	}
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}
