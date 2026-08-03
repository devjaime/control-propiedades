package config

import "testing"

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setStorageEnv(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}

func TestLoadParsesTypedValues(t *testing.T) {
	setStorageEnv(t)
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DATABASE_MAX_CONNS", "17")
	t.Setenv("SHUTDOWN_TIMEOUT", "4s")
	t.Setenv("WEB_ORIGIN", "https://web.example")
	t.Setenv("PUBLIC_WEB_URL", "https://public.example")

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
	if cfg.Storage.MaxUploadBytes != 50*1024*1024 {
		t.Fatalf("MaxUploadBytes = %d, want %d", cfg.Storage.MaxUploadBytes, 50*1024*1024)
	}
	if cfg.Auth.PublicWebURL != "https://public.example" {
		t.Fatalf("PublicWebURL = %q", cfg.Auth.PublicWebURL)
	}
}

func setStorageEnv(t *testing.T) {
	t.Helper()
	t.Setenv("STORAGE_ACCESS_KEY", "test-access")
	t.Setenv("STORAGE_SECRET_KEY", "test-secret")
	t.Setenv("STORAGE_REGION", "auto")
}
