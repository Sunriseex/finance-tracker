package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type contextKey string

const userClaimsKey contextKey = "user_claims"

func BearerTokenAuth(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(expectedToken) == "" {
				http.Error(w, "authentication is not configured", http.StatusServiceUnavailable)
				return
			}

			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func JWTAuth(tokens *auth.TokenService, refreshTokens repository.RefreshTokenRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokens == nil || refreshTokens == nil {
				http.Error(w, "authentication is not configured", http.StatusServiceUnavailable)
				return
			}

			token, ok := bearerToken(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := tokens.ValidateAccess(token, time.Now())
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			session, err := refreshTokens.GetByID(r.Context(), claims.SessionID)
			if err != nil || session.UserID != claims.UserID || !session.IsActive(time.Now()) {
				if err != nil && !errors.Is(err, repository.ErrNotFound) {
					http.Error(w, "authentication is not configured", http.StatusServiceUnavailable)
					return
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*auth.Claims)
	return claims, ok
}

func bearerToken(r *http.Request) (string, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	return token, token != ""
}
