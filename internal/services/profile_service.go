package services

import (
	"context"
	"fmt"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type ProfileService struct {
	users repository.UserRepository
	now   func() time.Time
}

func NewProfileService(users repository.UserRepository) *ProfileService {
	return &ProfileService{users: users, now: time.Now}
}

type UpdateProfileRequest struct {
	UserID          string
	PrimaryCurrency string
}

func (s *ProfileService) Get(ctx context.Context, userID string) (*models.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if s.users == nil {
		return nil, fmt.Errorf("profile service is not configured")
	}
	if userID == "" {
		return nil, validationError("user id is required")
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *ProfileService) Update(ctx context.Context, req UpdateProfileRequest) (*models.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	if s.users == nil {
		return nil, fmt.Errorf("profile service is not configured")
	}
	if req.UserID == "" {
		return nil, validationError("user id is required")
	}
	primaryCurrency := normalizePrimaryCurrency(req.PrimaryCurrency)
	if err := validateCurrency(primaryCurrency); err != nil {
		return nil, err
	}
	if err := s.users.UpdatePrimaryCurrency(ctx, req.UserID, primaryCurrency, s.now()); err != nil {
		return nil, fmt.Errorf("update primary currency: %w", err)
	}
	user, err := s.users.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get updated user: %w", err)
	}
	return user, nil
}
