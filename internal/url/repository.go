package url

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("short URL not found")

type Repository interface {
	Create(ctx context.Context, code, originalURL string) (ShortURL, error)
	FindByCode(ctx context.Context, code string) (ShortURL, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, code, originalURL string) (ShortURL, error) {
	var item ShortURL
	err := r.db.QueryRow(ctx, `
		INSERT INTO short_urls (code, original_url)
		VALUES ($1, $2)
		RETURNING id, code, original_url, created_at
	`, code, originalURL).Scan(&item.ID, &item.Code, &item.OriginalURL, &item.CreatedAt)
	return item, err
}

func (r *postgresRepository) FindByCode(ctx context.Context, code string) (ShortURL, error) {
	var item ShortURL
	err := r.db.QueryRow(ctx, `
		SELECT id, code, original_url, created_at
		FROM short_urls
		WHERE code = $1 AND status = 'active'
	`, code).Scan(&item.ID, &item.Code, &item.OriginalURL, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShortURL{}, ErrNotFound
	}
	return item, err
}
