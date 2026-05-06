package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/repository"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	if err := insertTransaction(ctx, r.pool, transaction); err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepository) CreateMany(ctx context.Context, transactions []models.Transaction) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create transactions: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for i := range transactions {
		if err := insertTransaction(ctx, tx, &transactions[i]); err != nil {
			return fmt.Errorf("create transaction %s: %w", transactions[i].ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create transactions: %w", err)
	}
	return nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*models.Transaction, error) {
	transaction, err := scanTransaction(r.pool.QueryRow(ctx, `
		SELECT id, account_id, related_account_id, type, amount_minor, category_id, description, occurred_at, created_at
		FROM transactions
		WHERE id = $1
	`, id))
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", mapNotFound(err))
	}
	return transaction, nil
}

func (r *TransactionRepository) List(ctx context.Context) ([]models.Transaction, error) {
	return r.list(ctx, `
		SELECT id, account_id, related_account_id, type, amount_minor, category_id, description, occurred_at, created_at
		FROM transactions
		ORDER BY occurred_at, created_at
	`)
}

func (r *TransactionRepository) ListByAccount(ctx context.Context, accountID string) ([]models.Transaction, error) {
	return r.list(ctx, `
		SELECT id, account_id, related_account_id, type, amount_minor, category_id, description, occurred_at, created_at
		FROM transactions
		WHERE account_id = $1
		ORDER BY occurred_at, created_at
	`, accountID)
}

func (r *TransactionRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete transaction: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *TransactionRepository) list(ctx context.Context, query string, args ...any) ([]models.Transaction, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transactions rows: %w", err)
	}
	return transactions, nil
}

type transactionScanner interface {
	Scan(dest ...any) error
}

func scanTransaction(row transactionScanner) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := row.Scan(
		&transaction.ID,
		&transaction.AccountID,
		&transaction.RelatedAccountID,
		&transaction.Type,
		&transaction.AmountMinor,
		&transaction.CategoryID,
		&transaction.Description,
		&transaction.OccurredAt,
		&transaction.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan transaction: %w", mapNotFound(err))
	}
	return &transaction, nil
}

func insertTransaction(ctx context.Context, execer sqlExecer, transaction *models.Transaction) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO transactions (id, account_id, related_account_id, type, amount_minor, category_id, description, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, transaction.ID, transaction.AccountID, transaction.RelatedAccountID, transaction.Type, transaction.AmountMinor, transaction.CategoryID, transaction.Description, transaction.OccurredAt, transaction.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}
