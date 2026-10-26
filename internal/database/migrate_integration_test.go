package database

import (
	"context"
	"os"
	"path/filepath"
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
	if migrationCount < 4 {
		t.Fatalf("migration count = %d, want at least 4", migrationCount)
	}
	var auditTable bool
	if err := db.QueryRow(ctx, `SELECT to_regclass('public.audit_logs') IS NOT NULL`).Scan(&auditTable); err != nil {
		t.Fatalf("check audit table: %v", err)
	}
	if !auditTable {
		t.Fatal("audit_logs table was not created")
	}
}
