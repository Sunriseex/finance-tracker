package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestBearerTokenAuthAllowsValidToken(t *testing.T) {
	handler := BearerTokenAuth("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
	req.Header.Set("Authorization", "Bearer secret-token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestBearerTokenAuthRejectsMissingToken(t *testing.T) {
	handler := BearerTokenAuth("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBearerTokenAuthRejectsInvalidToken(t *testing.T) {
	handler := BearerTokenAuth("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
	req.Header.Set("Authorization", "Bearer wrong-token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBearerTokenAuthRejectsEmptyConfiguredToken(t *testing.T) {
	handler := BearerTokenAuth("")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/accounts", http.NoBody)
	req.Header.Set("Authorization", "Bearer anything")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestJWTAuthStoresUserIDInContext(t *testing.T) {
	tokens, err := auth.NewTokenService("01234567890123456789012345678901", "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	pair, err := tokens.IssuePair("user-1", "user@example.com", time.Now())
	if err != nil {
		t.Fatalf("issue pair: %v", err)
	}
	refresh := &testRefreshRepo{byID: map[string]*models.RefreshToken{
		pair.RefreshTokenID: {
			ID:        pair.RefreshTokenID,
			UserID:    "user-1",
			TokenHash: pair.RefreshTokenHash,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		},
	}}
	handler := JWTAuth(tokens, refresh)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user id in context")
		}
		if userID != "user-1" {
			t.Fatalf("user id = %q, want user-1", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/settings/profile", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestUserIDFromContextRejectsMissingClaims(t *testing.T) {
	if userID, ok := UserIDFromContext(t.Context()); ok || userID != "" {
		t.Fatalf("user id = %q, ok = %v; want empty false", userID, ok)
	}
}

type testRefreshRepo struct {
	byID map[string]*models.RefreshToken
}

func (r *testRefreshRepo) Create(context.Context, *models.RefreshToken) error {
	return nil
}

func (r *testRefreshRepo) GetByID(_ context.Context, id string) (*models.RefreshToken, error) {
	token, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return token, nil
}

func (r *testRefreshRepo) GetByHash(context.Context, string) (*models.RefreshToken, error) {
	return nil, repository.ErrNotFound
}

func (r *testRefreshRepo) ListByUser(context.Context, string) ([]models.RefreshToken, error) {
	return nil, nil
}

func (r *testRefreshRepo) Revoke(context.Context, string, time.Time) error {
	return nil
}

func (r *testRefreshRepo) RevokeByUserSession(context.Context, string, string, time.Time) error {
	return nil
}

func (r *testRefreshRepo) RevokeByUser(context.Context, string, time.Time) error {
	return nil
}
