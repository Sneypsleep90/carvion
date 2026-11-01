package application

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/localization"
)

const maxLocalizationSourceBytes = 10 << 20

type LocalizationRepository interface {
	CreateLocalizationProject(context.Context, domain.LocalizationProject) (domain.LocalizationProject, error)
	ListLocalizationProjects(context.Context) ([]domain.LocalizationProjectSummary, error)
	GetLocalizationProject(context.Context, string) (domain.LocalizationProject, error)
	ReplaceLocalizationSource(context.Context, string, domain.LocalizationFormat, string, string, []byte, []domain.LocalizationResource) error
	SaveLocalizationTranslation(context.Context, string, domain.LocalizationTranslation) (domain.LocalizationTranslation, error)
	SaveLocalizationTranslations(context.Context, string, []domain.LocalizationTranslation) error
	SaveLocalizationValidation(context.Context, string, domain.LocalizationValidationReport) (string, error)
	SaveLocalizationPackage(context.Context, string, domain.LocalizationPackage, string, []byte) (domain.LocalizationPackage, error)
	GetLocalizationPackage(context.Context, string, string) ([]byte, error)
}

type LocalizationService struct {
	repository LocalizationRepository
}

func (s *Service) localizationService() (*LocalizationService, error) {
	if s == nil || s.localization == nil {
		return nil, fmt.Errorf("%w: localization repository is not configured", domain.ErrLocalizationUnavailable)
	}
	return s.localization, nil
}

func (s *Service) ImportLocalizationProject(ctx context.Context, request ImportLocalizationRequest) (domain.LocalizationProject, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return service.ImportProject(ctx, request)
}

func (s *Service) ListLocalizationProjects(ctx context.Context) ([]domain.LocalizationProjectSummary, error) {
	service, err := s.localizationService()
	if err != nil {
		return nil, err
	}
	return service.ListProjects(ctx)
}

func (s *Service) GetLocalizationProject(ctx context.Context, projectID string) (domain.LocalizationProject, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return service.GetProject(ctx, projectID)
}

func (s *Service) ReplaceLocalizationSource(ctx context.Context, projectID, filename string, content []byte) (domain.LocalizationProject, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return service.ReplaceSource(ctx, projectID, filename, content)
}

func (s *Service) UpdateLocalizationTranslation(ctx context.Context, projectID, resourceKey, targetText string) (domain.LocalizationTranslation, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationTranslation{}, err
	}
	return service.UpdateTranslation(ctx, projectID, resourceKey, targetText)
}

func (s *Service) ApproveLocalizationTranslation(ctx context.Context, projectID, resourceKey string) (domain.LocalizationTranslation, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationTranslation{}, err
	}
	return service.ApproveTranslation(ctx, projectID, resourceKey)
}

func (s *Service) ImportLocalizationCSV(ctx context.Context, projectID string, content []byte) (domain.LocalizationProject, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return service.ImportCSV(ctx, projectID, content)
}

func (s *Service) ExportLocalizationCSV(ctx context.Context, projectID string) (domain.LocalizationArtifact, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	return service.ExportCSV(ctx, projectID)
}

func (s *Service) ExportLocalizationResources(ctx context.Context, projectID string) (domain.LocalizationArtifact, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	return service.ExportResources(ctx, projectID)
}

func (s *Service) ValidateLocalizationProject(ctx context.Context, projectID string) (domain.LocalizationValidationReport, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationValidationReport{}, err
	}
	return service.ValidateProject(ctx, projectID)
}

func (s *Service) BuildLocalizationPackage(ctx context.Context, projectID string) (domain.LocalizationPackage, error) {
	service, err := s.localizationService()
	if err != nil {
		return domain.LocalizationPackage{}, err
	}
	return service.BuildLanguagePackage(ctx, projectID)
}

type ImportLocalizationRequest struct {
	Name     string
	Locale   string
	Format   domain.LocalizationFormat
	Filename string
	Content  []byte
}

func NewLocalizationService(repository LocalizationRepository) (*LocalizationService, error) {
	if repository == nil {
		return nil, errorsMissingDependency("localization repository")
	}
	return &LocalizationService{repository: repository}, nil
}

func (s *LocalizationService) ImportProject(ctx context.Context, request ImportLocalizationRequest) (domain.LocalizationProject, error) {
	if err := contextError(ctx); err != nil {
		return domain.LocalizationProject{}, err
	}
	for _, field := range []struct{ name, value string }{{"project.name", request.Name}, {"locale", request.Locale}, {"filename", request.Filename}} {
		if err := validateField(field.name, field.value); err != nil {
			return domain.LocalizationProject{}, err
		}
	}
	if len(request.Content) == 0 || len(request.Content) > maxLocalizationSourceBytes {
		return domain.LocalizationProject{}, invalidArgument("resource file must be between 1 byte and %d bytes", maxLocalizationSourceBytes)
	}
	doc, err := localization.Parse(request.Format, request.Filename, request.Content)
	if err != nil {
		return domain.LocalizationProject{}, invalidArgument("parse resource file: %v", err)
	}
	project := domain.LocalizationProject{
		Name: strings.TrimSpace(request.Name), Locale: strings.TrimSpace(request.Locale),
		Format: request.Format, Filename: request.Filename,
		SourceSHA256: localization.SHA256(request.Content), Source: append([]byte(nil), request.Content...),
		Resources: doc.Resources, Translations: []domain.LocalizationTranslation{},
	}
	created, err := s.repository.CreateLocalizationProject(ctx, project)
	if err != nil {
		return domain.LocalizationProject{}, fmt.Errorf("create localization project: %w", err)
	}
	return created, nil
}

func (s *LocalizationService) ListProjects(ctx context.Context) ([]domain.LocalizationProjectSummary, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	projects, err := s.repository.ListLocalizationProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("list localization projects: %w", err)
	}
	return projects, nil
}

func (s *LocalizationService) GetProject(ctx context.Context, projectID string) (domain.LocalizationProject, error) {
	if err := contextError(ctx); err != nil {
		return domain.LocalizationProject{}, err
	}
	if err := validateField("project_id", projectID); err != nil {
		return domain.LocalizationProject{}, err
	}
	return s.repository.GetLocalizationProject(ctx, projectID)
}

func (s *LocalizationService) ReplaceSource(ctx context.Context, projectID, filename string, content []byte) (domain.LocalizationProject, error) {
	if err := contextError(ctx); err != nil {
		return domain.LocalizationProject{}, err
	}
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	if err := validateField("filename", filename); err != nil {
		return domain.LocalizationProject{}, err
	}
	if len(content) == 0 || len(content) > maxLocalizationSourceBytes {
		return domain.LocalizationProject{}, invalidArgument("resource file must be between 1 byte and %d bytes", maxLocalizationSourceBytes)
	}
	doc, err := localization.Parse(project.Format, filename, content)
	if err != nil {
		return domain.LocalizationProject{}, invalidArgument("parse replacement source: %v", err)
	}
	if err := s.repository.ReplaceLocalizationSource(ctx, projectID, project.Format, filename, localization.SHA256(content), content, doc.Resources); err != nil {
		return domain.LocalizationProject{}, fmt.Errorf("replace localization source: %w", err)
	}
	return s.GetProject(ctx, projectID)
}

func (s *LocalizationService) UpdateTranslation(ctx context.Context, projectID, resourceKey, targetText string) (domain.LocalizationTranslation, error) {
	if err := contextError(ctx); err != nil {
		return domain.LocalizationTranslation{}, err
	}
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationTranslation{}, err
	}
	resource, ok := findResource(project.Resources, resourceKey)
	if !ok || !resource.Translatable {
		return domain.LocalizationTranslation{}, invalidArgument("resource key %q is unknown or not translatable", resourceKey)
	}
	if !utf8.ValidString(targetText) {
		return domain.LocalizationTranslation{}, invalidArgument("translation is not valid UTF-8")
	}
	status := domain.TranslationStatusDraft
	for _, existing := range project.Translations {
		if existing.ResourceKey == resourceKey && existing.Locale == project.Locale && existing.Status == domain.TranslationStatusApproved {
			status = domain.TranslationStatusNeedsReview
			break
		}
	}
	translation := domain.LocalizationTranslation{ResourceKey: resourceKey, Locale: project.Locale, TargetText: targetText, Status: status, UpdatedAt: time.Now().UTC()}
	return s.repository.SaveLocalizationTranslation(ctx, projectID, translation)
}

func (s *LocalizationService) ApproveTranslation(ctx context.Context, projectID, resourceKey string) (domain.LocalizationTranslation, error) {
	if err := contextError(ctx); err != nil {
		return domain.LocalizationTranslation{}, err
	}
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationTranslation{}, err
	}
	resource, ok := findResource(project.Resources, resourceKey)
	if !ok || !resource.Translatable {
		return domain.LocalizationTranslation{}, invalidArgument("resource key %q is unknown or not translatable", resourceKey)
	}
	var translation domain.LocalizationTranslation
	for _, item := range project.Translations {
		if item.ResourceKey == resourceKey && item.Locale == project.Locale {
			translation = item
			break
		}
	}
	if strings.TrimSpace(translation.TargetText) == "" {
		return domain.LocalizationTranslation{}, invalidArgument("translation %q is empty", resourceKey)
	}
	translation.Status = domain.TranslationStatusApproved
	translation.ApprovedSourceSHA256 = resource.SourceSHA256
	translation.UpdatedAt = time.Now().UTC()
	return s.repository.SaveLocalizationTranslation(ctx, projectID, translation)
}

func (s *LocalizationService) ImportCSV(ctx context.Context, projectID string, content []byte) (domain.LocalizationProject, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	rows, err := localization.ParseTranslationCSV(content)
	if err != nil {
		return domain.LocalizationProject{}, invalidArgument("parse translation CSV: %v", err)
	}
	resources := make(map[string]domain.LocalizationResource, len(project.Resources))
	for _, resource := range project.Resources {
		resources[resource.Key] = resource
	}
	translations := make([]domain.LocalizationTranslation, 0, len(rows))
	for _, row := range rows {
		resource, exists := resources[row.ResourceKey]
		if !exists || !resource.Translatable {
			return domain.LocalizationProject{}, invalidArgument("CSV references unknown or non-translatable resource %q", row.ResourceKey)
		}
		if row.SourceText != resource.SourceText {
			return domain.LocalizationProject{}, invalidArgument("source text for %q differs from the current project; re-export the CSV", row.ResourceKey)
		}
		if row.Locale != project.Locale {
			return domain.LocalizationProject{}, invalidArgument("CSV locale %q does not match project locale %q", row.Locale, project.Locale)
		}
		translations = append(translations, domain.LocalizationTranslation{ResourceKey: row.ResourceKey, Locale: row.Locale, TargetText: row.TargetText, Status: domain.TranslationStatusDraft, UpdatedAt: time.Now().UTC()})
	}
	if err := s.repository.SaveLocalizationTranslations(ctx, projectID, translations); err != nil {
		return domain.LocalizationProject{}, fmt.Errorf("import localization translations: %w", err)
	}
	return s.GetProject(ctx, projectID)
}

func (s *LocalizationService) ExportCSV(ctx context.Context, projectID string) (domain.LocalizationArtifact, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	content, err := localization.ExportTranslationCSV(project.Resources, project.Translations, project.Locale)
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	return artifact(project.Filename+".translations.csv", content), nil
}

func (s *LocalizationService) ExportResources(ctx context.Context, projectID string) (domain.LocalizationArtifact, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	doc, err := localization.Parse(project.Format, project.Filename, project.Source)
	if err != nil {
		return domain.LocalizationArtifact{}, fmt.Errorf("reparse saved localization source: %w", err)
	}
	content, err := localization.Export(doc, project.Translations)
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	return artifact(project.Filename, content), nil
}

func (s *LocalizationService) ValidateProject(ctx context.Context, projectID string) (domain.LocalizationValidationReport, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationValidationReport{}, err
	}
	report := localization.Validate(project.Resources, project.Translations)
	report.CheckedAt = time.Now().UTC()
	report.Locale = project.Locale
	if _, err := s.repository.SaveLocalizationValidation(ctx, projectID, report); err != nil {
		return domain.LocalizationValidationReport{}, fmt.Errorf("save localization validation: %w", err)
	}
	return report, nil
}

func (s *LocalizationService) BuildLanguagePackage(ctx context.Context, projectID string) (domain.LocalizationPackage, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return domain.LocalizationPackage{}, err
	}
	report := localization.Validate(project.Resources, project.Translations)
	report.CheckedAt = time.Now().UTC()
	report.Locale = project.Locale
	reportID, err := s.repository.SaveLocalizationValidation(ctx, projectID, report)
	if err != nil {
		return domain.LocalizationPackage{}, fmt.Errorf("save package validation: %w", err)
	}
	if !report.Valid {
		return domain.LocalizationPackage{}, invalidArgument("cannot build language package: validation failed")
	}
	for _, resource := range project.Resources {
		if !resource.Translatable || !resource.Present {
			continue
		}
		approved := false
		for _, translation := range project.Translations {
			if translation.ResourceKey == resource.Key && translation.Locale == project.Locale && translation.Status == domain.TranslationStatusApproved && translation.ApprovedSourceSHA256 == resource.SourceSHA256 {
				approved = true
				break
			}
		}
		if !approved {
			return domain.LocalizationPackage{}, invalidArgument("translation %q must be approved for the current source before packaging", resource.Key)
		}
	}
	doc, err := localization.Parse(project.Format, project.Filename, project.Source)
	if err != nil {
		return domain.LocalizationPackage{}, err
	}
	resourceBytes, err := localization.Export(doc, project.Translations)
	if err != nil {
		return domain.LocalizationPackage{}, err
	}
	content, hash, err := localization.BuildPackage(localization.PackageInput{
		ProjectID: project.ID, ProjectName: project.Name, Locale: project.Locale,
		Format: project.Format, Filename: project.Filename, SourceSHA256: project.SourceSHA256,
		ResourceContent: resourceBytes, Report: report,
	})
	if err != nil {
		return domain.LocalizationPackage{}, err
	}
	item := domain.LocalizationPackage{Filename: project.Name + "-" + project.Locale + ".carvion.zip", SHA256: hash, Size: len(content), ContentBase64: base64.StdEncoding.EncodeToString(content)}
	return s.repository.SaveLocalizationPackage(ctx, projectID, item, reportID, content)
}

func (s *LocalizationService) GetPackageContent(ctx context.Context, projectID, packageSHA256 string) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if err := validateField("project_id", projectID); err != nil {
		return nil, err
	}
	if err := validateField("package_sha256", packageSHA256); err != nil {
		return nil, err
	}
	return s.repository.GetLocalizationPackage(ctx, projectID, packageSHA256)
}

func findResource(resources []domain.LocalizationResource, key string) (domain.LocalizationResource, bool) {
	for _, resource := range resources {
		if resource.Key == key {
			return resource, true
		}
	}
	return domain.LocalizationResource{}, false
}

func artifact(filename string, content []byte) domain.LocalizationArtifact {
	return domain.LocalizationArtifact{Filename: filename, ContentBase64: base64.StdEncoding.EncodeToString(content)}
}
