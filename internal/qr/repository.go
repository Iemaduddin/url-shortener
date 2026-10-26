package qr

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("qr code not found")

type Config struct {
	Foreground string
	Background string
	FrameText  string
	Logo       []byte
}

type SavedConfig struct {
	Foreground string
	Background string
	FrameText  string
	HasLogo    bool
	UpdatedAt  time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) Get(ctx context.Context, id string) ([]byte, error) {
	var png []byte
	err := r.db.QueryRow(ctx, `SELECT png FROM qr_codes WHERE short_url_id = $1::uuid`, id).Scan(&png)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return png, err
}

func (r *Repository) GetConfig(ctx context.Context, id string) (SavedConfig, error) {
	var saved SavedConfig
	err := r.db.QueryRow(ctx, `
		SELECT foreground, background, frame_text, logo IS NOT NULL, updated_at
		FROM qr_codes
		WHERE short_url_id = $1::uuid
	`, id).Scan(&saved.Foreground, &saved.Background, &saved.FrameText, &saved.HasLogo, &saved.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedConfig{}, ErrNotFound
	}
	return saved, err
}

func (r *Repository) GetLogo(ctx context.Context, id string) ([]byte, error) {
	var logo []byte
	err := r.db.QueryRow(ctx, `SELECT logo FROM qr_codes WHERE short_url_id = $1::uuid`, id).Scan(&logo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return logo, err
}

func (r *Repository) Save(ctx context.Context, id string, png []byte, config Config) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO qr_codes (short_url_id, png, foreground, background, frame_text, logo)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		ON CONFLICT (short_url_id) DO UPDATE SET png = EXCLUDED.png,
			foreground = EXCLUDED.foreground, background = EXCLUDED.background,
			frame_text = EXCLUDED.frame_text, logo = EXCLUDED.logo, updated_at = NOW()
	`, id, png, config.Foreground, config.Background, config.FrameText, config.Logo)
	return err
}
