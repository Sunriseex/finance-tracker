package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (
			id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, user.ID, user.Email, user.PasswordHash, user.PrimaryCurrency, user.EmailVerifiedAt, user.EmailVerificationTokenHash, user.EmailVerificationSentAt, user.FailedLoginAttempts, user.LockedUntil, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("create user: %w", repository.ErrConflict)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.get(ctx, `
		SELECT id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until,
			created_at, updated_at
		FROM users
		WHERE lower(email) = lower($1)
	`, strings.TrimSpace(email))
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	return r.get(ctx, `
		SELECT id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until,
			created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)
}

func (r *UserRepository) UpdatePrimaryCurrency(ctx context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET primary_currency = $2, updated_at = $3
		WHERE id = $1
	`, id, primaryCurrency, updatedAt)
	if err != nil {
		return fmt.Errorf("update user primary currency: %w", err)
	}
	return nil
}

func (r *UserRepository) RecordLoginFailure(ctx context.Context, id string, attempts int, lockedUntil *time.Time, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET failed_login_attempts = $2, locked_until = $3, updated_at = $4
		WHERE id = $1
	`, id, attempts, lockedUntil, updatedAt)
	if err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	return nil
}

func (r *UserRepository) ClearLoginFailures(ctx context.Context, id string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL, updated_at = $2
		WHERE id = $1
	`, id, updatedAt)
	if err != nil {
		return fmt.Errorf("clear login failures: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, failed_login_attempts = 0, locked_until = NULL, updated_at = $3
		WHERE id = $1
	`, id, passwordHash, updatedAt)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	return nil
}

func (r *UserRepository) get(ctx context.Context, query string, args ...any) (*models.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get user: %w", mapNotFound(err))
	}
	return user, nil
}

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.RevokedAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByID(ctx context.Context, id string) (*models.RefreshToken, error) {
	token, err := scanRefreshToken(r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE id = $1
	`, id))
	if err != nil {
		return nil, fmt.Errorf("get refresh token by id: %w", mapNotFound(err))
	}
	return token, nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, err := scanRefreshToken(r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash))
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", mapNotFound(err))
	}
	return token, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string, revokedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, id, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke refresh token: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeByUser(ctx context.Context, userID string, revokedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(row userScanner) (*models.User, error) {
	var user models.User
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.PrimaryCurrency,
		&user.EmailVerifiedAt,
		&user.EmailVerificationTokenHash,
		&user.EmailVerificationSentAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan user: %w", mapNotFound(err))
	}
	return &user, nil
}

type refreshTokenScanner interface {
	Scan(dest ...any) error
}

func scanRefreshToken(row refreshTokenScanner) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt); err != nil {
		return nil, fmt.Errorf("scan refresh token: %w", mapNotFound(err))
	}
	return &token, nil
}

type AuthAuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuthAuditRepository(pool *pgxpool.Pool) *AuthAuditRepository {
	return &AuthAuditRepository{pool: pool}
}

func (r *AuthAuditRepository) Create(ctx context.Context, event *models.AuthAuditEvent) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO auth_audit_events (id, user_id, event_type, email, success, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.ID, event.UserID, event.EventType, event.Email, event.Success, event.Reason, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create auth audit event: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
