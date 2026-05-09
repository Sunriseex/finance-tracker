package handlers

import (
	"net/http"

	"github.com/sunriseex/finance-manager/internal/http/dto"
)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.store.Categories().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.CategoriesFromModels(categories))
}
