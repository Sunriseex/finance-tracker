package postgres

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
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

func (r *TransactionRepository) CreateForUser(ctx context.Context, userID string, transaction *models.Transaction) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create user transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var accountID string
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM accounts
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, transaction.AccountID, userID).Scan(&accountID); err != nil {
		return fmt.Errorf("lock transaction account: %w", mapNotFound(err))
	}

	if err := insertTransaction(ctx, tx, transaction); err != nil {
		return fmt.Errorf("create user transaction: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create user transaction: %w", err)
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

func (r *TransactionRepository) CreateTransfer(ctx context.Context, userID, fromAccountID, toAccountID, fromCurrency, toCurrency string, transactions []models.Transaction) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create transfer: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	accountIDs := []string{fromAccountID, toAccountID}
	slices.Sort(accountIDs)
	rows, err := tx.Query(ctx, `
		SELECT id, currency
		FROM accounts
		WHERE id IN ($1, $2) AND owner_user_id = $3
		ORDER BY id
		FOR UPDATE
	`, accountIDs[0], accountIDs[1], userID)
	if err != nil {
		return fmt.Errorf("lock transfer accounts: %w", err)
	}
	lockedCurrencies := map[string]string{}
	for rows.Next() {
		var id string
		var currency string
		if err := rows.Scan(&id, &currency); err != nil {
			rows.Close()
			return fmt.Errorf("scan locked transfer account: %w", err)
		}
		lockedCurrencies[id] = currency
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("lock transfer accounts rows: %w", err)
	}
	rows.Close()
	if len(lockedCurrencies) != 2 {
		return fmt.Errorf("lock transfer accounts: %w", repository.ErrNotFound)
	}
	if strings.TrimSpace(fromCurrency) != "" && lockedCurrencies[fromAccountID] != strings.TrimSpace(fromCurrency) {
		return fmt.Errorf("lock transfer accounts: %w", repository.ErrConflict)
	}
	if strings.TrimSpace(toCurrency) != "" && lockedCurrencies[toAccountID] != strings.TrimSpace(toCurrency) {
		return fmt.Errorf("lock transfer accounts: %w", repository.ErrConflict)
	}

	for i := range transactions {
		if err := insertTransaction(ctx, tx, &transactions[i]); err != nil {
			return fmt.Errorf("create transfer transaction %s: %w", transactions[i].ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create transfer: %w", err)
	}
	return nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*models.Transaction, error) {
	transaction, err := scanTransaction(r.pool.QueryRow(ctx, transactionSelectSQL+` WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", mapNotFound(err))
	}
	return transaction, nil
}

func (r *TransactionRepository) GetByIDForUser(ctx context.Context, id, userID string) (*models.Transaction, error) {
	transaction, err := scanTransaction(r.pool.QueryRow(ctx, transactionSelectSQL+`
		WHERE t.id = $1 AND EXISTS (
			SELECT 1 FROM accounts a WHERE a.id = t.account_id AND a.owner_user_id = $2
		)
	`, id, userID))
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", mapNotFound(err))
	}
	return transaction, nil
}

func (r *TransactionRepository) List(ctx context.Context) ([]models.Transaction, error) {
	return r.list(ctx, transactionSelectSQL+` ORDER BY occurred_at, created_at`)
}

func (r *TransactionRepository) ListByUser(ctx context.Context, userID string) ([]models.Transaction, error) {
	return r.list(ctx, transactionSelectSQL+`
		WHERE EXISTS (
			SELECT 1 FROM accounts a WHERE a.id = t.account_id AND a.owner_user_id = $1
		)
		ORDER BY occurred_at, created_at
	`, userID)
}

func (r *TransactionRepository) ListByAccount(ctx context.Context, accountID string) ([]models.Transaction, error) {
	return r.list(ctx, transactionSelectSQL+`
		WHERE t.account_id = $1
		ORDER BY occurred_at, created_at
	`, accountID)
}

func (r *TransactionRepository) ListByAccountForUser(ctx context.Context, accountID, userID string) ([]models.Transaction, error) {
	return r.list(ctx, transactionSelectSQL+`
		WHERE t.account_id = $1
			AND EXISTS (
				SELECT 1 FROM accounts a WHERE a.id = t.account_id AND a.owner_user_id = $2
			)
		ORDER BY occurred_at, created_at
	`, accountID, userID)
}

func (r *TransactionRepository) GetBalanceByAccountForUser(ctx context.Context, accountID, userID string) (balanceMinor, transactionCount int64, err error) {
	var balance int64
	var count int64
	if err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE
				WHEN t.type IN ('initial_balance', 'income', 'transfer_in', 'interest_income', 'adjustment') THEN t.amount_minor
				WHEN t.type IN ('expense', 'transfer_out') THEN -t.amount_minor
				ELSE 0
			END), 0),
			COUNT(t.id)
		FROM transactions t
		WHERE t.account_id = $1
			AND EXISTS (
				SELECT 1 FROM accounts a WHERE a.id = t.account_id AND a.owner_user_id = $2
			)
	`, accountID, userID).Scan(&balance, &count); err != nil {
		return 0, 0, fmt.Errorf("get account balance: %w", err)
	}
	return balance, count, nil
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

func (r *TransactionRepository) DeleteForUser(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM transactions t
		USING accounts a
		WHERE t.id = $1 AND t.account_id = a.id AND a.owner_user_id = $2
	`, id, userID)
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

const transactionSelectSQL = `
	SELECT t.id, t.account_id, t.related_account_id, t.type, t.amount_minor, t.category_id, t.description, t.occurred_at, t.created_at
	FROM transactions t
`

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
