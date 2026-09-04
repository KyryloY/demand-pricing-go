package app

import "os"

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	MigrationsPath string
}

func LoadConfig() Config {
	return Config{
		HTTPAddr:       envOr("HTTP_ADDR", ":8080"),
		DatabaseURL:    envOr("DATABASE_URL", "postgres://pricing:pricing@localhost:5432/pricing?sslmode=disable"),
		MigrationsPath: envOr("MIGRATIONS_PATH", "./db/migrations"),
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
