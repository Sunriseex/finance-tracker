package repository

import (
	"context"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

type AccountRepository interface {
	Create(ctx context.Context, account *models.Account) error
	GetByID(ctx context.Context, id string) (*models.Account, error)
	GetByIDForUser(ctx context.Context, id, userID string) (*models.Account, error)
	GetByLegacyID(ctx context.Context, legacyID string) (*models.Account, error)
	List(ctx context.Context) ([]models.Account, error)
	ListByUser(ctx context.Context, userID string) ([]models.Account, error)
	Update(ctx context.Context, account *models.Account) error
	UpdateForUser(ctx context.Context, account *models.Account, userID string) error
	Archive(ctx context.Context, id string) error
	ArchiveForUser(ctx context.Context, id, userID string) error
	ClaimUnowned(ctx context.Context, userID string) error
}

type DepositMigrationRepository interface {
	CreateMigratedDeposit(ctx context.Context, account *models.Account, rule *models.InterestRule, transaction *models.Transaction) error
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	CreateMany(ctx context.Context, transactions []models.Transaction) error
	GetByID(ctx context.Context, id string) (*models.Transaction, error)
	GetByIDForUser(ctx context.Context, id, userID string) (*models.Transaction, error)
	List(ctx context.Context) ([]models.Transaction, error)
	ListByUser(ctx context.Context, userID string) ([]models.Transaction, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.Transaction, error)
	ListByAccountForUser(ctx context.Context, accountID, userID string) ([]models.Transaction, error)
	Delete(ctx context.Context, id string) error
	DeleteForUser(ctx context.Context, id, userID string) error
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
	CreateWithTransaction(ctx context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error
	ReplaceRangeWithTransactions(ctx context.Context, accountID, ruleID string, fromDate, toDate time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error)
	GetByAccountDateRule(ctx context.Context, accountID, accrualDate, ruleID string) (*models.InterestAccrual, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.InterestAccrual, error)
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Count(ctx context.Context) (int64, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	UpdatePrimaryCurrency(ctx context.Context, id, primaryCurrency string, updatedAt time.Time) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByID(ctx context.Context, id string) (*models.RefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, id string, revokedAt time.Time) error
	RevokeByUser(ctx context.Context, userID string, revokedAt time.Time) error
}

type AuthAuditRepository interface {
	Create(ctx context.Context, event *models.AuthAuditEvent) error
}
