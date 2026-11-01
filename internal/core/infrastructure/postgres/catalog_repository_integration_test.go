//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/database"
)

func TestCatalogRepositoryReadsFoundationTables(t *testing.T) {
	databaseURL := os.Getenv("CARVION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARVION_TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer pool.Close()

	repository := NewCatalogRepository(pool)
	if _, err := repository.ListManufacturers(ctx); err != nil {
		t.Fatalf("ListManufacturers() error = %v", err)
	}
	if _, err := repository.ListVehicleModels(ctx, ""); err != nil {
		t.Fatalf("ListVehicleModels() error = %v", err)
	}
	if _, err := repository.ListVehicleModels(ctx, "00000000-0000-0000-0000-000000000000"); err != nil {
		t.Fatalf("ListVehicleModels() with manufacturer filter error = %v", err)
	}
	if _, err := repository.ListInfotainmentPlatforms(ctx, ""); err != nil {
		t.Fatalf("ListInfotainmentPlatforms() error = %v", err)
	}
	if _, err := repository.ListInfotainmentPlatforms(ctx, "00000000-0000-0000-0000-000000000000"); err != nil {
		t.Fatalf("ListInfotainmentPlatforms() with manufacturer filter error = %v", err)
	}
}
