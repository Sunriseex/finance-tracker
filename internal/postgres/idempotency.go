package postgres

import (
	"context"
	"fmt"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type IdempotencyRepository struct {
	pool queryExecer
}

func NewIdempotencyRepository(pool queryExecer) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

func (r *IdempotencyRepository) Get(ctx context.Context, key, userID, method, path string) (*models.IdempotencyRecord, error) {
	var record models.IdempotencyRecord
	if err := r.pool.QueryRow(ctx, `
		SELECT key, user_id, method, path, request_hash, status_code, response_body, created_at, expires_at
		FROM idempotency_keys
		WHERE key = $1 AND user_id = $2 AND method = $3 AND path = $4 AND expires_at > now()
	`, key, userID, method, path).Scan(
		&record.Key,
		&record.UserID,
		&record.Method,
		&record.Path,
		&record.RequestHash,
		&record.StatusCode,
		&record.ResponseBody,
		&record.CreatedAt,
		&record.ExpiresAt,
	); err != nil {
		return nil, fmt.Errorf("get idempotency key: %w", mapNotFound(err))
	}
	return &record, nil
}

func (r *IdempotencyRepository) CreatePending(ctx context.Context, record *models.IdempotencyRecord) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (key, user_id, method, path, request_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (key, user_id, method, path) DO UPDATE
		SET request_hash = EXCLUDED.request_hash,
			status_code = NULL,
			response_body = NULL,
			created_at = EXCLUDED.created_at,
			expires_at = EXCLUDED.expires_at
		WHERE idempotency_keys.expires_at <= now()
	`, record.Key, record.UserID, record.Method, record.Path, record.RequestHash, record.CreatedAt, record.ExpiresAt)
	if err != nil {
		return false, fmt.Errorf("create idempotency key: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *IdempotencyRepository) Complete(ctx context.Context, key, userID, method, path string, statusCode int, responseBody []byte) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status_code = $5, response_body = $6
		WHERE key = $1 AND user_id = $2 AND method = $3 AND path = $4
	`, key, userID, method, path, statusCode, responseBody)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("complete idempotency key: %w", repository.ErrNotFound)
	}
	return nil
}
