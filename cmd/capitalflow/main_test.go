package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestRunTransactionsCreateRejectsTransferTypes(t *testing.T) {
	oldConfig := config.AppConfig
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
	}
	t.Cleanup(func() {
		config.AppConfig = oldConfig
	})

	tests := []struct {
		name            string
		transactionType string
	}{
		{
			name:            "transfer in",
			transactionType: "transfer_in",
		},
		{
			name:            "transfer out",
			transactionType: "transfer_out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runTransactionsCreate(context.Background(), []string{
				"--account", "account-1",
				"--type", tt.transactionType,
				"--amount", "100.00",
			})

			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "transfer transactions") {
				t.Fatalf("error = %q, want transfer rejection", err.Error())
			}
		})
	}
}

func TestResolveOwnerUserIDAllowsImplicitOwnerForSingleUser(t *testing.T) {
	users := &fakeCLIUserRepo{
		byID: map[string]*models.User{
			"user-1": {ID: "user-1", Email: "user@example.com"},
		},
	}

	ownerUserID, err := resolveOwnerUserID(t.Context(), users, "")
	if err != nil {
		t.Fatalf("resolve owner user id: %v", err)
	}
	if ownerUserID != "" {
		t.Fatalf("owner user id = %q, want empty for repository single-user fallback", ownerUserID)
	}
}

func TestResolveOwnerUserIDRequiresOwnerForMultipleUsers(t *testing.T) {
	users := &fakeCLIUserRepo{
		byID: map[string]*models.User{
			"user-1": {ID: "user-1", Email: "one@example.com"},
			"user-2": {ID: "user-2", Email: "two@example.com"},
		},
	}

	_, err := resolveOwnerUserID(t.Context(), users, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "owner-user-id is required when multiple users exist") {
		t.Fatalf("error = %q, want multiple-user owner requirement", err.Error())
	}
}

func TestResolveOwnerUserIDAllowsUnownedBeforeSetup(t *testing.T) {
	users := &fakeCLIUserRepo{byID: map[string]*models.User{}}

	ownerUserID, err := resolveOwnerUserID(t.Context(), users, "")
	if err != nil {
		t.Fatalf("resolve owner user id: %v", err)
	}
	if ownerUserID != "" {
		t.Fatalf("owner user id = %q, want empty", ownerUserID)
	}
}

func TestResolveOwnerUserIDValidatesProvidedOwner(t *testing.T) {
	users := &fakeCLIUserRepo{
		byID: map[string]*models.User{
			"user-1": {ID: "user-1", Email: "user@example.com"},
		},
	}

	ownerUserID, err := resolveOwnerUserID(t.Context(), users, " user-1 ")
	if err != nil {
		t.Fatalf("resolve owner user id: %v", err)
	}
	if ownerUserID != "user-1" {
		t.Fatalf("owner user id = %q, want user-1", ownerUserID)
	}
}

type fakeCLIUserRepo struct {
	byID map[string]*models.User
}

func (r *fakeCLIUserRepo) Create(_ context.Context, user *models.User) error {
	r.byID[user.ID] = user
	return nil
}

func (r *fakeCLIUserRepo) Count(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *fakeCLIUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range r.byID {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeCLIUserRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeCLIUserRepo) RecordLoginFailure(_ context.Context, id string, attempts int, lockedUntil *time.Time, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.FailedLoginAttempts = attempts
	user.LockedUntil = lockedUntil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *fakeCLIUserRepo) ClearLoginFailures(_ context.Context, id string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *fakeCLIUserRepo) UpdatePrimaryCurrency(_ context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PrimaryCurrency = primaryCurrency
	user.UpdatedAt = updatedAt
	return nil
}
