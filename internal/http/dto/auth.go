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
