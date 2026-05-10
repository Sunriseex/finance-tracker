package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type Store interface {
	Accounts() repository.AccountRepository
	Transactions() repository.TransactionRepository
	Categories() repository.CategoryRepository
	InterestRules() repository.InterestRuleRepository
	InterestAccruals() repository.InterestAccrualRepository
	Ping(ctx context.Context) error
}

type Handler struct {
	store Store
}

type RouterConfig struct {
	APIAuthToken       string
	CORSAllowedOrigins []string
	RateLimitRequests  int
	RateLimitWindow    time.Duration
}

func NewRouter(store Store, cfg RouterConfig) http.Handler {
	h := &Handler{store: store}
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
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
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}))
	if cfg.RateLimitRequests > 0 && cfg.RateLimitWindow > 0 {
		r.Use(appmiddleware.RateLimitByIP(cfg.RateLimitRequests, cfg.RateLimitWindow))
	}

	r.Get("/health", h.health)
	r.Get("/ready", h.ready)

	r.Route("/api", func(r chi.Router) {
		r.Use(appmiddleware.BearerTokenAuth(cfg.APIAuthToken))

		r.Get("/categories", h.listCategories)
		r.Get("/currency-rates", h.getCurrencyRates)

		r.Get("/accounts", h.listAccounts)
		r.Post("/accounts", h.createAccount)
		r.Get("/accounts/{id}", h.getAccount)
		r.Patch("/accounts/{id}", h.updateAccount)
		r.Post("/accounts/{id}/archive", h.archiveAccount)
		r.Get("/accounts/{id}/balance", h.getAccountBalance)

		r.Get("/transactions", h.listTransactions)
		r.Post("/transactions", h.createTransaction)
		r.Get("/transactions/{id}", h.getTransaction)
		r.Delete("/transactions/{id}", h.deleteTransaction)

		r.Post("/transfers", h.createTransfer)

		r.Get("/accounts/{id}/interest-rules", h.listInterestRules)
		r.Post("/accounts/{id}/interest-rules", h.createInterestRule)
		r.Patch("/interest-rules/{id}", h.updateInterestRule)
		r.Post("/accounts/{id}/accrue-interest", h.accrueInterest)
		r.Post("/accounts/{id}/recalculate-interest", h.recalculateInterest)

		r.Get("/dashboard/summary", h.getDashboardSummary)
		r.Get("/dashboard/net-worth", h.getDashboardNetWorth)
		r.Get("/dashboard/cashflow", h.getDashboardCashflow)
		r.Get("/dashboard/interest-income", h.getDashboardInterestIncome)
	})

	return r
}
