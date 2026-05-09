package dto

import (
	"testing"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestCategoriesFromModels(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	categories := []models.Category{
		{
			ID:        "category-1",
			Slug:      "salary",
			Name:      "Salary",
			CreatedAt: now,
			UpdatedAt: now.Add(time.Hour),
		},
	}

	got := CategoriesFromModels(categories)

	if len(got) != 1 {
		t.Fatalf("categories len = %d, want 1", len(got))
	}
	if got[0].ID != categories[0].ID || got[0].Slug != categories[0].Slug || got[0].Name != categories[0].Name {
		t.Fatalf("category response = %+v, want id/slug/name from model", got[0])
	}
	if !got[0].CreatedAt.Equal(categories[0].CreatedAt) || !got[0].UpdatedAt.Equal(categories[0].UpdatedAt) {
		t.Fatalf("category timestamps = %s/%s, want %s/%s", got[0].CreatedAt, got[0].UpdatedAt, categories[0].CreatedAt, categories[0].UpdatedAt)
	}
}
