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
	List(ctx context.Context, userID string, admin bool, page, pageSize int, search, status string) ([]ShortURL, int, error)
	FindByID(ctx context.Context, id, userID string, admin bool) (ShortURL, error)
	Disable(ctx context.Context, id, userID string, admin bool) error
	Enable(ctx context.Context, id, userID string, admin bool) error
	RecordClick(ctx context.Context, id, userAgent, referer string) error
}

func (r *postgresRepository) FindByID(ctx context.Context, id, userID string, admin bool) (ShortURL, error) {
	var item ShortURL
	err := r.db.QueryRow(ctx, `
		SELECT s.id, s.code, s.original_url, s.created_at, s.status,
			s.total_clicks, s.last_clicked_at
		FROM short_urls s
		WHERE s.id = $1::uuid AND ($2::boolean OR s.user_id = NULLIF($3, '')::uuid)
	`, id, admin, userID).Scan(
		&item.ID, &item.Code, &item.OriginalURL, &item.CreatedAt,
		&item.Status, &item.TotalClicks, &item.LastClicked,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShortURL{}, ErrNotFound
	}
	return item, err
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

func (r *postgresRepository) List(ctx context.Context, userID string, admin bool, page, pageSize int, search, status string) ([]ShortURL, int, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT s.id, s.code, s.original_url, s.created_at, s.status,
			s.total_clicks, s.last_clicked_at
		FROM short_urls s
		WHERE ($1::boolean OR s.user_id = $2::uuid)
			AND ($3 = '' OR s.code ILIKE '%' || $3 || '%' OR s.original_url ILIKE '%' || $3 || '%')
			AND ($4 = '' OR s.status = $4)
		ORDER BY s.created_at DESC
		LIMIT $5 OFFSET $6`
	rows, err := r.db.Query(ctx, query, admin, userID, search, status, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ShortURL, 0)
	for rows.Next() {
		var item ShortURL
		if err := rows.Scan(&item.ID, &item.Code, &item.OriginalURL, &item.CreatedAt, &item.Status, &item.TotalClicks, &item.LastClicked); err != nil {
			return nil, 0, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM short_urls s
		WHERE ($1::boolean OR s.user_id = $2::uuid)
			AND ($3 = '' OR s.code ILIKE '%' || $3 || '%' OR s.original_url ILIKE '%' || $3 || '%')
			AND ($4 = '' OR s.status = $4)
	`, admin, userID, search, status).Scan(&total)
	return items, total, err
}

func (r *postgresRepository) RecordClick(ctx context.Context, id, userAgent, referer string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO click_events (short_url_id, user_agent, referer)
		VALUES ($1::uuid, $2, $3)
	`, id, userAgent, referer)
	return err
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
