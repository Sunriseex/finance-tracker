package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	zxcvbn "github.com/nbutton23/zxcvbn-go"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/security"
)

const loginLockoutThreshold = 5

var loginLockoutDelays = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

type AuthService struct {
	users        repository.UserRepository
	refresh      repository.RefreshTokenRepository
	audit        repository.AuthAuditRepository
	accounts     repository.AccountRepository
	tokens       *auth.TokenService
	passwordFunc func(string, security.PasswordParams) (string, error)
	verifyFunc   func(string, string) (bool, error)
	now          func() time.Time
}

func NewAuthService(users repository.UserRepository, refresh repository.RefreshTokenRepository, tokens *auth.TokenService, audit ...repository.AuthAuditRepository) *AuthService {
	var auditRepo repository.AuthAuditRepository
	if len(audit) > 0 {
		auditRepo = audit[0]
	}

	return &AuthService{
		users:        users,
		refresh:      refresh,
		audit:        auditRepo,
		tokens:       tokens,
		passwordFunc: security.HashPassword,
		verifyFunc:   security.VerifyPassword,
		now:          time.Now,
	}
}

func (s *AuthService) WithAccountRepository(repo repository.AccountRepository) *AuthService {
	s.accounts = repo
	return s
}

type AuthRequest struct {
	Email           string
	Password        string
	PrimaryCurrency string
}

type ChangePasswordRequest struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type AuthSession struct {
	User             *models.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type SessionInfo struct {
	ID        string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	Active    bool
	Current   bool
}

func (s *AuthService) Setup(ctx context.Context, req AuthRequest) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("setup auth: %w", err)
	}
	if s.users == nil || s.refresh == nil || s.tokens == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		s.auditEvent(ctx, "setup_failed", req.Email, nil, false, "setup_complete")
		return nil, validationError("setup is already complete")
	}

	user, err := s.buildUser(req)
	if err != nil {
		s.auditEvent(ctx, "setup_failed", req.Email, nil, false, "validation_error")
		return nil, err
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			s.auditEvent(ctx, "setup_failed", req.Email, nil, false, "setup_complete")
			return nil, validationError("setup is already complete")
		}
		s.auditEvent(ctx, "setup_failed", req.Email, nil, false, "save_failed")
		return nil, fmt.Errorf("save user: %w", err)
	}

	if s.accounts != nil {
		if err := s.accounts.ClaimUnowned(ctx, user.ID); err != nil {
			s.auditEvent(ctx, "setup_failed", req.Email, &user.ID, false, "claim_unowned_accounts_failed")
			return nil, fmt.Errorf("claim unowned accounts: %w", err)
		}
	}

	session, err := s.issueSession(ctx, user)
	if err != nil {
		s.auditEvent(ctx, "setup_failed", req.Email, &user.ID, false, "issue_session_failed")
		return nil, err
	}
	s.auditEvent(ctx, "setup_success", user.Email, &user.ID, true, "")
	return session, nil
}

func (s *AuthService) Login(ctx context.Context, req AuthRequest) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if s.users == nil || s.refresh == nil || s.tokens == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		s.auditEvent(ctx, "login_failed", req.Email, nil, false, "validation_error")
		return nil, err
	}
	if strings.TrimSpace(req.Password) == "" {
		s.auditEvent(ctx, "login_failed", email, nil, false, "invalid_credentials")
		return nil, validationError("invalid email or password")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.auditEvent(ctx, "login_failed", email, nil, false, "invalid_credentials")
			return nil, validationError("invalid email or password")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	now := s.now()
	if user.LockedUntil != nil && now.Before(*user.LockedUntil) {
		s.auditEvent(ctx, "login_failed", email, &user.ID, false, "account_locked")
		return nil, validationError("invalid email or password")
	}

	ok, err := s.verifyFunc(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		attempts := user.FailedLoginAttempts + 1
		lockedUntil := loginLockoutUntil(now, attempts)
		if err := s.users.RecordLoginFailure(ctx, user.ID, attempts, lockedUntil, now); err != nil {
			return nil, fmt.Errorf("record login failure: %w", err)
		}
		reason := "invalid_credentials"
		if lockedUntil != nil {
			reason = "account_locked"
		}
		s.auditEvent(ctx, "login_failed", email, &user.ID, false, reason)
		return nil, validationError("invalid email or password")
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		if err := s.users.ClearLoginFailures(ctx, user.ID, now); err != nil {
			return nil, fmt.Errorf("clear login failures: %w", err)
		}
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		user.UpdatedAt = now
	}

	session, err := s.issueSession(ctx, user)
	if err != nil {
		s.auditEvent(ctx, "login_failed", email, &user.ID, false, "issue_session_failed")
		return nil, err
	}
	s.auditEvent(ctx, "login_success", email, &user.ID, true, "")
	return session, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("refresh session: %w", err)
	}
	if s.users == nil || s.refresh == nil || s.tokens == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		s.auditEvent(ctx, "refresh_failed", "", nil, false, "missing_refresh_token")
		return nil, validationError("refresh token is required")
	}

	now := s.now()
	token, err := s.refresh.GetByHash(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.auditEvent(ctx, "refresh_failed", "", nil, false, "invalid_refresh_token")
			return nil, validationError("invalid refresh token")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if !token.IsActive(now) {
		if token.RevokedAt != nil {
			if err := s.refresh.RevokeByUser(ctx, token.UserID, now); err != nil {
				return nil, fmt.Errorf("revoke refresh token family: %w", err)
			}
			s.auditEvent(ctx, "refresh_reuse_detected", "", &token.UserID, false, "revoked_refresh_token_reused")
			return nil, validationError("invalid refresh token")
		}
		s.auditEvent(ctx, "refresh_failed", "", &token.UserID, false, "inactive_refresh_token")
		return nil, validationError("invalid refresh token")
	}

	if err := s.refresh.Revoke(ctx, token.ID, now); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.auditEvent(ctx, "refresh_failed", "", &token.UserID, false, "refresh_token_already_rotated")
			return nil, validationError("invalid refresh token")
		}
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}

	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	session, err := s.issueSession(ctx, user)
	if err != nil {
		s.auditEvent(ctx, "refresh_failed", user.Email, &user.ID, false, "issue_session_failed")
		return nil, err
	}
	s.auditEvent(ctx, "refresh_success", user.Email, &user.ID, true, "")
	return session, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	if s.refresh == nil {
		return fmt.Errorf("auth service is not configured")
	}

	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		s.auditEvent(ctx, "logout", "", nil, true, "missing_refresh_token")
		return nil
	}

	token, err := s.refresh.GetByHash(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.auditEvent(ctx, "logout", "", nil, true, "unknown_refresh_token")
			return nil
		}
		return fmt.Errorf("get refresh token: %w", err)
	}
	if err := s.refresh.Revoke(ctx, token.ID, s.now()); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	s.auditEvent(ctx, "logout", "", &token.UserID, true, "")
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	if s.users == nil || s.refresh == nil {
		return fmt.Errorf("auth service is not configured")
	}

	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return validationError("user is required")
	}
	if strings.TrimSpace(req.CurrentPassword) == "" {
		s.auditEvent(ctx, "change_password_failed", "", &userID, false, "invalid_current_password")
		return validationError("current password is required")
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		s.auditEvent(ctx, "change_password_failed", "", &userID, false, "validation_error")
		return validationError("new password is required")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	ok, err := s.verifyFunc(req.CurrentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		s.auditEvent(ctx, "change_password_failed", user.Email, &user.ID, false, "invalid_current_password")
		return validationError("invalid current password")
	}
	if req.CurrentPassword == req.NewPassword {
		s.auditEvent(ctx, "change_password_failed", user.Email, &user.ID, false, "password_reuse")
		return validationError("new password must be different")
	}
	if err := validatePasswordPolicy(req.NewPassword, user.Email); err != nil {
		s.auditEvent(ctx, "change_password_failed", user.Email, &user.ID, false, "validation_error")
		return err
	}

	hash, err := s.passwordFunc(req.NewPassword, security.DefaultPasswordParams())
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	now := s.now()
	if err := s.users.UpdatePassword(ctx, user.ID, hash, now); err != nil {
		s.auditEvent(ctx, "change_password_failed", user.Email, &user.ID, false, "save_failed")
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.refresh.RevokeByUser(ctx, user.ID, now); err != nil {
		s.auditEvent(ctx, "change_password_failed", user.Email, &user.ID, false, "revoke_sessions_failed")
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	s.auditEvent(ctx, "change_password_success", user.Email, &user.ID, true, "")
	return nil
}

func (s *AuthService) ListSessions(ctx context.Context, userID, currentRefreshTokenID string) ([]SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	if s.refresh == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, validationError("user is required")
	}

	tokens, err := s.refresh.ListByUser(ctx, userID)
	if err != nil {
		s.auditEvent(ctx, "sessions_list_failed", "", &userID, false, "list_failed")
		return nil, fmt.Errorf("list refresh tokens: %w", err)
	}

	now := s.now()
	sessions := make([]SessionInfo, 0, len(tokens))
	for _, token := range tokens {
		sessions = append(sessions, SessionInfo{
			ID:        token.ID,
			ExpiresAt: token.ExpiresAt,
			RevokedAt: token.RevokedAt,
			CreatedAt: token.CreatedAt,
			Active:    token.IsActive(now),
			Current:   token.ID == currentRefreshTokenID,
		})
	}
	s.auditEvent(ctx, "sessions_listed", "", &userID, true, "")
	return sessions, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if s.refresh == nil {
		return fmt.Errorf("auth service is not configured")
	}

	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" {
		return validationError("user is required")
	}
	if sessionID == "" {
		s.auditEvent(ctx, "session_revoke_failed", "", &userID, false, "validation_error")
		return validationError("session is required")
	}

	if err := s.refresh.RevokeByUserSession(ctx, userID, sessionID, s.now()); err != nil {
		reason := "revoke_failed"
		if errors.Is(err, repository.ErrNotFound) {
			reason = "session_not_found"
		}
		s.auditEvent(ctx, "session_revoke_failed", "", &userID, false, reason)
		return fmt.Errorf("revoke session: %w", err)
	}
	s.auditEvent(ctx, "session_revoked", "", &userID, true, "")
	return nil
}

func (s *AuthService) buildUser(req AuthRequest) (*models.User, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if err := validatePasswordPolicy(req.Password, email); err != nil {
		return nil, err
	}
	primaryCurrency := normalizePrimaryCurrency(req.PrimaryCurrency)
	if err := validateCurrency(primaryCurrency); err != nil {
		return nil, err
	}

	hash, err := s.passwordFunc(req.Password, security.DefaultPasswordParams())
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	now := s.now()

	return &models.User{
		ID:              uuid.NewString(),
		Email:           email,
		PasswordHash:    hash,
		PrimaryCurrency: primaryCurrency,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (s *AuthService) issueSession(ctx context.Context, user *models.User) (*AuthSession, error) {
	now := s.now()
	pair, err := s.tokens.IssuePair(user.ID, user.Email, now)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	refreshToken := &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    user.ID,
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: pair.RefreshExpiresAt,
		CreatedAt: now,
	}
	if err := s.refresh.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &AuthSession{
		User:             user,
		AccessToken:      pair.AccessToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshToken:     pair.RefreshToken,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}, nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", validationError("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", validationError("invalid email")
	}
	return email, nil
}

func normalizePrimaryCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return "RUB"
	}
	return currency
}

func validateCurrency(currency string) error {
	if len(currency) != 3 {
		return validationError("invalid currency: " + currency)
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return validationError("invalid currency: " + currency)
		}
	}
	return nil
}

func validatePasswordPolicy(password, email string) error {
	if len(password) < 12 {
		return validationError("password must be at least 12 characters")
	}

	strength := zxcvbn.PasswordStrength(password, passwordUserInputs(email))
	if strength.Score < 3 {
		return validationError("password is too weak")
	}
	return nil
}

func passwordUserInputs(email string) []string {
	inputs := []string{email}
	local, domain, found := strings.Cut(email, "@")
	if found {
		inputs = append(inputs, local, domain)
	}
	return inputs
}

func loginLockoutUntil(now time.Time, attempts int) *time.Time {
	if attempts < loginLockoutThreshold {
		return nil
	}
	delayIndex := min(attempts-loginLockoutThreshold, len(loginLockoutDelays)-1)
	return new(now.Add(loginLockoutDelays[delayIndex]))
}

func (s *AuthService) auditEvent(ctx context.Context, eventType, email string, userID *string, success bool, reason string) {
	recordAuthEventMetric(eventType, success, reason)

	if s.audit == nil {
		return
	}

	event := &models.AuthAuditEvent{
		ID:        uuid.NewString(),
		UserID:    userID,
		EventType: eventType,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Success:   success,
		Reason:    reason,
		CreatedAt: s.now(),
	}
	if err := s.audit.Create(ctx, event); err != nil {
		slog.Warn("auth audit event was not persisted", "event_type", eventType, "error", err)
	}
}
