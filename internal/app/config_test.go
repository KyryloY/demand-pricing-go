package app

import "testing"

func TestLoadConfigUsesSafeDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MIGRATIONS_PATH", "")

	config := LoadConfig()
	if config.HTTPAddr != ":8080" {
		t.Fatalf("HTTP address = %q, want :8080", config.HTTPAddr)
	}
	if config.DatabaseURL == "" || config.MigrationsPath == "" {
		t.Fatalf("config defaults are incomplete: %#v", config)
	}
}
