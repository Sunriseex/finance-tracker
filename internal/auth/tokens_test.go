package auth

import (
	"testing"
	"time"
)

const testSecret = "01234567890123456789012345678901"

func TestTokenServiceIssueAndValidateAccess(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	service, err := NewTokenService(testSecret, "capitalflow", 15*time.Minute, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	pair, err := service.IssuePair("user-1", "user@example.com", now)
	if err != nil {
		t.Fatalf("issue pair: %v", err)
	}
	if pair.RefreshToken == "" || pair.RefreshTokenHash == "" || pair.RefreshToken == pair.RefreshTokenHash {
		t.Fatal("refresh token and hash were not generated correctly")
	}

	claims, err := service.ValidateAccess(pair.AccessToken, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("validate access: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "user@example.com" || claims.SessionID != pair.RefreshTokenID {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestTokenServiceRejectsExpiredAccessToken(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	service, err := NewTokenService(testSecret, "capitalflow", time.Minute, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	pair, err := service.IssuePair("user-1", "user@example.com", now)
	if err != nil {
		t.Fatalf("issue pair: %v", err)
	}

	if _, err := service.ValidateAccess(pair.AccessToken, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestTokenServiceRejectsInvalidSignature(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	service, err := NewTokenService(testSecret, "capitalflow", time.Minute, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	otherService, err := NewTokenService("abcdefghijklmnopqrstuvwxyz123456", "capitalflow", time.Minute, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("new other token service: %v", err)
	}
	pair, err := otherService.IssuePair("user-1", "user@example.com", now)
	if err != nil {
		t.Fatalf("issue pair: %v", err)
	}

	if _, err := service.ValidateAccess(pair.AccessToken, now); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestHashRefreshToken(t *testing.T) {
	first := HashRefreshToken("refresh-token")
	second := HashRefreshToken("refresh-token")
	if first == "" || first != second {
		t.Fatal("refresh token hash must be stable")
	}
	if first == "refresh-token" {
		t.Fatal("refresh token hash must not store raw token")
	}
}
