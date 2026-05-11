package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type TokenService struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	SessionID string `json:"session_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshTokenID   string
	RefreshTokenHash string
	RefreshExpiresAt time.Time
}

func NewTokenService(secret, issuer string, accessTTL, refreshTTL time.Duration) (*TokenService, error) {
	if len(secret) < 32 {
		slog.Warn("token service rejected short JWT secret", "secret_length", len(secret))
		return nil, fmt.Errorf("JWT secret must be at least 32 bytes")
	}
	if issuer == "" {
		slog.Warn("token service rejected empty issuer")
		return nil, fmt.Errorf("issuer is required")
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		slog.Warn("token service rejected invalid TTL", "access_ttl", accessTTL, "refresh_ttl", refreshTTL)
		return nil, fmt.Errorf("token TTLs must be positive")
	}
	slog.Info("token service initialized", "issuer", issuer, "access_ttl", accessTTL, "refresh_ttl", refreshTTL)
	return &TokenService{secret: []byte(secret), issuer: issuer, accessTTL: accessTTL, refreshTTL: refreshTTL}, nil
}

func (s *TokenService) IssuePair(userID, email string, now time.Time) (*TokenPair, error) {
	accessExpiresAt := now.Add(s.accessTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)
	refreshTokenID := uuid.NewString()

	accessClaims := Claims{
		UserID:    userID,
		Email:     email,
		SessionID: refreshTokenID,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.secret)
	if err != nil {
		slog.Error("access token signing failed", "user_id", userID, "error", err)
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := randomToken(32)
	if err != nil {
		slog.Error("refresh token generation failed", "user_id", userID, "error", err)
		return nil, err
	}

	slog.Info("token pair issued", "user_id", userID, "access_expires_at", accessExpiresAt, "refresh_expires_at", refreshExpiresAt)
	return &TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshTokenID:   refreshTokenID,
		RefreshTokenHash: HashRefreshToken(refreshToken),
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *TokenService) ValidateAccess(tokenString string, now time.Time) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return s.secret, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(s.issuer),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		slog.Warn("access token validation failed", "error", err)
		return nil, fmt.Errorf("parse access token: %w", err)
	}
	if !token.Valid {
		slog.Warn("access token validation failed", "reason", "invalid_token")
		return nil, fmt.Errorf("invalid token")
	}
	if claims.TokenType != TokenTypeAccess {
		slog.Warn("access token validation failed", "reason", "invalid_token_type", "token_type", claims.TokenType)
		return nil, fmt.Errorf("invalid token type")
	}
	if claims.SessionID == "" {
		slog.Warn("access token validation failed", "reason", "missing_session_id")
		return nil, fmt.Errorf("missing session id")
	}
	slog.Debug("access token validated", "user_id", claims.UserID, "expires_at", claims.ExpiresAt)
	return claims, nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken(size int) (string, error) {
	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		slog.Error("secure random token generation failed", "size", size, "error", err)
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
