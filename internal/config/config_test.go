package config

import "testing"

func TestLoadUsesCoreDefaultsAndReadsRequiredSettings(t *testing.T) {
	setCoreSettings(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want %q", got.AppEnv, "development")
	}
	if got.AdaptersDir != "adapters" {
		t.Errorf("AdaptersDir = %q, want %q", got.AdaptersDir, "adapters")
	}
	if got.DatabaseURL != "postgres://dbuser:secret@localhost:5432/carvion?sslmode=disable" {
		t.Errorf("DatabaseURL = %q", got.DatabaseURL)
	}
}

func TestLoadReadsAdaptersDirectory(t *testing.T) {
	setCoreSettings(t)
	t.Setenv("ADAPTERS_DIR", "/trusted/adapters")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.AdaptersDir != "/trusted/adapters" {
		t.Errorf("AdaptersDir = %q, want %q", got.AdaptersDir, "/trusted/adapters")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setCoreSettings(t)
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing DATABASE_URL error")
	}
}

func TestLoadRejectsUnsupportedAppEnv(t *testing.T) {
	setCoreSettings(t)
	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want unsupported APP_ENV error")
	}
}

func TestLoadRejectsInvalidDatabaseURLs(t *testing.T) {
	for _, databaseURL := range []string{
		"http://localhost/carvion",
		"postgres:///carvion",
		"postgres://localhost",
		"://bad-url",
	} {
		t.Run(databaseURL, func(t *testing.T) {
			setCoreSettings(t)
			t.Setenv("DATABASE_URL", databaseURL)

			if _, err := Load(); err == nil {
				t.Fatalf("Load() error = nil, want invalid DATABASE_URL %q error", databaseURL)
			}
		})
	}
}

func TestLoadHTTPConfigReadsHostAndPort(t *testing.T) {
	t.Setenv("HTTP_HOST", "0.0.0.0")
	t.Setenv("HTTP_PORT", "9090")

	got, err := LoadHTTP()
	if err != nil {
		t.Fatalf("LoadHTTP() error = %v", err)
	}
	if got.Host != "0.0.0.0" || got.Port != 9090 {
		t.Fatalf("LoadHTTP() = %+v, want 0.0.0.0:9090", got)
	}
}

func TestLoadHTTPConfigRejectsInvalidPorts(t *testing.T) {
	for _, port := range []string{"not-a-number", "0", "65536", "-1"} {
		t.Run(port, func(t *testing.T) {
			t.Setenv("HTTP_PORT", port)
			if _, err := LoadHTTP(); err == nil {
				t.Fatalf("LoadHTTP() error = nil, want invalid port %q error", port)
			}
		})
	}
}

func setCoreSettings(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"APP_ENV":           "",
		"DATABASE_URL":      "postgres://dbuser:secret@localhost:5432/carvion?sslmode=disable",
		"ADAPTERS_DIR":      "",
		"HTTP_HOST":         "",
		"HTTP_PORT":         "",
		"POSTGRES_DB":       "",
		"POSTGRES_USER":     "",
		"POSTGRES_PASSWORD": "",
	} {
		t.Setenv(key, value)
	}
}
