package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}

func TestLoadParsesTypedValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DATABASE_MAX_CONNS", "17")
	t.Setenv("SHUTDOWN_TIMEOUT", "4s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.MaxConns != 17 {
		t.Fatalf("MaxConns = %d, want 17", cfg.Database.MaxConns)
	}
	if cfg.ShutdownTimeout.String() != "4s" {
		t.Fatalf("ShutdownTimeout = %s, want 4s", cfg.ShutdownTimeout)
	}
}
