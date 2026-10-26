package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ActorUserID  string
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]string
}

type Log struct {
	ID           string    `json:"id"`
	ActorUserID  *string   `json:"actor_user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *string   `json:"resource_id"`
	Metadata     []byte    `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Record(ctx context.Context, event Event) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, action, resource_type, resource_id, metadata)
		VALUES (NULLIF($1, '')::uuid, $2, $3, NULLIF($4, ''), $5::jsonb)
	`, event.ActorUserID, event.Action, event.ResourceType, event.ResourceID, metadata)
	return err
}

func (r *Repository) List(ctx context.Context, page, pageSize int) ([]Log, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx, `
		SELECT id, actor_user_id::text, action, resource_type, resource_id, metadata, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	logs := make([]Log, 0)
	for rows.Next() {
		var item Log
		if err := rows.Scan(&item.ID, &item.ActorUserID, &item.Action, &item.ResourceType, &item.ResourceID, &item.Metadata, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
