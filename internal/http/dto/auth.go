package dto

import "time"

type AuthRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	PrimaryCurrency string `json:"primary_currency"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type AuthSessionInfo struct {
	ID        string     `json:"id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitzero"`
	CreatedAt time.Time  `json:"created_at"`
	Active    bool       `json:"active"`
	Current   bool       `json:"current"`
}

type AuthSessionsResponse struct {
	Sessions []AuthSessionInfo `json:"sessions"`
}

type AuthUser struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	PrimaryCurrency string `json:"primary_currency"`
}

type AuthResponse struct {
	User             AuthUser  `json:"user"`
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type AuthStatusResponse struct {
	SetupRequired bool `json:"setup_required"`
}

type ProfileResponse struct {
	User AuthUser `json:"user"`
}

type UpdateProfileRequest struct {
	PrimaryCurrency string `json:"primary_currency"`
}
