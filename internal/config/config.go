package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	PublicBaseURL        string
	AutoMigrate          bool
	SeedAccounts         bool
	DefaultUserEmail     string
	DefaultUserPassword  string
	DefaultAdminEmail    string
	DefaultAdminPassword string
}

func Load() Config {
	return Config{
		HTTPAddr:             valueOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:          valueOrDefault("DATABASE_URL", "postgres://postgres:12345@localhost:5432/url_shortener?sslmode=disable"),
		PublicBaseURL:        valueOrDefault("PUBLIC_BASE_URL", "http://localhost:8080"),
		AutoMigrate:          boolOrDefault("AUTO_MIGRATE", true),
		SeedAccounts:         boolOrDefault("SEED_DEFAULT_ACCOUNTS", true),
		DefaultUserEmail:     valueOrDefault("DEFAULT_USER_EMAIL", "user@example.com"),
		DefaultUserPassword:  valueOrDefault("DEFAULT_USER_PASSWORD", "password"),
		DefaultAdminEmail:    valueOrDefault("DEFAULT_ADMIN_EMAIL", "admin@example.com"),
		DefaultAdminPassword: valueOrDefault("DEFAULT_ADMIN_PASSWORD", "password"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
