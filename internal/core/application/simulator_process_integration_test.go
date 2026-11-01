//go:build integration

package application

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/core/infrastructure/adapterbridge"
	"github.com/Sneypsleep90/carvion/internal/core/infrastructure/postgres"
	"github.com/Sneypsleep90/carvion/internal/database"
	"github.com/Sneypsleep90/carvion/internal/pluginmanager"
)

func TestSimulatorScenarioRunsThroughDemoJSONRPCProcess(t *testing.T) {
	manager := discoverBuiltDemoAdapter(t)
	core, err := NewServiceWithLocalization(adapterbridge.New(manager), &catalogRepositoryStub{}, &localizationRepositoryFake{})
	if err != nil {
		t.Fatalf("create Core service: %v", err)
	}
	result, err := core.RunSimulatorScenario(context.Background(), "demo-android-platform-a")
	if err != nil {
		t.Fatalf("run simulator over JSON-RPC: %v", err)
	}
	if result.Status != domain.SimulatorRunCompleted || !result.PackageVerified || result.AdapterID != "adapter-android-demo" {
		t.Fatalf("simulator result = %+v; want verified run through a separate demo adapter process", result)
	}
	if len(manager.List()) != 1 || manager.List()[0].Status != pluginmanager.AdapterStatusAvailable {
		t.Fatalf("registered adapters = %+v; want available demo adapter", manager.List())
	}
}

func TestSimulatorScenarioPersistsLocalizationThroughJSONRPCAndPostgres(t *testing.T) {
	databaseURL := os.Getenv("CARVION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARVION_TEST_DATABASE_URL to run the PostgreSQL simulator integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect dedicated test PostgreSQL: %v", err)
	}
	defer pool.Close()

	manager := discoverBuiltDemoAdapter(t)
	repository := postgres.NewLocalizationRepository(pool)
	core, err := NewServiceWithLocalization(adapterbridge.New(manager), &catalogRepositoryStub{}, repository)
	if err != nil {
		t.Fatalf("create PostgreSQL-backed Core service: %v", err)
	}
	result, err := core.RunSimulatorScenario(ctx, "demo-android-platform-a")
	if err != nil {
		t.Fatalf("run simulator through JSON-RPC and PostgreSQL: %v", err)
	}
	if result.Status != domain.SimulatorRunCompleted || !result.PackageVerified || result.Package.ID == "" {
		t.Fatalf("simulator result = %+v; want completed run and persisted package ID", result)
	}
	project, err := core.GetLocalizationProject(ctx, result.LocalizationProjectID)
	if err != nil {
		t.Fatalf("read persisted simulator localization project: %v", err)
	}
	if len(project.Resources) != 3 || len(project.Translations) != 3 {
		t.Fatalf("persisted project has %d resources and %d translations; want 3 of each", len(project.Resources), len(project.Translations))
	}
	for _, translation := range project.Translations {
		if translation.Status != domain.TranslationStatusApproved || translation.Locale != "ru-RU" {
			t.Errorf("persisted translation = %+v; want approved ru-RU", translation)
		}
	}
	var packageRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM language_packages WHERE id = $1::uuid AND project_id = $2::uuid AND sha256 = $3`, result.Package.ID, result.LocalizationProjectID, result.Package.SHA256).Scan(&packageRows); err != nil {
		t.Fatalf("query persisted simulator language package: %v", err)
	}
	if packageRows != 1 {
		t.Fatalf("persisted language package rows = %d; want exactly 1", packageRows)
	}
}

func discoverBuiltDemoAdapter(t *testing.T) *pluginmanager.Manager {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate integration test source")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../.."))
	pluginDirectory := filepath.Join(t.TempDir(), "adapter-android-demo")
	if err := os.MkdirAll(filepath.Join(pluginDirectory, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestSource := filepath.Join(repositoryRoot, "adapters", "adapter-android-demo", "manifest.json")
	manifestContent, err := os.ReadFile(manifestSource)
	if err != nil {
		t.Fatalf("read demo adapter manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDirectory, "manifest.json"), manifestContent, 0o644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(pluginDirectory, "bin", "adapter-android-demo")
	build := exec.Command("go", "build", "-o", executable, "./cmd/adapter-android-demo")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build demo adapter process: %v\n%s", err, output)
	}
	if err := os.Chmod(executable, 0o755); err != nil {
		t.Fatal(err)
	}

	manager, err := pluginmanager.DiscoverContext(context.Background(), filepath.Dir(pluginDirectory), 3*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("discover demo adapter process: %v", err)
	}
	return manager
}
