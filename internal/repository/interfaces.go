package repository

import (
	"context"

	"github.com/sunriseex/finance-manager/internal/models"
)

type AccountRepository interface {
	Create(ctx context.Context, account *models.Account) error
	GetByID(ctx context.Context, id string) (*models.Account, error)
	GetByLegacyID(ctx context.Context, legacyID string) (*models.Account, error)
	List(ctx context.Context) ([]models.Account, error)
	Update(ctx context.Context, account *models.Account) error
	Archive(ctx context.Context, id string) error
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	CreateMany(ctx context.Context, transactions []models.Transaction) error
	GetByID(ctx context.Context, id string) (*models.Transaction, error)
	List(ctx context.Context) ([]models.Transaction, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.Transaction, error)
	Delete(ctx context.Context, id string) error
}

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	List(ctx context.Context) ([]models.Category, error)
}

type InterestRuleRepository interface {
	Create(ctx context.Context, rule *models.InterestRule) error
	GetByID(ctx context.Context, id string) (*models.InterestRule, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.InterestRule, error)
	Update(ctx context.Context, rule *models.InterestRule) error
}

type InterestAccrualRepository interface {
	Create(ctx context.Context, accrual *models.InterestAccrual) error
	GetByAccountDateRule(ctx context.Context, accountID string, accrualDate string, ruleID string) (*models.InterestAccrual, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.InterestAccrual, error)
}
