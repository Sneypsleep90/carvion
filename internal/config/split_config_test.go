package config

import "testing"

func TestLoadCoreConfigDoesNotRequireHTTPOrDatabaseContainerCredentials(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("DATABASE_URL", "postgres://app:secret@localhost:5432/carvion?sslmode=disable")
	t.Setenv("ADAPTERS_DIR", "./adapters")
	t.Setenv("HTTP_PORT", "invalid-for-core")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.DatabaseURL != "postgres://app:secret@localhost:5432/carvion?sslmode=disable" || got.AdaptersDir != "./adapters" {
		t.Fatalf("Load() = %+v, want core settings without transport/container config", got)
	}
}

func TestLoadHTTPConfigUsesDefaults(t *testing.T) {
	t.Setenv("HTTP_HOST", "")
	t.Setenv("HTTP_PORT", "")

	got, err := LoadHTTP()
	if err != nil {
		t.Fatalf("LoadHTTP() error = %v", err)
	}
	if got.Host != "127.0.0.1" || got.Port != 8080 {
		t.Fatalf("LoadHTTP() = %+v, want loopback:8080", got)
	}
}

func TestLoadHTTPConfigRejectsInvalidPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "70000")
	if _, err := LoadHTTP(); err == nil {
		t.Fatal("LoadHTTP() error = nil, want invalid port error")
	}
}
