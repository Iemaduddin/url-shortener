package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/internal/audit"
	"url-shortener/internal/config"
	"url-shortener/internal/database"
	apphttp "url-shortener/internal/http"
	qrservice "url-shortener/internal/qr"
	urlservice "url-shortener/internal/url"
	userservice "url-shortener/internal/user"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if cfg.AutoMigrate {
		if err := database.Migrate(ctx, db, "migrations"); err != nil {
			logger.Error("database migration failed", "error", err)
			os.Exit(1)
		}
		logger.Info("database migrations applied")
	} else {
		logger.Info("database migrations skipped")
	}

	repository := urlservice.NewRepository(db)
	service := urlservice.NewService(repository)
	userRepository := userservice.NewRepository(db)
	userService := userservice.NewService(userRepository)
	if cfg.SeedAccounts {
		seedDefaultAccounts(ctx, userService, cfg, logger)
	}
	handler := apphttp.NewHandlerWithCookieSecurity(service, userService, cfg.PublicBaseURL, logger, audit.NewRepository(db), qrservice.NewService(qrservice.NewRepository(db)), cfg.CookieSecure)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func seedDefaultAccounts(ctx context.Context, service *userservice.Service, cfg config.Config, logger *slog.Logger) {
	accounts := []struct {
		email    string
		password string
		role     string
	}{
		{cfg.DefaultUserEmail, cfg.DefaultUserPassword, userservice.RoleUser},
		{cfg.DefaultAdminEmail, cfg.DefaultAdminPassword, userservice.RoleAdmin},
	}
	for _, account := range accounts {
		created, err := service.EnsureDefaultUser(ctx, account.email, account.password, account.role)
		if err != nil {
			logger.Error("default account seeding failed", "email", account.email, "role", account.role, "error", err)
			os.Exit(1)
		}
		if created {
			logger.Info("default account created", "email", account.email, "role", account.role)
		}
	}
}
