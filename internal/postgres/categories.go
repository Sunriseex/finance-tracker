package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/finance-manager/internal/models"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO categories (id, slug, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, category.ID, category.Slug, category.Name, category.CreatedAt, category.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	return r.get(ctx, `SELECT id, slug, name, created_at, updated_at FROM categories WHERE id = $1`, id)
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	return r.get(ctx, `SELECT id, slug, name, created_at, updated_at FROM categories WHERE slug = $1`, slug)
}

func (r *CategoryRepository) List(ctx context.Context) ([]models.Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, name, created_at, updated_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list categories rows: %w", err)
	}
	return categories, nil
}

func (r *CategoryRepository) get(ctx context.Context, query string, args ...any) (*models.Category, error) {
	category, err := scanCategory(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get category: %w", mapNotFound(err))
	}
	return category, nil
}

type categoryScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row categoryScanner) (*models.Category, error) {
	var category models.Category
	if err := row.Scan(&category.ID, &category.Slug, &category.Name, &category.CreatedAt, &category.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan category: %w", mapNotFound(err))
	}
	return &category, nil
}
