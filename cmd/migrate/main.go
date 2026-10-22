package main

import (
	"context"
	"fmt"
	"os"

	"url-shortener/internal/config"
	"url-shortener/internal/database"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db, "migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("migrations applied")
}
