package url

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("short URL not found")

type Repository interface {
	Create(ctx context.Context, code, originalURL, userID string) (ShortURL, error)
	FindByCode(ctx context.Context, code string) (ShortURL, error)
	List(ctx context.Context, userID string, admin bool) ([]ShortURL, error)
	Disable(ctx context.Context, id, userID string, admin bool) error
	Enable(ctx context.Context, id, userID string, admin bool) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, code, originalURL, userID string) (ShortURL, error) {
	var item ShortURL
	err := r.db.QueryRow(ctx, `
		INSERT INTO short_urls (code, original_url, user_id)
		VALUES ($1, $2, NULLIF($3, '')::uuid)
		RETURNING id, code, original_url, created_at
	`, code, originalURL, userID).Scan(&item.ID, &item.Code, &item.OriginalURL, &item.CreatedAt)
	return item, err
}

func (r *postgresRepository) List(ctx context.Context, userID string, admin bool) ([]ShortURL, error) {
	query := `
		SELECT id, code, original_url, created_at, status
		FROM short_urls
		WHERE ($1::boolean OR user_id = $2::uuid)
		ORDER BY created_at DESC
		LIMIT 100`
	rows, err := r.db.Query(ctx, query, admin, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ShortURL, 0)
	for rows.Next() {
		var item ShortURL
		if err := rows.Scan(&item.ID, &item.Code, &item.OriginalURL, &item.CreatedAt, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *postgresRepository) Disable(ctx context.Context, id, userID string, admin bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE short_urls
		SET status = 'disabled', updated_at = NOW()
		WHERE id = $1::uuid AND ($2::boolean OR user_id = $3::uuid)
	`, id, admin, userID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Enable(ctx context.Context, id, userID string, admin bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE short_urls
		SET status = 'active', updated_at = NOW()
		WHERE id = $1::uuid AND ($2::boolean OR user_id = $3::uuid)
	`, id, admin, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
