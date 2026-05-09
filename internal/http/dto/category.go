package dto

import (
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

type CategoryResponse struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CategoryFromModel(category *models.Category) CategoryResponse {
	return CategoryResponse{
		ID:        category.ID,
		Slug:      category.Slug,
		Name:      category.Name,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}

func CategoriesFromModels(categories []models.Category) []CategoryResponse {
	response := make([]CategoryResponse, 0, len(categories))
	for i := range categories {
		response = append(response, CategoryFromModel(&categories[i]))
	}
	return response
}
