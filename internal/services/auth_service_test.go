package services

import (
	"cmp"
	"context"
	"errors"
	"expvar"
	"fmt"
	"slices"
	"strconv"
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
	if users.byID["user-1"].FailedLoginAttempts != 1 {
		t.Fatalf("failed attempts = %d, want 1", users.byID["user-1"].FailedLoginAttempts)
	}
}

func TestAuthServiceLoginProgressivelyLocksAccount(t *testing.T) {
	service, users, _, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{
		ID:                  "user-1",
		Email:               "user@example.com",
		PasswordHash:        "hash:correct horse battery staple",
		FailedLoginAttempts: loginLockoutThreshold - 1,
	}

	_, err := service.Login(t.Context(), AuthRequest{
		Email:    "user@example.com",
		Password: "wrong password",
	})
	if err == nil || err.Error() != "invalid email or password" {
		t.Fatalf("unexpected error: %v", err)
	}

	user := users.byID["user-1"]
	if user.FailedLoginAttempts != loginLockoutThreshold {
		t.Fatalf("failed attempts = %d, want %d", user.FailedLoginAttempts, loginLockoutThreshold)
	}
	if user.LockedUntil == nil {
		t.Fatal("expected account lock")
	}
	wantLockedUntil := service.now().Add(5 * time.Minute)
	if !user.LockedUntil.Equal(wantLockedUntil) {
		t.Fatalf("locked_until = %s, want %s", user.LockedUntil, wantLockedUntil)
	}
	if !audit.hasEventReason("login_failed", "account_locked") {
		t.Fatal("expected account_locked audit event")
	}
}

func TestAuthServiceLoginRejectsActiveLockout(t *testing.T) {
	service, users, _, audit := newTestAuthService(t)
	lockedUntil := service.now().Add(time.Minute)
	users.byID["user-1"] = &models.User{
		ID:                  "user-1",
		Email:               "user@example.com",
		PasswordHash:        "hash:correct horse battery staple",
		FailedLoginAttempts: loginLockoutThreshold,
		LockedUntil:         &lockedUntil,
	}

	_, err := service.Login(t.Context(), AuthRequest{
		Email:    "user@example.com",
		Password: "correct horse battery staple",
	})
	if err == nil || err.Error() != "invalid email or password" {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users.byID) != 1 || users.byID["user-1"].FailedLoginAttempts != loginLockoutThreshold {
		t.Fatal("expected lockout to remain unchanged")
	}
	if !audit.hasEventReason("login_failed", "account_locked") {
		t.Fatal("expected account_locked audit event")
	}
}

func TestAuthServiceLoginClearsFailuresOnSuccess(t *testing.T) {
	service, users, _, audit := newTestAuthService(t)
	lockedUntil := service.now().Add(-time.Minute)
	users.byID["user-1"] = &models.User{
		ID:                  "user-1",
		Email:               "user@example.com",
		PasswordHash:        "hash:correct horse battery staple",
		FailedLoginAttempts: 3,
		LockedUntil:         &lockedUntil,
	}

	session, err := service.Login(t.Context(), AuthRequest{
		Email:    "user@example.com",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if session.AccessToken == "" {
		t.Fatal("expected access token")
	}
	user := users.byID["user-1"]
	if user.FailedLoginAttempts != 0 || user.LockedUntil != nil {
		t.Fatalf("expected login failures cleared, got attempts=%d lockedUntil=%v", user.FailedLoginAttempts, user.LockedUntil)
	}
	if !audit.hasEvent("login_success") {
		t.Fatal("expected login success audit event")
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
	if refresh.byHash[oldHash].RevokedReason == nil || *refresh.byHash[oldHash].RevokedReason != refreshRevokedReasonRotated {
		t.Fatalf("revoked reason = %v, want %q", refresh.byHash[oldHash].RevokedReason, refreshRevokedReasonRotated)
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
	revokedReason := refreshRevokedReasonRotated

	oldRaw := "old-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:            "token-1",
		UserID:        "user-1",
		TokenHash:     oldHash,
		ExpiresAt:     service.now().Add(time.Hour),
		RevokedAt:     &revokedAt,
		RevokedReason: &revokedReason,
		CreatedAt:     service.now().Add(-time.Hour),
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
	if refresh.byHash[activeHash].RevokedReason == nil || *refresh.byHash[activeHash].RevokedReason != refreshRevokedReasonReuseDetected {
		t.Fatalf("active token revoked reason = %v, want %q", refresh.byHash[activeHash].RevokedReason, refreshRevokedReasonReuseDetected)
	}
	if refresh.revokeByUserCalls != 1 {
		t.Fatalf("revoke by user calls = %d, want 1", refresh.revokeByUserCalls)
	}
	if refresh.lastRevokeByUserReason != refreshRevokedReasonReuseDetected {
		t.Fatalf("last revoke by user reason = %q, want %q", refresh.lastRevokeByUserReason, refreshRevokedReasonReuseDetected)
	}
	if !audit.hasEvent("refresh_reuse_detected") {
		t.Fatal("expected refresh reuse audit event")
	}
}

func TestAuthServiceRefreshLegacyRevokedTokenRevokesUserTokens(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}

	revokedAt := service.now().Add(-time.Minute)

	oldRaw := "legacy-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:            "token-1",
		UserID:        "user-1",
		TokenHash:     oldHash,
		ExpiresAt:     service.now().Add(time.Hour),
		RevokedAt:     &revokedAt,
		RevokedReason: nil,
		CreatedAt:     service.now().Add(-time.Hour),
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
		t.Fatal("expected active user refresh token to be revoked for legacy revoked token")
	}
	if refresh.revokeByUserCalls != 1 {
		t.Fatalf("revoke by user calls = %d, want 1", refresh.revokeByUserCalls)
	}
	if refresh.lastRevokeByUserReason != refreshRevokedReasonReuseDetected {
		t.Fatalf("reason = %q, want %q", refresh.lastRevokeByUserReason, refreshRevokedReasonReuseDetected)
	}
	if !audit.hasEvent("refresh_reuse_detected") {
		t.Fatal("expected refresh reuse audit event")
	}
}

func TestAuthServiceRefreshDoesNotTreatManualRevocationAsReuse(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}

	revokedAt := service.now().Add(-time.Minute)
	revokedReason := refreshRevokedReasonManual

	oldRaw := "old-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:            "token-1",
		UserID:        "user-1",
		TokenHash:     oldHash,
		ExpiresAt:     service.now().Add(time.Hour),
		RevokedAt:     &revokedAt,
		RevokedReason: &revokedReason,
		CreatedAt:     service.now().Add(-time.Hour),
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
	if refresh.byHash[activeHash].RevokedAt != nil {
		t.Fatal("manual revoked token reuse must not revoke active sessions")
	}
	if refresh.revokeByUserCalls != 0 {
		t.Fatalf("revoke by user calls = %d, want 0", refresh.revokeByUserCalls)
	}
	if audit.hasEvent("refresh_reuse_detected") {
		t.Fatal("manual revoked token must not emit reuse audit event")
	}
}

func TestAuthServiceRefreshExpectedRevokedReasonsDoNotRevokeUserTokens(t *testing.T) {
	reasons := []string{
		refreshRevokedReasonLogout,
		refreshRevokedReasonManual,
		refreshRevokedReasonPasswordChange,
		refreshRevokedReasonReuseDetected,
	}

	for _, reason := range reasons {
		t.Run(reason, func(t *testing.T) {
			service, users, refresh, audit := newTestAuthService(t)
			users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}

			revokedAt := service.now().Add(-time.Minute)
			currentReason := reason

			oldRaw := "old-refresh-token"
			oldHash := auth.HashRefreshToken(oldRaw)
			refresh.byHash[oldHash] = &models.RefreshToken{
				ID:            "token-1",
				UserID:        "user-1",
				TokenHash:     oldHash,
				ExpiresAt:     service.now().Add(time.Hour),
				RevokedAt:     &revokedAt,
				RevokedReason: &currentReason,
				CreatedAt:     service.now().Add(-time.Hour),
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
			if refresh.byHash[activeHash].RevokedAt != nil {
				t.Fatalf("%s revoked token must not revoke active sessions", reason)
			}
			if refresh.revokeByUserCalls != 0 {
				t.Fatalf("revoke by user calls = %d, want 0", refresh.revokeByUserCalls)
			}
			if audit.hasEvent("refresh_reuse_detected") {
				t.Fatalf("%s revoked token must not emit reuse audit event", reason)
			}
		})
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
	if refresh.byHash[hash].RevokedReason == nil || *refresh.byHash[hash].RevokedReason != refreshRevokedReasonLogout {
		t.Fatalf("revoked reason = %v, want %q", refresh.byHash[hash].RevokedReason, refreshRevokedReasonLogout)
	}
	if !audit.hasEvent("logout") {
		t.Fatal("expected logout audit event")
	}
}

func TestAuthServiceChangePasswordRevokesUserSessions(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: "hash:correct horse battery staple",
	}
	refresh.byHash["token-1"] = &models.RefreshToken{
		ID:        "refresh-1",
		UserID:    "user-1",
		TokenHash: "token-1",
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}
	refresh.byHash["token-2"] = &models.RefreshToken{
		ID:        "refresh-2",
		UserID:    "user-1",
		TokenHash: "token-2",
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	err := service.ChangePassword(t.Context(), ChangePasswordRequest{
		UserID:          "user-1",
		CurrentPassword: "correct horse battery staple",
		NewPassword:     "fresh correct horse battery staple 2026!",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if users.byID["user-1"].PasswordHash != "hash:fresh correct horse battery staple 2026!" {
		t.Fatalf("password hash = %q", users.byID["user-1"].PasswordHash)
	}
	for hash, token := range refresh.byHash {
		if token.RevokedAt == nil {
			t.Fatalf("token %s was not revoked", hash)
		}
		if token.RevokedReason == nil || *token.RevokedReason != refreshRevokedReasonPasswordChange {
			t.Fatalf("token %s revoked reason = %v, want %q", hash, token.RevokedReason, refreshRevokedReasonPasswordChange)
		}
	}
	if !audit.hasEvent("change_password_success") {
		t.Fatal("expected password change audit event")
	}
}

func TestAuthServiceChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.byID["user-1"] = &models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: "hash:correct horse battery staple",
	}

	err := service.ChangePassword(t.Context(), ChangePasswordRequest{
		UserID:          "user-1",
		CurrentPassword: "wrong password",
		NewPassword:     "fresh correct horse battery staple 2026!",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if users.byID["user-1"].PasswordHash != "hash:correct horse battery staple" {
		t.Fatal("password hash changed")
	}
	if len(refresh.byHash) != 0 {
		t.Fatal("unexpected refresh revoke")
	}
	if !audit.hasEventReason("change_password_failed", "invalid_current_password") {
		t.Fatal("expected invalid current password audit event")
	}
}

func TestAuthServiceChangePasswordDoesNotRevokeSessionsWhenPasswordUpdateFails(t *testing.T) {
	service, users, refresh, audit := newTestAuthService(t)
	users.updatePasswordErr = errors.New("database failed")
	users.byID["user-1"] = &models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: "hash:correct horse battery staple",
	}
	refresh.byHash["token-1"] = &models.RefreshToken{
		ID:        "refresh-1",
		UserID:    "user-1",
		TokenHash: "token-1",
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	err := service.ChangePassword(t.Context(), ChangePasswordRequest{
		UserID:          "user-1",
		CurrentPassword: "correct horse battery staple",
		NewPassword:     "fresh correct horse battery staple 2026!",
	})

	if err == nil {
		t.Fatal("expected password update error")
	}
	if users.byID["user-1"].PasswordHash != "hash:correct horse battery staple" {
		t.Fatal("password hash changed")
	}
	if refresh.byHash["token-1"].RevokedAt != nil {
		t.Fatal("refresh token was revoked before password update succeeded")
	}
	if refresh.revokeByUserCalls != 0 {
		t.Fatalf("revoke by user calls = %d, want 0", refresh.revokeByUserCalls)
	}
	if !audit.hasEventReason("change_password_failed", "save_failed") {
		t.Fatal("expected save failure audit event")
	}
}

func TestAuthServiceListSessionsMarksCurrentAndActive(t *testing.T) {
	service, _, refresh, audit := newTestAuthService(t)

	expiredAt := service.now().Add(-time.Hour)
	revokedAt := service.now().Add(-time.Minute)

	refresh.byHash["active"] = &models.RefreshToken{
		ID:        "session-1",
		UserID:    "user-1",
		TokenHash: "active",
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}
	refresh.byHash["revoked"] = &models.RefreshToken{
		ID:        "session-2",
		UserID:    "user-1",
		TokenHash: "revoked",
		ExpiresAt: service.now().Add(time.Hour),
		RevokedAt: &revokedAt,
		CreatedAt: service.now().Add(-time.Minute),
	}
	refresh.byHash["expired"] = &models.RefreshToken{
		ID:        "session-3",
		UserID:    "user-1",
		TokenHash: "expired",
		ExpiresAt: expiredAt,
		CreatedAt: service.now().Add(-2 * time.Minute),
	}
	refresh.byHash["other"] = &models.RefreshToken{
		ID:        "session-4",
		UserID:    "user-2",
		TokenHash: "other",
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	sessions, err := service.ListSessions(t.Context(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("sessions count = %d, want 3", len(sessions))
	}
	if !sessions[0].Current || !sessions[0].Active {
		t.Fatalf("first session = %+v, want current active", sessions[0])
	}
	if sessions[1].Active {
		t.Fatalf("revoked session = %+v, want inactive", sessions[1])
	}
	if sessions[2].Active {
		t.Fatalf("expired session = %+v, want inactive", sessions[2])
	}
	if !audit.hasEvent("sessions_listed") {
		t.Fatal("expected sessions listed audit event")
	}
}

func TestAuthServiceRevokeSessionScopesByUser(t *testing.T) {
	service, _, refresh, audit := newTestAuthService(t)
	refresh.byHash["active"] = &models.RefreshToken{
		ID:        "session-1",
		UserID:    "user-1",
		TokenHash: "active",
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	if err := service.RevokeSession(t.Context(), "user-2", "session-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if refresh.byHash["active"].RevokedAt != nil {
		t.Fatal("session was revoked for wrong user")
	}
	if !audit.hasEventReason("session_revoke_failed", "session_not_found") {
		t.Fatal("expected failed session revoke audit event")
	}

	if err := service.RevokeSession(t.Context(), "user-1", "session-1"); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if refresh.byHash["active"].RevokedAt == nil {
		t.Fatal("expected session to be revoked")
	}
	if refresh.byHash["active"].RevokedReason == nil || *refresh.byHash["active"].RevokedReason != refreshRevokedReasonManual {
		t.Fatalf("revoked reason = %v, want %q", refresh.byHash["active"].RevokedReason, refreshRevokedReasonManual)
	}
	if !audit.hasEvent("session_revoked") {
		t.Fatal("expected session revoked audit event")
	}
}

func TestAuthServiceAuditEventRecordsMetric(t *testing.T) {
	service, _, _, _ := newTestAuthService(t)
	key := authEventMetricKey("login_failed", false, "invalid_credentials")
	before := authMetricValue(t, key)

	service.auditEvent(t.Context(), "login_failed", "user@example.com", nil, false, "invalid_credentials")

	after := authMetricValue(t, key)
	if after != before+1 {
		t.Fatalf("metric = %d, want %d", after, before+1)
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
	byID              map[string]*models.User
	createErr         error
	updatePasswordErr error
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

func (r *fakeUserRepo) RecordLoginFailure(_ context.Context, id string, _ int, _ []time.Duration, updatedAt time.Time) (int, *time.Time, error) {
	user, ok := r.byID[id]
	if !ok {
		return 0, nil, repository.ErrNotFound
	}

	attempts := user.FailedLoginAttempts + 1
	var lockedUntil *time.Time
	if attempts >= loginLockoutThreshold {
		delayIndex := min(attempts-loginLockoutThreshold, len(loginLockoutDelays)-1)
		lockedUntil = new(updatedAt.Add(loginLockoutDelays[delayIndex]))
	}

	user.FailedLoginAttempts = attempts
	user.LockedUntil = lockedUntil
	user.UpdatedAt = updatedAt

	return attempts, lockedUntil, nil
}

func (r *fakeUserRepo) ClearLoginFailures(_ context.Context, id string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *fakeUserRepo) UpdatePassword(_ context.Context, id, passwordHash string, updatedAt time.Time) error {
	if r.updatePasswordErr != nil {
		return r.updatePasswordErr
	}
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PasswordHash = passwordHash
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
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
	byHash                   map[string]*models.RefreshToken
	revokeErr                error
	revokeByUserCalls        int
	lastRevokeByUserReason   string
	revokeByUserSessionCalls int
	lastRevokeSessionReason  string
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

func (r *fakeRefreshRepo) ListByUser(_ context.Context, userID string) ([]models.RefreshToken, error) {
	tokens := []models.RefreshToken{}
	for _, token := range r.byHash {
		if token.UserID == userID {
			tokens = append(tokens, *token)
		}
	}
	slices.SortFunc(tokens, func(a, b models.RefreshToken) int {
		return cmp.Compare(b.CreatedAt.UnixNano(), a.CreatedAt.UnixNano())
	})
	return tokens, nil
}

func (r *fakeRefreshRepo) Revoke(_ context.Context, id string, revokedAt time.Time, reason string) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}

	for _, token := range r.byHash {
		if token.ID == id && token.RevokedAt == nil {
			token.RevokedAt = &revokedAt
			token.RevokedReason = &reason
			return nil
		}
	}

	return repository.ErrNotFound
}

func (r *fakeRefreshRepo) RevokeByUserSession(_ context.Context, userID, id string, revokedAt time.Time, reason string) error {
	r.revokeByUserSessionCalls++
	r.lastRevokeSessionReason = reason

	for _, token := range r.byHash {
		if token.UserID == userID && token.ID == id && token.RevokedAt == nil {
			token.RevokedAt = &revokedAt
			token.RevokedReason = &reason
			return nil
		}
	}

	return repository.ErrNotFound
}

func (r *fakeRefreshRepo) RevokeByUser(_ context.Context, userID string, revokedAt time.Time, reason string) error {
	r.revokeByUserCalls++
	r.lastRevokeByUserReason = reason

	if r.revokeErr != nil {
		return r.revokeErr
	}

	for _, token := range r.byHash {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &revokedAt
			token.RevokedReason = &reason
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

func (r *fakeAuditRepo) hasEventReason(eventType, reason string) bool {
	for _, event := range r.events {
		if event.EventType == eventType && event.Reason == reason {
			return true
		}
	}
	return false
}

func authMetricValue(t *testing.T, key string) int64 {
	t.Helper()

	metrics, ok := expvar.Get("capitalflow_auth_events_total").(*expvar.Map)
	if !ok {
		t.Fatal("missing auth metrics map")
	}

	value := metrics.Get(key)
	if value == nil {
		return 0
	}

	count, err := strconv.ParseInt(value.String(), 10, 64)
	if err != nil {
		t.Fatalf("parse metric value %q: %v", value.String(), err)
	}

	return count
}
