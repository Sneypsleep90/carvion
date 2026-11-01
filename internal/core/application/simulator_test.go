package application

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/simulator"
)

func TestRunSimulatorScenarioCompletesVerifiedLocalizationPackage(t *testing.T) {
	profile, err := simulator.LoadProfile("demo-android-platform-a")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &simulatorGatewayStub{
		items: []domain.AdapterInfo{simulatorTestAdapterInfo()},
		detection: domain.DetectionResult{
			Status:   domain.StatusSupported,
			Identity: profile.Public.Identity,
		},
		compatibility: domain.CompatibilityResult{Status: domain.StatusSupported, Reason: "exact synthetic profile match"},
		inspection: domain.ResourceInspection{Resources: []domain.TranslatableResource{
			{ID: "app_title", Text: "演示多媒体系统"},
			{ID: "welcome_title", Text: "你好，欢迎使用演示系统"},
			{ID: "audio_mode_standard", Text: "标准模式"},
		}},
	}
	repository := &localizationRepositoryFake{}
	service := newSimulatorTestService(t, gateway, repository)

	result, err := service.RunSimulatorScenario(context.Background(), profile.Public.ID)
	if err != nil {
		t.Fatalf("RunSimulatorScenario() error = %v", err)
	}
	if result.Status != domain.SimulatorRunCompleted || !result.PackageVerified || result.AdapterID != "adapter-android-demo" {
		t.Fatalf("simulation result = %+v, want completed verified run through demo adapter", result)
	}
	if result.LocalizationProjectID == "" || result.Package.SHA256 == "" || result.Package.Size <= 0 {
		t.Fatalf("simulation result is missing its localization project or package metadata: %+v", result)
	}
	if len(gateway.calls) != 4 || strings.Join(gateway.calls, ",") != "list,detect,compatibility,inspect_resources" {
		t.Fatalf("adapter gateway calls = %v, want capability discovery and ordered simulator operations", gateway.calls)
	}
	wantEvents := []string{
		"scenario.started", "device.detected", "adapter.selected", "compatibility.checked",
		"resources.inspected", "localization.imported", "translations.prepared",
		"translations.validated", "package.built", "package.verified", "scenario.completed",
	}
	if len(result.Events) != len(wantEvents) {
		t.Fatalf("events = %+v, want steps %v", result.Events, wantEvents)
	}
	for i, want := range wantEvents {
		if result.Events[i].Step != want {
			t.Errorf("event[%d].Step = %q, want %q", i, result.Events[i].Step, want)
		}
	}
	if len(repository.project.Translations) != 3 {
		t.Fatalf("translations = %+v, want three synthetic Russian translations", repository.project.Translations)
	}
	for _, translation := range repository.project.Translations {
		if translation.Status != domain.TranslationStatusApproved || translation.Locale != "ru-RU" {
			t.Errorf("translation = %+v, want approved ru-RU translation", translation)
		}
	}
	verifySimulatorPackageForTest(t, result.Package.ContentBase64, result.Package.SHA256)
}

func TestRunSimulatorScenarioStopsForUnknownAndUnsupportedCompatibility(t *testing.T) {
	for _, test := range []struct {
		profileID string
		status    domain.CompatibilityStatus
		wantRun   domain.SimulatorRunStatus
	}{
		{profileID: "unknown-firmware", status: domain.StatusUnknown, wantRun: domain.SimulatorRunUnknown},
		{profileID: "incompatible-hardware", status: domain.StatusUnsupported, wantRun: domain.SimulatorRunUnsupported},
	} {
		t.Run(test.profileID, func(t *testing.T) {
			profile, err := simulator.LoadProfile(test.profileID)
			if err != nil {
				t.Fatal(err)
			}
			gateway := &simulatorGatewayStub{
				items:         []domain.AdapterInfo{simulatorTestAdapterInfo()},
				detection:     domain.DetectionResult{Status: domain.StatusSupported, Identity: profile.Public.Identity},
				compatibility: domain.CompatibilityResult{Status: test.status, Reason: "profile outcome"},
			}
			repository := &localizationRepositoryFake{}
			service := newSimulatorTestService(t, gateway, repository)

			result, err := service.RunSimulatorScenario(context.Background(), test.profileID)
			if err != nil {
				t.Fatalf("RunSimulatorScenario() error = %v", err)
			}
			if result.Status != test.wantRun || result.PackageVerified || result.LocalizationProjectID != "" {
				t.Fatalf("simulation result = %+v, want early %s without package", result, test.wantRun)
			}
			if repository.project.ID != "" {
				t.Fatalf("localization project was created for %s compatibility", test.status)
			}
			if got := strings.Join(gateway.calls, ","); got != "list,detect,compatibility" {
				t.Fatalf("adapter calls = %v, want no resource inspection after %s", gateway.calls, test.status)
			}
		})
	}
}

func TestRunSimulatorScenarioRejectsAdapterDetectionWithDifferentIdentity(t *testing.T) {
	profile, err := simulator.LoadProfile("demo-android-platform-a")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &simulatorGatewayStub{
		items: []domain.AdapterInfo{simulatorTestAdapterInfo()},
		detection: domain.DetectionResult{
			Status:   domain.StatusSupported,
			Identity: domain.VehicleIdentity{Manufacturer: "CARVION TEST LAB", Model: "Different Synthetic Unit", PlatformID: "android-demo-platform"},
		},
		compatibility: domain.CompatibilityResult{Status: domain.StatusSupported, Reason: "misreported supported response"},
	}
	repository := &localizationRepositoryFake{}
	service := newSimulatorTestService(t, gateway, repository)

	result, err := service.RunSimulatorScenario(context.Background(), profile.Public.ID)
	if err != nil {
		t.Fatalf("RunSimulatorScenario() error = %v", err)
	}
	if result.Status != domain.SimulatorRunUnknown || result.AdapterID != "" || repository.project.ID != "" {
		t.Fatalf("simulation result = %+v, project = %+v; want unknown without selecting the false-positive adapter", result, repository.project)
	}
	if got := strings.Join(gateway.calls, ","); got != "list,detect" {
		t.Fatalf("adapter calls = %v, want no compatibility check after identity mismatch", gateway.calls)
	}
}

func TestRunSimulatorScenarioReportsRPCFailureAsStructuredEvent(t *testing.T) {
	profile, err := simulator.LoadProfile("simulated-communication-failure")
	if err != nil {
		t.Fatal(err)
	}
	gateway := &simulatorGatewayStub{
		items:       []domain.AdapterInfo{simulatorTestAdapterInfo()},
		detection:   domain.DetectionResult{Status: domain.StatusUnknown},
		detectError: errors.New("simulated adapter communication failure"),
	}
	service := newSimulatorTestService(t, gateway, &localizationRepositoryFake{})

	result, err := service.RunSimulatorScenario(context.Background(), profile.Public.ID)
	if err != nil {
		t.Fatalf("simulated RPC error should be returned in the run result, got error %v", err)
	}
	if result.Status != domain.SimulatorRunFailed || len(result.Events) == 0 || result.Events[len(result.Events)-1].Status != "failed" {
		t.Fatalf("simulation result = %+v, want failed status and failure event", result)
	}
	if !strings.Contains(result.ErrorMessage, "communication failure") {
		t.Fatalf("error message = %q, want adapter communication failure", result.ErrorMessage)
	}
}

func TestRunSimulatorScenarioHonorsCancellationDuringAdapterCall(t *testing.T) {
	profile, err := simulator.LoadProfile("demo-android-platform-a")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	gateway := &simulatorGatewayStub{
		items:             []domain.AdapterInfo{simulatorTestAdapterInfo()},
		detection:         domain.DetectionResult{Status: domain.StatusSupported, Identity: profile.Public.Identity},
		cancelAfterDetect: cancel,
	}
	service := newSimulatorTestService(t, gateway, &localizationRepositoryFake{})

	result, err := service.RunSimulatorScenario(ctx, profile.Public.ID)
	if err != nil {
		t.Fatalf("cancellation during scenario should be represented in the result, got error %v", err)
	}
	if result.Status != domain.SimulatorRunFailed || !strings.Contains(result.ErrorMessage, context.Canceled.Error()) {
		t.Fatalf("simulation result = %+v, want failed cancelled run", result)
	}
	if got := result.Events[len(result.Events)-1].Step; got != "scenario.cancelled" {
		t.Fatalf("final event = %q, want scenario.cancelled", got)
	}
	if got := strings.Join(gateway.calls, ","); got != "list,detect" {
		t.Fatalf("adapter calls after cancellation = %v, want stop after Detect", gateway.calls)
	}
}

func TestRunSimulatorScenarioRejectsProfileOutsideLocalCatalog(t *testing.T) {
	gateway := &simulatorGatewayStub{}
	service := newSimulatorTestService(t, gateway, &localizationRepositoryFake{})

	if _, err := service.RunSimulatorScenario(context.Background(), "unknown-car"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("RunSimulatorScenario() error = %v, want invalid argument", err)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("adapter called for unlisted profile: %v", gateway.calls)
	}
}

func newSimulatorTestService(t *testing.T, gateway AdapterGateway, repository LocalizationRepository) *Service {
	t.Helper()
	service, err := NewServiceWithLocalization(gateway, &catalogRepositoryStub{}, repository)
	if err != nil {
		t.Fatalf("NewServiceWithLocalization() error = %v", err)
	}
	return service
}

func simulatorTestAdapterInfo() domain.AdapterInfo {
	return domain.AdapterInfo{
		ID: "adapter-android-demo", Capabilities: domain.AdapterCapabilities{Detect: true, CheckCompatibility: true, InspectResources: true},
		ResourceFormats: []string{"android-strings-xml"}, Status: domain.AdapterStatusAvailable,
	}
}

type simulatorGatewayStub struct {
	items             []domain.AdapterInfo
	detection         domain.DetectionResult
	compatibility     domain.CompatibilityResult
	inspection        domain.ResourceInspection
	detectError       error
	cancelAfterDetect context.CancelFunc
	calls             []string
}

func (s *simulatorGatewayStub) List(context.Context) ([]domain.AdapterInfo, error) {
	s.calls = append(s.calls, "list")
	return s.items, nil
}

func (s *simulatorGatewayStub) GetInfo(context.Context, string) (domain.AdapterInfo, error) {
	return domain.AdapterInfo{}, nil
}

func (s *simulatorGatewayStub) Detect(ctx context.Context, _ string, _ domain.DetectRequest) (domain.DetectionResult, error) {
	s.calls = append(s.calls, "detect")
	if s.cancelAfterDetect != nil {
		s.cancelAfterDetect()
		return domain.DetectionResult{}, ctx.Err()
	}
	return s.detection, s.detectError
}

func (s *simulatorGatewayStub) CheckCompatibility(context.Context, string, domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	s.calls = append(s.calls, "compatibility")
	return s.compatibility, nil
}

func (s *simulatorGatewayStub) InspectResources(context.Context, string, domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	s.calls = append(s.calls, "inspect_resources")
	return s.inspection, nil
}

func verifySimulatorPackageForTest(t *testing.T, contentBase64, expectedSHA string) {
	t.Helper()
	itemContent, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(itemContent)
	if got := hex.EncodeToString(sum[:]); got != expectedSHA {
		t.Fatalf("package SHA-256 = %s, want %s", got, expectedSHA)
	}
	reader, err := zip.NewReader(bytes.NewReader(itemContent), int64(len(itemContent)))
	if err != nil {
		t.Fatalf("package is not a valid ZIP: %v", err)
	}
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[file.Name] = file
	}
	manifestFile := files["manifest.json"]
	if manifestFile == nil || files["checksums.sha256"] == nil || files["validation.json"] == nil {
		t.Fatalf("package entries = %v; want manifest, checksums and validation report", files)
	}
	manifestReader, err := manifestFile.Open()
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := io.ReadAll(manifestReader)
	manifestReader.Close()
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(manifestBytes)
	if !strings.Contains(manifest, `"package_type": "carvion-localization"`) || !strings.Contains(manifest, `"oem_firmware": false`) {
		t.Fatalf("manifest = %s; want CARVION localization package marked non-OEM", manifest)
	}
	resourceFile := files["resources/strings.xml"]
	if resourceFile == nil {
		t.Fatalf("package resource entry missing: %v", files)
	}
	resourceReader, err := resourceFile.Open()
	if err != nil {
		t.Fatal(err)
	}
	resourceBytes, err := io.ReadAll(resourceReader)
	resourceReader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(resourceBytes), "Стандартный режим") || strings.Contains(string(resourceBytes), "标准模式") {
		t.Fatalf("packaged resources = %s; want prepared Russian text", resourceBytes)
	}
	if expectedSHA == "" {
		t.Fatal("package SHA-256 is empty")
	}
}
