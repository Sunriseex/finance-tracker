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
				writeJSONError(w, http.StatusServiceUnavailable, "authentication_not_configured", "Authentication is not configured", nil)
				return
			}

			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
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
				writeJSONError(w, http.StatusServiceUnavailable, "authentication_not_configured", "Authentication is not configured", nil)
				return
			}

			token, ok := bearerToken(r)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
				return
			}

			claims, err := tokens.ValidateAccess(token, time.Now())
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
				return
			}
			session, err := refreshTokens.GetByID(r.Context(), claims.SessionID)
			if err != nil || session.UserID != claims.UserID || !session.IsActive(time.Now()) {
				if err != nil && !errors.Is(err, repository.ErrNotFound) {
					writeJSONError(w, http.StatusServiceUnavailable, "authentication_not_configured", "Authentication is not configured", nil)
					return
				}
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
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

func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return "", false
	}
	return claims.UserID, true
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
