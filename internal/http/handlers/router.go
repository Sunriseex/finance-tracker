package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/sunriseex/capitalflow/internal/auth"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

type Store interface {
	Accounts() repository.AccountRepository
	Transactions() repository.TransactionRepository
	Categories() repository.CategoryRepository
	InterestRules() repository.InterestRuleRepository
	InterestAccruals() repository.InterestAccrualRepository
	Users() repository.UserRepository
	RefreshTokens() repository.RefreshTokenRepository
	AuthAuditEvents() repository.AuthAuditRepository
	Idempotency() repository.IdempotencyRepository
	Ping(ctx context.Context) error
}

type Handler struct {
	store         Store
	tokens        *auth.TokenService
	accounts      *services.AccountService
	transactions  *services.TransactionService
	interestRules *services.InterestRuleService
}

type RouterConfig struct {
	APIAuthToken              string
	TokenService              *auth.TokenService
	CORSAllowedOrigins        []string
	RateLimitRequests         int
	RateLimitWindow           time.Duration
	AuthRateLimitRequests     int
	AuthRateLimitWindow       time.Duration
	MutationRateLimitRequests int
	MutationRateLimitWindow   time.Duration
}

func NewRouter(store Store, cfg *RouterConfig) http.Handler {
	if cfg == nil {
		cfg = &RouterConfig{}
	}

	var accountRepo repository.AccountRepository
	var transactionRepo repository.TransactionRepository
	var interestRuleRepo repository.InterestRuleRepository
	var interestAccrualRepo repository.InterestAccrualRepository
	if store != nil {
		accountRepo = store.Accounts()
		transactionRepo = store.Transactions()
		interestRuleRepo = store.InterestRules()
		interestAccrualRepo = store.InterestAccruals()
	}

	transactionService := services.NewTransactionService(transactionRepo)
	h := &Handler{
		store:        store,
		tokens:       cfg.TokenService,
		accounts:     services.NewAccountService(accountRepo),
		transactions: transactionService,
		interestRules: services.NewInterestRuleService(
			transactionService,
			services.WithInterestRuleRepository(interestRuleRepo),
			services.WithInterestAccrualRepository(interestAccrualRepo),
		),
	}
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(appmiddleware.RequestLogger)
	r.Use(chimiddleware.Recoverer)
	r.Use(appmiddleware.CORS(&appmiddleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"Authorization", "Content-Type", appmiddleware.IdempotencyKeyHeader},
		AllowCredentials: true,
	}))

	authRateLimit := appmiddleware.RateLimitByIP(
		firstPositive(cfg.AuthRateLimitRequests, cfg.RateLimitRequests),
		firstPositiveDuration(cfg.AuthRateLimitWindow, cfg.RateLimitWindow),
	)

	mutationRateLimit := appmiddleware.RateLimitByIP(
		firstPositive(cfg.MutationRateLimitRequests, cfg.RateLimitRequests),
		firstPositiveDuration(cfg.MutationRateLimitWindow, cfg.RateLimitWindow),
	)

	r.Get("/health", h.health)
	r.Get("/ready", h.ready)
	r.Get("/metrics", h.metrics)
	r.Get("/auth/status", h.authStatus)
	r.With(authRateLimit).Post("/auth/setup", h.authSetup)
	r.With(authRateLimit).Post("/auth/login", h.authLogin)
	r.With(authRateLimit).Post("/auth/refresh", h.authRefresh)
	r.With(authRateLimit).Post("/auth/logout", h.authLogout)

	r.Route("/api/v1", func(r chi.Router) {
		if cfg.TokenService != nil {
			r.Use(appmiddleware.JWTAuth(cfg.TokenService, h.store.RefreshTokens()))
		} else {
			r.Use(appmiddleware.BearerTokenAuth(cfg.APIAuthToken))
		}

		r.With(appmiddleware.MutationOnly(mutationRateLimit), appmiddleware.Idempotency(h.idempotency())).Group(func(r chi.Router) {
			r.Post("/auth/password", h.changePassword)
			r.Delete("/auth/sessions/{id}", h.revokeSession)
			r.Patch("/settings/profile", h.updateProfile)
			r.Post("/accounts", h.createAccount)
			r.Patch("/accounts/{id}", h.updateAccount)
			r.Post("/accounts/{id}/archive", h.archiveAccount)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transactions", h.createTransaction)
			r.Delete("/transactions/{id}", h.deleteTransaction)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transfers", h.createTransfer)
			r.Post("/accounts/{id}/interest-rules", h.createInterestRule)
			r.Patch("/interest-rules/{id}", h.updateInterestRule)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/accounts/{id}/accrue-interest", h.accrueInterest)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/accounts/{id}/recalculate-interest", h.recalculateInterest)
		})

		r.Get("/categories", h.listCategories)
		r.Get("/auth/sessions", h.listSessions)
		r.Get("/currency-rates", h.getCurrencyRates)
		r.Get("/settings/profile", h.getProfile)

		r.Get("/accounts", h.listAccounts)
		r.Get("/accounts/{id}", h.getAccount)
		r.Get("/accounts/{id}/balance", h.getAccountBalance)

		r.Get("/transactions", h.listTransactions)
		r.Get("/transactions/{id}", h.getTransaction)

		r.Get("/accounts/{id}/interest-rules", h.listInterestRules)

		r.Get("/dashboard/summary", h.getDashboardSummary)
		r.Get("/dashboard/net-worth", h.getDashboardNetWorth)
		r.Get("/dashboard/cashflow", h.getDashboardCashflow)
		r.Get("/dashboard/interest-income", h.getDashboardInterestIncome)
	})

	return r
}

func (h *Handler) idempotency() repository.IdempotencyRepository {
	if h.store == nil {
		return nil
	}
	return h.store.Idempotency()
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func firstPositiveDuration(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}
