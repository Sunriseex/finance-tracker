package postgres

import (
	"context"
	"errors"

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

func (s *Store) Accounts() *AccountRepository {
	return NewAccountRepository(s.pool)
}

func (s *Store) Transactions() *TransactionRepository {
	return NewTransactionRepository(s.pool)
}

func (s *Store) Categories() *CategoryRepository {
	return NewCategoryRepository(s.pool)
}

func (s *Store) InterestRules() *InterestRuleRepository {
	return NewInterestRuleRepository(s.pool)
}

func (s *Store) InterestAccruals() *InterestAccrualRepository {
	return NewInterestAccrualRepository(s.pool)
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrNotFound
	}
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
