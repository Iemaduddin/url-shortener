package config

import "os"

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	PublicBaseURL string
}

func Load() Config {
	return Config{
		HTTPAddr:      valueOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:   valueOrDefault("DATABASE_URL", "postgres://postgres:12345@localhost:5432/url_shortener?sslmode=disable"),
		PublicBaseURL: valueOrDefault("PUBLIC_BASE_URL", "http://localhost:8080"),
	}
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
