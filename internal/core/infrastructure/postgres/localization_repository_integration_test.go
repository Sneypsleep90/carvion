//go:build integration

package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/database"
	"github.com/Sneypsleep90/carvion/internal/localization"
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
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(value[:])
}
