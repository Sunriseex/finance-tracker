package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/security"
)

func TestAuthServiceSetupCreatesFirstUserSession(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)

	session, err := service.Setup(t.Context(), AuthRequest{
		Email:           " User@Example.COM ",
		Password:        "correct horse battery staple",
		PrimaryCurrency: "usd",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if session.User.Email != "user@example.com" {
		t.Fatalf("email = %q", session.User.Email)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("expected issued tokens")
	}
	if session.User.PrimaryCurrency != "USD" {
		t.Fatalf("primary currency = %q, want USD", session.User.PrimaryCurrency)
	}
	if len(users.byID) != 1 || len(refresh.byHash) != 1 {
		t.Fatalf("expected persisted user and refresh token")
	}
	if !audit.hasEvent("setup_success") {
		t.Fatal("expected setup success audit event")
	}
}

func TestAuthServiceSetupRejectsInvalidPrimaryCurrency(t *testing.T) {
	service, _, _, audit := newTestAuthService(t)

	_, err := service.Setup(t.Context(), AuthRequest{
		Email:           "user@example.com",
		Password:        "correct horse battery staple",
		PrimaryCurrency: "RU",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !audit.hasEvent("setup_failed") {
		t.Fatal("expected setup failed audit event")
	}
}

func TestAuthServiceSetupRejectsWeakPassword(t *testing.T) {
	service, _, _, audit := newTestAuthService(t)

	_, err := service.Setup(t.Context(), AuthRequest{
		Email:           "user@example.com",
		Password:        "password12345",
		PrimaryCurrency: "RUB",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if err.Error() != "password is too weak" {
		t.Fatalf("error = %q, want weak password error", err.Error())
	}
	if !audit.hasEvent("setup_failed") {
		t.Fatal("expected setup failed audit event")
	}
}

func TestAuthServiceSetupRejectsSecondUser(t *testing.T) {
	service, users, _, _ := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}

	_, err := service.Setup(t.Context(), AuthRequest{
		Email:    "other@example.com",
		Password: "correct horse battery staple",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceSetupRejectsConcurrentCreateConflict(t *testing.T) {
	service, users, _, audit := newTestAuthService(t)
	users.createErr = fmt.Errorf("create user: %w", repository.ErrConflict)

	_, err := service.Setup(t.Context(), AuthRequest{
		Email:           "user@example.com",
		Password:        "correct horse battery staple",
		PrimaryCurrency: "RUB",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if err.Error() != "setup is already complete" {
		t.Fatalf("error = %q", err.Error())
	}
	if !audit.hasEvent("setup_failed") {
		t.Fatal("expected setup failed audit event")
	}
}

func TestAuthServiceLoginRejectsWrongPasswordWithSafeMessage(t *testing.T) {
	service, users, _, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: "secret",
	}

	_, err := service.Login(t.Context(), AuthRequest{
		Email:    "user@example.com",
		Password: "wrong password",
	})
	if err == nil || err.Error() != "invalid email or password" {
		t.Fatalf("unexpected error: %v", err)
	}
	if !audit.hasEvent("login_failed") {
		t.Fatal("expected login failed audit event")
	}
}

func TestAuthServiceRefreshRotatesToken(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}
	oldRaw := "old-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: oldHash,
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	session, err := service.Refresh(t.Context(), oldRaw)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if session.RefreshToken == "" || session.RefreshToken == oldRaw {
		t.Fatal("expected rotated refresh token")
	}
	if refresh.byHash[oldHash].RevokedAt == nil {
		t.Fatal("expected old refresh token to be revoked")
	}
	if len(refresh.byHash) != 2 {
		t.Fatalf("refresh token count = %d, want 2", len(refresh.byHash))
	}
	if !audit.hasEvent("refresh_success") {
		t.Fatal("expected refresh success audit event")
	}
}

func TestAuthServiceRefreshDetectsReuseAndRevokesUserTokens(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}
	revokedAt := service.now().Add(-time.Minute)

	oldRaw := "old-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: oldHash,
		ExpiresAt: service.now().Add(time.Hour),
		RevokedAt: &revokedAt,
		CreatedAt: service.now().Add(-time.Hour),
	}

	activeHash := auth.HashRefreshToken("active-refresh-token")
	refresh.byHash[activeHash] = &models.RefreshToken{
		ID:        "token-2",
		UserID:    "user-1",
		TokenHash: activeHash,
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	_, err := service.Refresh(t.Context(), oldRaw)
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if refresh.byHash[activeHash].RevokedAt == nil {
		t.Fatal("expected active user refresh token to be revoked")
	}
	if !audit.hasEvent("refresh_reuse_detected") {
		t.Fatal("expected refresh reuse audit event")
	}
}

func TestAuthServiceRefreshHandlesConcurrentRevokeRace(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}
	oldRaw := "old-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: oldHash,
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}
	refresh.revokeErr = repository.ErrNotFound

	_, err := service.Refresh(t.Context(), oldRaw)
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !audit.hasEvent("refresh_failed") {
		t.Fatal("expected refresh failure audit event")
	}
}

func TestAuthServiceLogoutRevokesRefreshToken(t *testing.T) {
	service, _, refresh, audit := newTestAuthService(t)
	raw := "refresh-token"
	hash := auth.HashRefreshToken(raw)
	refresh.byHash[hash] = &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: hash,
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	if err := service.Logout(t.Context(), raw); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if refresh.byHash[hash].RevokedAt == nil {
		t.Fatal("expected refresh token to be revoked")
	}
	if !audit.hasEvent("logout") {
		t.Fatal("expected logout audit event")
	}
}

func newTestAuthService(t *testing.T) (*AuthService, *fakeUserRepo, *fakeRefreshRepo, *fakeAuditRepo) {
	t.Helper()

	tokens, err := auth.NewTokenService("01234567890123456789012345678901", "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	users := &fakeUserRepo{byID: map[string]*models.User{}}
	refresh := &fakeRefreshRepo{byHash: map[string]*models.RefreshToken{}}
	audit := &fakeAuditRepo{}
	service := NewAuthService(users, refresh, tokens, audit)
	service.now = func() time.Time {
		return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	service.passwordFunc = func(password string, _ security.PasswordParams) (string, error) {
		return "hash:" + password, nil
	}
	service.verifyFunc = func(password, encodedHash string) (bool, error) {
		return encodedHash == "hash:"+password, nil
	}

	return service, users, refresh, audit
}

type fakeUserRepo struct {
	byID      map[string]*models.User
	createErr error
}

func (r *fakeUserRepo) Create(_ context.Context, user *models.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.byID[user.ID] = user
	return nil
}

func (r *fakeUserRepo) Count(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range r.byID {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeUserRepo) UpdatePrimaryCurrency(_ context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PrimaryCurrency = primaryCurrency
	user.UpdatedAt = updatedAt
	return nil
}

type fakeRefreshRepo struct {
	byHash    map[string]*models.RefreshToken
	revokeErr error
}

func (r *fakeRefreshRepo) Create(_ context.Context, token *models.RefreshToken) error {
	if token.TokenHash == "" {
		return errors.New("token hash is required")
	}
	r.byHash[token.TokenHash] = token
	return nil
}

func (r *fakeRefreshRepo) GetByID(_ context.Context, id string) (*models.RefreshToken, error) {
	for _, token := range r.byHash {
		if token.ID == id {
			return token, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeRefreshRepo) GetByHash(_ context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, ok := r.byHash[tokenHash]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return token, nil
}

func (r *fakeRefreshRepo) Revoke(_ context.Context, id string, revokedAt time.Time) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	for _, token := range r.byHash {
		if token.ID == id {
			token.RevokedAt = &revokedAt
			return nil
		}
	}
	return repository.ErrNotFound
}

func (r *fakeRefreshRepo) RevokeByUser(_ context.Context, userID string, revokedAt time.Time) error {
	for _, token := range r.byHash {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &revokedAt
		}
	}
	return nil
}

type fakeAuditRepo struct {
	events []models.AuthAuditEvent
}

func (r *fakeAuditRepo) Create(_ context.Context, event *models.AuthAuditEvent) error {
	r.events = append(r.events, *event)
	return nil
}

func (r *fakeAuditRepo) hasEvent(eventType string) bool {
	for _, event := range r.events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
