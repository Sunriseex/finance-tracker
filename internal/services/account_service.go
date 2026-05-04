package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/finance-manager/internal/models"
)

type AccountService struct{}

func NewAccountService() *AccountService {
	return &AccountService{}
}

type CreateAccountRequest struct {
	Name     string
	Bank     string
	Type     models.AccountType
	Currency string
	OpenedAt time.Time
}

func (s *AccountService) Create(ctx context.Context, req CreateAccountRequest) (*models.Account, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create account: %w", ctx.Err())
	default:
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("account name is required")
	}
	if !validAccountType(req.Type) {
		return nil, fmt.Errorf("invalid account type: %s", req.Type)
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "RUB"
	}

	openedAt := req.OpenedAt
	if openedAt.IsZero() {
		openedAt = time.Now()
	}
	now := time.Now()

	return &models.Account{
		ID:        uuid.NewString(),
		Name:      name,
		Bank:      strings.TrimSpace(req.Bank),
		Type:      req.Type,
		Currency:  strings.ToUpper(currency),
		IsActive:  true,
		OpenedAt:  openedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func validAccountType(accountType models.AccountType) bool {
	switch accountType {
	case models.AccountTypeCash,
		models.AccountTypeCard,
		models.AccountTypeSavings,
		models.AccountTypeTermDeposit,
		models.AccountTypeBroker,
		models.AccountTypeOther:
		return true
	default:
		return false
	}
}
