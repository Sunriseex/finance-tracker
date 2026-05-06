package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/repository"
)

type AccountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repos ...repository.AccountRepository) *AccountService {
	var repo repository.AccountRepository
	if len(repos) > 0 {
		repo = repos[0]
	}
	return &AccountService{repo: repo}
}

type CreateAccountRequest struct {
	Name     string
	Bank     string
	Type     models.AccountType
	Currency string
	OpenedAt time.Time
}

func (s *AccountService) Create(ctx context.Context, req *CreateAccountRequest) (*models.Account, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create account: %w", ctx.Err())
	default:
	}
	if req == nil {
		return nil, fmt.Errorf("create account request is required")
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
	currency = strings.ToUpper(currency)
	if !validCurrency(currency) {
		return nil, fmt.Errorf("invalid currency: %s", currency)
	}

	openedAt := req.OpenedAt
	if openedAt.IsZero() {
		openedAt = time.Now()
	}
	now := time.Now()

	account := &models.Account{
		ID:        uuid.NewString(),
		Name:      name,
		Bank:      strings.TrimSpace(req.Bank),
		Type:      req.Type,
		Currency:  currency,
		IsActive:  true,
		OpenedAt:  openedAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if s.repo != nil {
		if err := s.repo.Create(ctx, account); err != nil {
			return nil, fmt.Errorf("save account: %w", err)
		}
	}

	return account, nil
}

func validCurrency(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
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
