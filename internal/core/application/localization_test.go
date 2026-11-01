package application

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestLocalizationServiceImportsAndPersistsParsedResources(t *testing.T) {
	repository := &localizationRepositoryFake{}
	service, err := NewLocalizationService(repository)
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.ImportProject(context.Background(), ImportLocalizationRequest{
		Name: "Synthetic UI", Locale: "ru-RU", Format: domain.LocalizationFormatAndroidXML,
		Filename: "strings.xml", Content: []byte(`<resources><string name="title">Home</string></resources>`),
	})
	if err != nil {
		t.Fatalf("ImportProject() error = %v", err)
	}
	if project.ID != "project-1" || len(project.Resources) != 1 || project.Resources[0].Key != "title" || project.SourceSHA256 == "" {
		t.Fatalf("imported project = %+v", project)
	}
	_, err = service.ImportProject(context.Background(), ImportLocalizationRequest{Name: "bad", Locale: "ru-RU", Format: domain.LocalizationFormatAndroidXML, Filename: "bad.xml", Content: []byte("<resources>")})
	if err == nil {
		t.Fatal("ImportProject() accepted malformed XML")
	}
}

func TestLocalizationServiceMarksEditedApprovedTranslationForReview(t *testing.T) {
	service, repository, projectID := newLocalizationServiceWithProject(t)
	translation, err := service.UpdateTranslation(context.Background(), projectID, "title", "Главная")
	if err != nil || translation.Status != domain.TranslationStatusDraft {
		t.Fatalf("UpdateTranslation() = %+v, %v; want draft", translation, err)
	}
	approved, err := service.ApproveTranslation(context.Background(), projectID, "title")
	if err != nil || approved.Status != domain.TranslationStatusApproved || approved.ApprovedSourceSHA256 != repository.project.Resources[0].SourceSHA256 {
		t.Fatalf("ApproveTranslation() = %+v, %v", approved, err)
	}
	revised, err := service.UpdateTranslation(context.Background(), projectID, "title", "Стартовая")
	if err != nil || revised.Status != domain.TranslationStatusNeedsReview {
		t.Fatalf("editing an approved translation = %+v, %v; want needs_review", revised, err)
	}
	report, err := service.ValidateProject(context.Background(), projectID)
	if err != nil || report.Valid || len(report.Issues) == 0 {
		t.Fatalf("ValidateProject() = %+v, %v; want stale approval issue", report, err)
	}
	if repository.validationSaves != 1 {
		t.Fatalf("validation runs saved = %d, want 1", repository.validationSaves)
	}
}

func TestLocalizationServiceReimportMarksApprovedTranslationForReview(t *testing.T) {
	service, _, projectID := newLocalizationServiceWithProject(t)
	if _, err := service.UpdateTranslation(context.Background(), projectID, "title", "Главная"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApproveTranslation(context.Background(), projectID, "title"); err != nil {
		t.Fatal(err)
	}
	updated, err := service.ReplaceSource(context.Background(), projectID, "strings-v2.xml", []byte(`<resources><string name="title">Home screen</string></resources>`))
	if err != nil {
		t.Fatalf("ReplaceSource() error = %v", err)
	}
	if updated.Resources[0].SourceText != "Home screen" || updated.Translations[0].Status != domain.TranslationStatusNeedsReview || updated.Translations[0].ApprovedSourceSHA256 != "" {
		t.Fatalf("reimported project = %+v; want changed source and stale approval", updated)
	}
}

func TestLocalizationServiceRejectsStaleCSVAndImportsCurrentRowsAsDraft(t *testing.T) {
	service, repository, projectID := newLocalizationServiceWithProject(t)
	stale := []byte("resource_key,source_text,target_text,locale\ntitle,Old,Старое,ru-RU\n")
	if _, err := service.ImportCSV(context.Background(), projectID, stale); err == nil {
		t.Fatal("ImportCSV() accepted a stale source string")
	}
	if repository.translationSaves != 0 {
		t.Fatalf("stale CSV wrote %d translations", repository.translationSaves)
	}
	current := []byte("resource_key,source_text,target_text,locale\ntitle,Home,Главная,ru-RU\n")
	project, err := service.ImportCSV(context.Background(), projectID, current)
	if err != nil {
		t.Fatalf("ImportCSV() current CSV error = %v", err)
	}
	if len(project.Translations) != 1 || project.Translations[0].Status != domain.TranslationStatusDraft {
		t.Fatalf("CSV imported translations = %+v, want one draft", project.Translations)
	}
}

func TestLocalizationServiceBuildsPackageOnlyAfterApproval(t *testing.T) {
	service, _, projectID := newLocalizationServiceWithProject(t)
	_, err := service.UpdateTranslation(context.Background(), projectID, "title", "Главная")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.BuildLanguagePackage(context.Background(), projectID); err == nil {
		t.Fatal("BuildLanguagePackage() accepted an unapproved translation")
	}
	if _, err := service.ApproveTranslation(context.Background(), projectID, "title"); err != nil {
		t.Fatal(err)
	}
	item, err := service.BuildLanguagePackage(context.Background(), projectID)
	if err != nil {
		t.Fatalf("BuildLanguagePackage() error = %v", err)
	}
	content, err := base64.StdEncoding.DecodeString(item.ContentBase64)
	if err != nil || len(content) == 0 || item.SHA256 == "" || item.Size != len(content) {
		t.Fatalf("package artifact = %+v; decode error=%v", item, err)
	}
}

func TestLocalizationServiceHonorsCanceledContext(t *testing.T) {
	service, _, projectID := newLocalizationServiceWithProject(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.GetProject(ctx, projectID); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetProject() error = %v, want context.Canceled", err)
	}
}

func newLocalizationServiceWithProject(t *testing.T) (*LocalizationService, *localizationRepositoryFake, string) {
	t.Helper()
	repository := &localizationRepositoryFake{}
	service, err := NewLocalizationService(repository)
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.ImportProject(context.Background(), ImportLocalizationRequest{
		Name: "Synthetic UI", Locale: "ru-RU", Format: domain.LocalizationFormatAndroidXML,
		Filename: "strings.xml", Content: []byte(`<resources><string name="title">Home</string></resources>`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, project.ID
}

type localizationRepositoryFake struct {
	project          domain.LocalizationProject
	translationSaves int
	validationSaves  int
}

func (r *localizationRepositoryFake) CreateLocalizationProject(_ context.Context, project domain.LocalizationProject) (domain.LocalizationProject, error) {
	project.ID = "project-1"
	r.project = project
	return project, nil
}

func (r *localizationRepositoryFake) ListLocalizationProjects(context.Context) ([]domain.LocalizationProjectSummary, error) {
	return []domain.LocalizationProjectSummary{}, nil
}

func (r *localizationRepositoryFake) GetLocalizationProject(_ context.Context, id string) (domain.LocalizationProject, error) {
	if id != r.project.ID {
		return domain.LocalizationProject{}, domain.ErrLocalizationProjectNotFound
	}
	return r.project, nil
}

func (r *localizationRepositoryFake) ReplaceLocalizationSource(_ context.Context, id string, format domain.LocalizationFormat, filename, sourceHash string, source []byte, resources []domain.LocalizationResource) error {
	if id != r.project.ID {
		return domain.ErrLocalizationProjectNotFound
	}
	byKey := make(map[string]domain.LocalizationResource, len(resources))
	for _, resource := range resources {
		byKey[resource.Key] = resource
	}
	for i := range r.project.Resources {
		updated, ok := byKey[r.project.Resources[i].Key]
		if !ok {
			r.project.Resources[i].Present = false
			continue
		}
		r.project.Resources[i] = updated
	}
	for key, resource := range byKey {
		if _, ok := findResource(r.project.Resources, key); !ok {
			r.project.Resources = append(r.project.Resources, resource)
		}
	}
	for i := range r.project.Translations {
		for _, resource := range resources {
			translation := &r.project.Translations[i]
			if translation.ResourceKey == resource.Key && translation.Status == domain.TranslationStatusApproved && translation.ApprovedSourceSHA256 != resource.SourceSHA256 {
				translation.Status = domain.TranslationStatusNeedsReview
				translation.ApprovedSourceSHA256 = ""
			}
		}
	}
	r.project.Format, r.project.Filename, r.project.SourceSHA256, r.project.Source = format, filename, sourceHash, source
	return nil
}

func (r *localizationRepositoryFake) SaveLocalizationTranslation(_ context.Context, _ string, translation domain.LocalizationTranslation) (domain.LocalizationTranslation, error) {
	r.translationSaves++
	updated := false
	for i := range r.project.Translations {
		if r.project.Translations[i].ResourceKey == translation.ResourceKey && r.project.Translations[i].Locale == translation.Locale {
			r.project.Translations[i] = translation
			updated = true
		}
	}
	if !updated {
		r.project.Translations = append(r.project.Translations, translation)
	}
	return translation, nil
}

func (r *localizationRepositoryFake) SaveLocalizationTranslations(ctx context.Context, id string, translations []domain.LocalizationTranslation) error {
	for _, translation := range translations {
		if _, err := r.SaveLocalizationTranslation(ctx, id, translation); err != nil {
			return err
		}
	}
	return nil
}

func (r *localizationRepositoryFake) SaveLocalizationValidation(context.Context, string, domain.LocalizationValidationReport) (string, error) {
	r.validationSaves++
	return "validation-1", nil
}

func (r *localizationRepositoryFake) SaveLocalizationPackage(_ context.Context, _ string, item domain.LocalizationPackage, _ string, _ []byte) (domain.LocalizationPackage, error) {
	item.ID = "package-1"
	return item, nil
}
