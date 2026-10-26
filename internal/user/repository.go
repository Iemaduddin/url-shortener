package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user or session not found")

type Repository interface {
	CreateUser(context.Context, string, string) (User, error)
	EnsureUser(context.Context, string, string, string) (bool, error)
	ListUsers(context.Context, int, int, string) ([]User, int, error)
	CreateManagedUser(context.Context, string, string, string) (User, error)
	UpdateUser(context.Context, string, string, string, string) error
	UpdateStatus(context.Context, string, string) error
	DeleteUser(context.Context, string) error
	FindByEmail(context.Context, string) (User, string, error)
	CreateSession(context.Context, string, string, time.Time) error
	FindSession(context.Context, string) (Session, error)
	DeleteSession(context.Context, string) error
}

type postgresRepository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var item User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role) VALUES ($1, $2, 'user')
		RETURNING id, email, role, status, created_at
	`, email, passwordHash).Scan(&item.ID, &item.Email, &item.Role, &item.Status, &item.CreatedAt)
	return item, err
}

func (r *postgresRepository) ListUsers(ctx context.Context, page, pageSize int, search string) ([]User, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `SELECT id, email, role, status, created_at FROM users WHERE ($1 = '' OR email ILIKE '%' || $1 || '%' OR role = $1) ORDER BY created_at DESC LIMIT $2 OFFSET $3`, search, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()
	users := make([]User, 0)
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.Email, &item.Role, &item.Status, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE ($1 = '' OR email ILIKE '%' || $1 || '%' OR role = $1)`, search).Scan(&total)
	return users, total, err
}

func (r *postgresRepository) CreateManagedUser(ctx context.Context, email, passwordHash, role string) (User, error) {
	var item User
	err := r.db.QueryRow(ctx, `INSERT INTO users (email, password_hash, role, status) VALUES ($1, $2, $3, 'active') RETURNING id, email, role, status, created_at`, email, passwordHash, role).
		Scan(&item.ID, &item.Email, &item.Role, &item.Status, &item.CreatedAt)
	return item, err
}

func (r *postgresRepository) UpdateUser(ctx context.Context, id, email, passwordHash, role string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET email = $2, password_hash = CASE WHEN $3 = '' THEN password_hash ELSE $3 END, role = $4, updated_at = NOW() WHERE id = $1::uuid`, id, email, passwordHash, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id, status string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1::uuid`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if status == "disabled" {
		_, err = r.db.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1::uuid`, id)
	}
	return err
}

func (r *postgresRepository) DeleteUser(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) EnsureUser(ctx context.Context, email, passwordHash, role string) (bool, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)
		ON CONFLICT (email) DO NOTHING
		RETURNING id
	`, email, passwordHash, role).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (User, string, error) {
	var item User
	var passwordHash string
	err := r.db.QueryRow(ctx, `SELECT id, email, role, status, password_hash FROM users WHERE email = $1`, email).
		Scan(&item.ID, &item.Email, &item.Role, &item.Status, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	return item, passwordHash, err
}

func (r *postgresRepository) CreateSession(ctx context.Context, token, userID string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, hashToken(token), userID, expiresAt)
	return err
}

func (r *postgresRepository) FindSession(ctx context.Context, token string) (Session, error) {
	var session Session
	err := r.db.QueryRow(ctx, `
		SELECT s.token_hash, s.user_id, u.role, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > NOW() AND u.status = 'active'
	`, hashToken(token)).Scan(&session.Token, &session.UserID, &session.Role, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return session, err
}

func (r *postgresRepository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token))
	return err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
