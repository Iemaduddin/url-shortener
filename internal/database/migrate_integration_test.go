package database

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigratePostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	if err := Migrate(ctx, db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate PostgreSQL: %v", err)
	}
	var migrationCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("read migration tracking: %v", err)
	}
	if migrationCount < 6 {
		t.Fatalf("migration count = %d, want at least 6", migrationCount)
	}
	var auditTable bool
	if err := db.QueryRow(ctx, `SELECT to_regclass('public.audit_logs') IS NOT NULL`).Scan(&auditTable); err != nil {
		t.Fatalf("check audit table: %v", err)
	}
	if !auditTable {
		t.Fatal("audit_logs table was not created")
	}

	code := "i" + strconv.FormatInt(time.Now().UnixNano(), 36)
	var urlID string
	if err := db.QueryRow(ctx, `
		INSERT INTO short_urls (code, original_url) VALUES ($1, 'https://example.com') RETURNING id
	`, code).Scan(&urlID); err != nil {
		t.Fatalf("insert click summary fixture: %v", err)
	}
	defer db.Exec(context.Background(), `DELETE FROM short_urls WHERE id = $1::uuid`, urlID)
	if _, err := db.Exec(ctx, `
		INSERT INTO click_events (short_url_id) VALUES ($1::uuid), ($1::uuid)
	`, urlID); err != nil {
		t.Fatalf("insert click summary events: %v", err)
	}
	var totalClicks int64
	var hasLastClicked bool
	if err := db.QueryRow(ctx, `
		SELECT total_clicks, last_clicked_at IS NOT NULL FROM short_urls WHERE id = $1::uuid
	`, urlID).Scan(&totalClicks, &hasLastClicked); err != nil {
		t.Fatalf("read click summary: %v", err)
	}
	if totalClicks != 2 || !hasLastClicked {
		t.Fatalf("click summary = (%d, %t), want (2, true)", totalClicks, hasLastClicked)
	}
}
