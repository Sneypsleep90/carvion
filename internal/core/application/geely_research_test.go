package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	research "github.com/Sneypsleep90/carvion/internal/geelyresearch"
)

const testGeelyProfileJSON = `{"schema":"carvion.geely-research/v1","profile_id":"test-only-profile","vehicle":{"manufacturer":"TEST ONLY","model":"TEST ONLY","model_year":2024,"market":"TEST ONLY","trim":"TEST ONLY"},"infotainment":{"platform":"TEST ONLY","os_name":"TEST ONLY","os_version":"TEST ONLY","hardware_identifier":"TEST ONLY","hardware_revision":"TEST ONLY","firmware_version":"TEST ONLY","firmware_build_id":"TEST ONLY"}}`

type geelyResearchGatewayStub struct {
	*adapterGatewayStub
	profileRequest []byte
	profileResult  domain.CompatibilityResult
	profileErr     error
	info           domain.AdapterInfo
}

func newGeelyResearchGatewayStub() *geelyResearchGatewayStub {
	return &geelyResearchGatewayStub{adapterGatewayStub: &adapterGatewayStub{
		detection:  domain.DetectionResult{Status: domain.StatusUnknown, Reason: "imported profile only"},
		inspection: domain.ResourceInspection{Resources: []domain.TranslatableResource{{ID: "title", Text: "Synthetic title"}}},
	}, profileResult: domain.CompatibilityResult{Status: domain.StatusUnknown, Reason: "no exact reviewed record"}}
}

func (s *geelyResearchGatewayStub) GetInfo(ctx context.Context, adapterID string) (domain.AdapterInfo, error) {
	s.calls = append(s.calls, "get_info")
	s.gotContext, s.gotID = ctx, adapterID
	info := s.info
	if info.ID == "" {
		info = domain.AdapterInfo{ID: adapterID, Status: domain.AdapterStatusAvailable, Capabilities: domain.AdapterCapabilities{Detect: true, CheckCompatibility: true, InspectResources: true}, InputTypes: []string{"vehicle-research-profile-json"}, ResourceFormats: []string{"android-strings-xml", "carvion-json-v1"}}
	}
	return info, s.err
}

func (s *geelyResearchGatewayStub) CheckCompatibilityWithVehicleProfile(ctx context.Context, adapterID string, request domain.CompatibilityRequest, profile json.RawMessage) (domain.CompatibilityResult, error) {
	s.calls = append(s.calls, "compatibility_with_profile")
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	s.profileRequest = append([]byte(nil), profile...)
	return s.profileResult, s.profileErr
}

func TestAnalyzeGeelyProfileValidatesAndUsesTheFixedAdapter(t *testing.T) {
	gateway := newGeelyResearchGatewayStub()
	service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
	ctx := context.Background()

	result, err := service.AnalyzeGeelyProfile(ctx, []byte(testGeelyProfileJSON))
	if err != nil {
		t.Fatalf("AnalyzeGeelyProfile() error = %v", err)
	}
	if result.Compatibility.Status != domain.StatusUnknown || result.Detection.Status != domain.StatusUnknown {
		t.Fatalf("analysis = %+v, want unknown imported profile", result)
	}
	if gateway.gotID != "adapter-geely-research" || string(gateway.profileRequest) != testGeelyProfileJSON {
		t.Fatalf("gateway received adapter=%q profile=%s", gateway.gotID, gateway.profileRequest)
	}
	if result.Profile.Origin != string(research.ProfileOriginImportedFile) {
		t.Fatalf("profile origin = %q, want imported and unverified", result.Profile.Origin)
	}
}

func TestAnalyzeGeelyProfileRejectsMalformedProfileBeforeAdapterCall(t *testing.T) {
	gateway := newGeelyResearchGatewayStub()
	service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
	_, err := service.AnalyzeGeelyProfile(context.Background(), []byte(`{"schema":"bad"}`))
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("AnalyzeGeelyProfile() error = %v, want invalid argument", err)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("malformed profile made adapter calls: %v", gateway.calls)
	}
}

func TestAnalyzeGeelyProfilePropagatesOnlyExplicitCatalogStatuses(t *testing.T) {
	for _, status := range []domain.CompatibilityStatus{domain.StatusSupported, domain.StatusUnsupported} {
		t.Run(string(status), func(t *testing.T) {
			gateway := newGeelyResearchGatewayStub()
			gateway.profileResult = domain.CompatibilityResult{Status: status, Reason: "synthetic test catalog entry"}
			service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
			result, err := service.AnalyzeGeelyProfile(context.Background(), []byte(testGeelyProfileJSON))
			if err != nil || result.Compatibility.Status != status {
				t.Fatalf("AnalyzeGeelyProfile() = %+v, %v; want exact test status %q", result, err, status)
			}
		})
	}
}

func TestAnalyzeGeelyProfileRequiresDeclaredAdapterCapabilities(t *testing.T) {
	gateway := newGeelyResearchGatewayStub()
	gateway.info = domain.AdapterInfo{ID: "adapter-geely-research", Status: domain.AdapterStatusAvailable, InputTypes: []string{"vehicle-research-profile-json"}, Capabilities: domain.AdapterCapabilities{CheckCompatibility: true}}
	service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
	if _, err := service.AnalyzeGeelyProfile(context.Background(), []byte(testGeelyProfileJSON)); !errors.Is(err, domain.ErrCapabilityUnavailable) {
		t.Fatalf("AnalyzeGeelyProfile() error = %v, want missing capability", err)
	}
	if len(gateway.calls) != 1 || gateway.calls[0] != "get_info" {
		t.Fatalf("gateway calls after capability rejection = %v, want GetInfo only", gateway.calls)
	}
}

func TestAnalyzeGeelyProfilePropagatesAdapterCompatibilityError(t *testing.T) {
	wantErr := errors.New("synthetic adapter RPC failure")
	gateway := newGeelyResearchGatewayStub()
	gateway.profileErr = wantErr
	service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
	if _, err := service.AnalyzeGeelyProfile(context.Background(), []byte(testGeelyProfileJSON)); !errors.Is(err, wantErr) {
		t.Fatalf("AnalyzeGeelyProfile() error = %v, want adapter RPC failure", err)
	}
}

func TestAnalyzeGeelyProfileReturnsUnknownWithoutCompatibilityCallForMissingExactFields(t *testing.T) {
	partial := `{"schema":"carvion.geely-research/v1","profile_id":"test-partial","vehicle":{"manufacturer":"TEST ONLY","model":"TEST ONLY"},"infotainment":{"platform":"TEST ONLY"}}`
	gateway := newGeelyResearchGatewayStub()
	service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
	result, err := service.AnalyzeGeelyProfile(context.Background(), []byte(partial))
	if err != nil {
		t.Fatalf("AnalyzeGeelyProfile() error = %v", err)
	}
	if result.Compatibility.Status != domain.StatusUnknown {
		t.Fatalf("compatibility = %+v, want unknown for incomplete profile", result.Compatibility)
	}
	for _, call := range gateway.calls {
		if call == "compatibility_with_profile" {
			t.Fatal("compatibility gateway called with incomplete tuple")
		}
	}
}

func TestGeelyResourceAnalysisAndExplicitLocalizationImportUseSameContent(t *testing.T) {
	gateway := newGeelyResearchGatewayStub()
	repository := &localizationRepositoryFake{}
	service := newTestServiceWithLocalization(t, gateway, repository)
	content := `<resources><string name="title">Synthetic title</string></resources>`
	request := domain.GeelyLocalizationImportRequest{
		ProfileJSON: testGeelyProfileJSON, Name: "Synthetic research", Locale: "ru-RU",
		Format: domain.LocalizationFormatAndroidXML, Filename: "strings.xml", Content: content,
	}

	inspection, err := service.InspectGeelyResources(context.Background(), domain.GeelyResourceRequest{Filename: "strings.xml", Format: domain.LocalizationFormatAndroidXML, Content: content})
	if err != nil || len(inspection.Resources) != 1 || inspection.Resources[0].ID != "title" {
		t.Fatalf("InspectGeelyResources() = %+v, %v", inspection, err)
	}
	result, err := service.ImportGeelyLocalizationProject(context.Background(), request)
	if err != nil {
		t.Fatalf("ImportGeelyLocalizationProject() error = %v", err)
	}
	if result.Analysis.Compatibility.Status != domain.StatusUnknown || result.Project.ID != "project-1" || len(result.Project.Resources) != 1 {
		t.Fatalf("localization import result = %+v", result)
	}
	if string(gateway.gotRequest.(domain.InspectResourcesRequest).Content) != content {
		t.Fatalf("gateway resource content differs from imported resource")
	}
}

func TestGeelyLocalizationRejectsPathAndAdapterParserMismatchBeforePersistence(t *testing.T) {
	for _, test := range []struct {
		name       string
		filename   string
		inspection domain.ResourceInspection
	}{
		{name: "path is rejected", filename: "../strings.xml", inspection: domain.ResourceInspection{Resources: []domain.TranslatableResource{{ID: "title", Text: "Synthetic title"}}}},
		{name: "parser mismatch is rejected", filename: "strings.xml", inspection: domain.ResourceInspection{Resources: []domain.TranslatableResource{{ID: "other", Text: "Synthetic title"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			gateway := newGeelyResearchGatewayStub()
			gateway.inspection = test.inspection
			repository := &localizationRepositoryFake{}
			service := newTestServiceWithLocalization(t, gateway, repository)
			request := domain.GeelyLocalizationImportRequest{
				ProfileJSON: testGeelyProfileJSON, Name: "Synthetic research", Locale: "ru-RU",
				Format: domain.LocalizationFormatAndroidXML, Filename: test.filename, Content: `<resources><string name="title">Synthetic title</string></resources>`,
			}
			if _, err := service.ImportGeelyLocalizationProject(context.Background(), request); err == nil {
				t.Fatal("ImportGeelyLocalizationProject() accepted invalid file/parser output")
			}
			if repository.project.ID != "" {
				t.Fatalf("project persisted before resource checks: %+v", repository.project)
			}
			if test.name == "path is rejected" {
				for _, call := range gateway.calls {
					if call == "inspect_resources" {
						t.Fatal("filesystem path was forwarded to the adapter")
					}
				}
			}
		})
	}
}

func TestBuildGeelyDiagnosticReportExcludesResourcePathAndContents(t *testing.T) {
	gateway := newGeelyResearchGatewayStub()
	gateway.inspection = domain.ResourceInspection{Resources: []domain.TranslatableResource{{ID: "title", Text: "SECRET RESOURCE TEXT"}}}
	service := newTestServiceWithLocalization(t, gateway, &localizationRepositoryFake{})
	resource := domain.GeelyResourceRequest{Filename: "strings.xml", Format: domain.LocalizationFormatAndroidXML, Content: `<resources><string name="title">SECRET RESOURCE TEXT</string></resources>`}
	report, err := service.BuildGeelyDiagnosticReport(context.Background(), domain.GeelyDiagnosticReportRequest{ProfileJSON: testGeelyProfileJSON, Resource: &resource})
	if err != nil {
		t.Fatalf("BuildGeelyDiagnosticReport() error = %v", err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"SECRET RESOURCE TEXT", "../", "/Users/", "VIN", "credentials"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("diagnostic report contains %q: %s", forbidden, encoded)
		}
	}
	if !strings.Contains(string(encoded), "sha256") || !strings.Contains(string(encoded), "title") {
		t.Fatalf("diagnostic report missing resource hash/key metadata: %s", encoded)
	}
}

func TestGeelyUseCasesHonorCanceledContext(t *testing.T) {
	service := newTestServiceWithLocalization(t, newGeelyResearchGatewayStub(), &localizationRepositoryFake{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ListGeelyResearchEvidence(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListGeelyResearchEvidence() error = %v, want canceled", err)
	}
	if _, err := service.AnalyzeGeelyProfile(ctx, []byte(testGeelyProfileJSON)); !errors.Is(err, context.Canceled) {
		t.Fatalf("AnalyzeGeelyProfile() error = %v, want canceled", err)
	}
}

func newTestServiceWithLocalization(t *testing.T, gateway AdapterGateway, repository LocalizationRepository) *Service {
	t.Helper()
	service, err := NewServiceWithLocalization(gateway, &catalogRepositoryStub{}, repository)
	if err != nil {
		t.Fatalf("NewServiceWithLocalization() error = %v", err)
	}
	return service
}
