//go:build integration

package postgres

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/application"
	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/core/infrastructure/adapterbridge"
	"github.com/Sneypsleep90/carvion/internal/database"
	"github.com/Sneypsleep90/carvion/internal/localization"
	"github.com/Sneypsleep90/carvion/internal/pluginmanager"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestLocalizationRepositoryPersistsProjectHistoryValidationAndPackage(t *testing.T) {
	databaseURL := os.Getenv("CARVION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARVION_TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer pool.Close()
	repository := NewLocalizationRepository(pool)
	resource := domain.LocalizationResource{Key: "screen.title", Kind: domain.ResourceKindString, SourceText: "Home", SourceSHA256: localization.SHA256([]byte("Home")), Order: 0, Translatable: true, Formatted: true, Present: true}
	rollbackName := "Rollback " + randomSuffix(t)
	_, err = repository.CreateLocalizationProject(ctx, domain.LocalizationProject{
		Name: rollbackName, Locale: "ru-RU", Format: domain.LocalizationFormatJSON,
		Filename: "rollback.json", SourceSHA256: localization.SHA256([]byte("rollback")), Source: []byte("rollback"),
		Resources: []domain.LocalizationResource{resource, resource},
	})
	if err == nil {
		t.Fatal("CreateLocalizationProject() accepted duplicate resource keys")
	}
	var rolledBackCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM localization_projects WHERE name = $1`, rollbackName).Scan(&rolledBackCount); err != nil {
		t.Fatal(err)
	}
	if rolledBackCount != 0 {
		t.Fatalf("failed project import left %d project rows, want transaction rollback", rolledBackCount)
	}
	project, err := repository.CreateLocalizationProject(ctx, domain.LocalizationProject{
		Name: "Integration " + randomSuffix(t), Locale: "ru-RU", Format: domain.LocalizationFormatJSON,
		Filename: "resources.json", SourceSHA256: localization.SHA256([]byte("source")), Source: []byte("source"), Resources: []domain.LocalizationResource{resource},
	})
	if err != nil {
		t.Fatalf("CreateLocalizationProject() error = %v", err)
	}
	translation := domain.LocalizationTranslation{ResourceKey: resource.Key, Locale: project.Locale, TargetText: "Главная", Status: domain.TranslationStatusDraft, UpdatedAt: time.Now().UTC()}
	if _, err := repository.SaveLocalizationTranslation(ctx, project.ID, translation); err != nil {
		t.Fatalf("SaveLocalizationTranslation() error = %v", err)
	}
	translation.Status = domain.TranslationStatusApproved
	translation.ApprovedSourceSHA256 = resource.SourceSHA256
	if _, err := repository.SaveLocalizationTranslation(ctx, project.ID, translation); err != nil {
		t.Fatalf("SaveLocalizationTranslation() approval error = %v", err)
	}
	saved, err := repository.GetLocalizationProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetLocalizationProject() error = %v", err)
	}
	if len(saved.Resources) != 1 || len(saved.Translations) != 1 || saved.Translations[0].Status != domain.TranslationStatusApproved {
		t.Fatalf("project round trip = %+v", saved)
	}
	var historyCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM translation_history WHERE project_id = $1::uuid`, project.ID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 2 {
		t.Fatalf("translation history rows = %d, want 2", historyCount)
	}
	if _, err := pool.Exec(ctx, `UPDATE translation_history SET next_target_text = 'mutated' WHERE project_id = $1::uuid`, project.ID); err == nil {
		t.Fatal("translation history accepted an UPDATE, want append-only enforcement")
	}
	updatedResource := resource
	updatedResource.SourceText = "Home screen"
	updatedResource.SourceSHA256 = localization.SHA256([]byte(updatedResource.SourceText))
	if err := repository.ReplaceLocalizationSource(ctx, project.ID, project.Format, project.Filename, localization.SHA256([]byte("replacement")), []byte("replacement"), []domain.LocalizationResource{updatedResource}); err != nil {
		t.Fatalf("ReplaceLocalizationSource() error = %v", err)
	}
	updated, err := repository.GetLocalizationProject(ctx, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Translations[0].Status != domain.TranslationStatusNeedsReview || updated.Translations[0].ApprovedSourceSHA256 != "" {
		t.Fatalf("source replacement left approval active: %+v", updated.Translations[0])
	}
	var historicalApprovalHash string
	if err := pool.QueryRow(ctx, `SELECT approved_source_sha256 FROM translation_history WHERE project_id = $1::uuid AND next_status = 'needs_review'`, project.ID).Scan(&historicalApprovalHash); err != nil {
		t.Fatalf("read historical approval source hash: %v", err)
	}
	if historicalApprovalHash != resource.SourceSHA256 {
		t.Fatalf("historical approval hash = %q, want %q", historicalApprovalHash, resource.SourceSHA256)
	}
	report := domain.LocalizationValidationReport{Valid: true, Locale: project.Locale, Issues: []domain.LocalizationValidationIssue{}}
	validationID, err := repository.SaveLocalizationValidation(ctx, project.ID, report)
	if err != nil {
		t.Fatalf("SaveLocalizationValidation() error = %v", err)
	}
	artifact := []byte("synthetic package")
	item := domain.LocalizationPackage{Filename: "integration-ru-RU.carvion.zip", SHA256: localization.SHA256(artifact), Size: len(artifact)}
	savedPackage, err := repository.SaveLocalizationPackage(ctx, project.ID, item, validationID, artifact)
	if err != nil {
		t.Fatalf("SaveLocalizationPackage() error = %v", err)
	}
	if savedPackage.ID == "" {
		t.Fatal("SaveLocalizationPackage() returned an empty ID")
	}
	storedArtifact, err := repository.GetLocalizationPackage(ctx, project.ID, savedPackage.SHA256)
	if err != nil {
		t.Fatalf("GetLocalizationPackage() error = %v", err)
	}
	if !bytes.Equal(storedArtifact, artifact) {
		t.Fatal("GetLocalizationPackage() returned bytes different from the stored artifact")
	}
}

func TestGeelyResearchImportPersistsSelectedResourceThroughLocalizationRepository(t *testing.T) {
	databaseURL := os.Getenv("CARVION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set CARVION_TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer pool.Close()

	repository := NewLocalizationRepository(pool)
	manager := discoverGeelyResearchAdapter(t)
	core, err := application.NewServiceWithLocalization(adapterbridge.New(manager), integrationCatalogRepository{}, repository)
	if err != nil {
		t.Fatalf("NewServiceWithLocalization() error = %v", err)
	}
	profileJSON := []byte(`{"schema":"carvion.geely-research/v1","profile_id":"test-only-profile","vehicle":{"manufacturer":"TEST ONLY","model":"TEST ONLY","model_year":2024,"market":"TEST ONLY","trim":"TEST ONLY"},"infotainment":{"platform":"TEST ONLY","os_name":"TEST ONLY","os_version":"TEST ONLY","hardware_identifier":"TEST ONLY","hardware_revision":"TEST ONLY","firmware_version":"TEST ONLY","firmware_build_id":"TEST ONLY"}}`)
	resourceFile := `<resources><string name="synthetic.title">TEST ONLY RESOURCE</string></resources>`
	result, err := core.ImportGeelyLocalizationProject(ctx, domain.GeelyLocalizationImportRequest{
		ProfileJSON: string(profileJSON), Name: "Geely research test " + randomSuffix(t), Locale: "ru-RU",
		Filename: "test-resources.xml", Format: domain.LocalizationFormatAndroidXML, Content: resourceFile,
	})
	if err != nil {
		t.Fatalf("ImportGeelyLocalizationProject() error = %v", err)
	}
	if result.Analysis.Compatibility.Status != domain.StatusUnknown {
		t.Fatalf("compatibility = %+v, want unknown with empty reviewed catalog", result.Analysis.Compatibility)
	}
	saved, err := repository.GetLocalizationProject(ctx, result.Project.ID)
	if err != nil {
		t.Fatalf("GetLocalizationProject() error = %v", err)
	}
	if saved.Filename != "test-resources.xml" || len(saved.Resources) != 1 || saved.Resources[0].Key != "synthetic.title" || saved.Resources[0].SourceText != "TEST ONLY RESOURCE" {
		t.Fatalf("saved project = %+v, want selected XML source persisted", saved)
	}
}

func discoverGeelyResearchAdapter(t *testing.T) *pluginmanager.Manager {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pluginRoot := t.TempDir()
	pluginDir := filepath.Join(pluginRoot, "adapter-geely-research")
	binDir := filepath.Join(pluginDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(binDir, "adapter-geely-research")
	build := exec.Command("go", "build", "-o", executable, "./cmd/adapter-geely-research")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Geely research adapter process: %v\n%s", err, output)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(repositoryRoot, "adapters", "adapter-geely-research", "manifest.json"))
	if err != nil {
		t.Fatalf("read Geely adapter manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	var manifest sdk.AdapterManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil || manifest.ID != "adapter-geely-research" {
		t.Fatalf("decode Geely adapter manifest: manifest=%+v error=%v", manifest, err)
	}
	manager, err := pluginmanager.DiscoverContext(context.Background(), pluginRoot, 3*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("DiscoverContext() error = %v", err)
	}
	if adapters := manager.List(); len(adapters) != 1 || adapters[0].ID != manifest.ID {
		t.Fatalf("registered adapters = %+v", adapters)
	}
	return manager
}

type integrationCatalogRepository struct{}

func (integrationCatalogRepository) ListManufacturers(context.Context) ([]domain.Manufacturer, error) {
	return []domain.Manufacturer{}, nil
}
func (integrationCatalogRepository) ListVehicleModels(context.Context, string) ([]domain.VehicleModel, error) {
	return []domain.VehicleModel{}, nil
}
func (integrationCatalogRepository) ListInfotainmentPlatforms(context.Context, string) ([]domain.InfotainmentPlatform, error) {
	return []domain.InfotainmentPlatform{}, nil
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(value[:])
}
