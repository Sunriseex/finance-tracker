package models

import "time"

type User struct {
	ID                         string     `json:"id"`
	Email                      string     `json:"email"`
	PasswordHash               string     `json:"-"`
	PrimaryCurrency            string     `json:"primary_currency"`
	EmailVerifiedAt            *time.Time `json:"email_verified_at,omitzero"`
	EmailVerificationTokenHash *string    `json:"-"`
	EmailVerificationSentAt    *time.Time `json:"email_verification_sent_at,omitzero"`
	FailedLoginAttempts        int        `json:"-"`
	LockedUntil                *time.Time `json:"-"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitzero"`
	CreatedAt time.Time  `json:"created_at"`
}

func (t *RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

type AuthAuditEvent struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"user_id,omitzero"`
	EventType string    `json:"event_type"`
	Email     string    `json:"email"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}
