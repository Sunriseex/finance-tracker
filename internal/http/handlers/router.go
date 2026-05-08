package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	appmiddleware "github.com/sunriseex/finance-manager/internal/http/middleware"
	"github.com/sunriseex/finance-manager/internal/postgres"
)

type Handler struct {
	store *postgres.Store
}

func NewRouter(store *postgres.Store, apiAuthToken string) http.Handler {
	h := &Handler{store: store}
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(appmiddleware.RequestLogger)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", h.health)
	r.Get("/ready", h.ready)

	r.Route("/api", func(r chi.Router) {
		r.Use(appmiddleware.BearerTokenAuth(apiAuthToken))

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
