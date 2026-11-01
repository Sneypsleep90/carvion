package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/config"
	"github.com/Sneypsleep90/carvion/internal/core/application"
	"github.com/Sneypsleep90/carvion/internal/core/domain"
	wailsapp "github.com/wailsapp/wails/v3/pkg/application"
)

func TestServiceStartupKeepsWindowAvailableWhenCoreCannotStart(t *testing.T) {
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, errors.New("DATABASE_URL is required")
		},
	}

	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() error = %v, want degraded startup", err)
	}

	state := service.GetApplicationState()
	if state.CoreStatus != "unavailable" {
		t.Errorf("CoreStatus = %q, want unavailable", state.CoreStatus)
	}
	if !strings.Contains(state.Message, "DATABASE_URL is required") {
		t.Errorf("Message = %q, want configuration error", state.Message)
	}
	if _, err := service.ListAdapters(); !errors.Is(err, ErrCoreUnavailable) {
		t.Errorf("ListAdapters() error = %v, want ErrCoreUnavailable", err)
	}
}

func TestServiceDelegatesToCoreAndClosesRuntime(t *testing.T) {
	core := &coreSpy{
		adapters: []domain.AdapterInfo{{
			ID:      "adapter-android-demo",
			Name:    "CARVION Synthetic Android Demo",
			Version: "0.1.0",
		}},
		compatibility: domain.CompatibilityResult{
			Status: domain.StatusSupported,
			Reason: "exact synthetic demo profile match",
		},
	}
	runtime := &runtimeSpy{}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, runtime, nil
		},
	}

	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() error = %v", err)
	}

	request := domain.CompatibilityRequest{
		Identity: domain.VehicleIdentity{Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit"},
	}
	result, err := service.CheckCompatibility("adapter-android-demo", request)
	if err != nil {
		t.Fatalf("CheckCompatibility() error = %v", err)
	}
	if result.Status != domain.StatusSupported {
		t.Errorf("Status = %q, want %q", result.Status, domain.StatusSupported)
	}
	if core.compatibilityCalls != 1 || core.compatibilityRequest.Identity.Model != request.Identity.Model {
		t.Errorf("Core compatibility call = %+v, want one delegated request", core.compatibilityRequest)
	}

	state := service.GetApplicationState()
	if state.CoreStatus != "ready" || state.DatabaseStatus != "connected" {
		t.Errorf("GetApplicationState() = %+v, want ready local Core", state)
	}
	if state.AdapterCount != 1 {
		t.Errorf("AdapterCount = %d, want 1", state.AdapterCount)
	}

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown() error = %v", err)
	}
	if !runtime.closed {
		t.Error("ServiceShutdown() did not close the local runtime")
	}
}

func TestServiceStartupRejectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) {
			t.Fatal("configuration loaded after startup context was cancelled")
			return config.Config{}, nil
		},
	}

	if err := service.ServiceStartup(ctx, wailsapp.ServiceOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ServiceStartup() error = %v, want context.Canceled", err)
	}
}

func TestLocalizationMethodsDelegateDirectlyToCore(t *testing.T) {
	core := &localizationCoreSpy{coreSpy: &coreSpy{}}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, &runtimeSpy{}, nil
		},
	}
	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	project, err := service.ImportLocalizationProject("Synthetic", "ru-RU", string(domain.LocalizationFormatAndroidXML), "strings.xml", "<resources />")
	if err != nil {
		t.Fatalf("ImportLocalizationProject() error = %v", err)
	}
	if project.ID != "local-project" || core.importRequest.Name != "Synthetic" || string(core.importRequest.Content) != "<resources />" {
		t.Fatalf("project = %+v, delegated request = %+v", project, core.importRequest)
	}
	if _, err := service.UpdateLocalizationTranslation(project.ID, "title", "Главная"); err != nil {
		t.Fatalf("UpdateLocalizationTranslation() error = %v", err)
	}
	if core.updateCalls != 1 || core.projectID != project.ID {
		t.Fatalf("translation update calls=%d projectID=%q", core.updateCalls, core.projectID)
	}
}

func TestSimulatorMethodsDelegateDirectlyToCoreWithWailsContext(t *testing.T) {
	wailsContext := context.WithValue(context.Background(), simulatorContextKey{}, "wails-context")
	core := &simulatorCoreSpy{coreSpy: &coreSpy{}}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, &runtimeSpy{}, nil
		},
	}
	if err := service.ServiceStartup(wailsContext, wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}

	profiles, err := service.ListSimulatorProfiles()
	if err != nil || len(profiles) != 1 || profiles[0].ID != "demo-profile" {
		t.Fatalf("ListSimulatorProfiles() = %+v, %v", profiles, err)
	}
	result, err := service.RunSimulatorScenario("demo-profile", "request-1")
	if err != nil || result.RunID != "run-1" {
		t.Fatalf("RunSimulatorScenario() = %+v, %v", result, err)
	}
	if core.profileCalls != 1 || core.runCalls != 1 || core.context == nil || core.profileID != "demo-profile" || core.context.Value(simulatorContextKey{}) != "wails-context" {
		t.Fatalf("simulator core calls profiles=%d run=%d context=%v profile=%q", core.profileCalls, core.runCalls, core.context, core.profileID)
	}
}

func TestSimulatorMethodsReportUnavailableCore(t *testing.T) {
	service := &ApplicationService{}
	if _, err := service.ListSimulatorProfiles(); !errors.Is(err, ErrCoreUnavailable) {
		t.Fatalf("ListSimulatorProfiles() error = %v, want ErrCoreUnavailable", err)
	}
	if _, err := service.RunSimulatorScenario("demo-profile", "request-1"); !errors.Is(err, ErrCoreUnavailable) {
		t.Fatalf("RunSimulatorScenario() error = %v, want ErrCoreUnavailable", err)
	}
}

func TestSimulatorCancellationStopsActiveCoreRun(t *testing.T) {
	core := &simulatorCoreSpy{coreSpy: &coreSpy{}, waitForCancel: true, started: make(chan struct{})}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, &runtimeSpy{}, nil
		},
	}
	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	resultChannel := make(chan domain.SimulatorRunResult, 1)
	errorChannel := make(chan error, 1)
	go func() {
		result, err := service.RunSimulatorScenario("demo-profile", "request-cancel")
		resultChannel <- result
		errorChannel <- err
	}()
	<-core.started
	if !service.CancelSimulatorScenario("request-cancel") {
		t.Fatal("CancelSimulatorScenario() did not find the active request")
	}
	result := <-resultChannel
	if err := <-errorChannel; err != nil {
		t.Fatalf("cancelled run returned error = %v", err)
	}
	if result.Status != domain.SimulatorRunFailed || result.ErrorMessage != context.Canceled.Error() {
		t.Fatalf("cancelled run result = %+v", result)
	}
	if service.CancelSimulatorScenario("request-cancel") {
		t.Fatal("CancelSimulatorScenario() reported a completed request as active")
	}
}

type coreSpy struct {
	adapters             []domain.AdapterInfo
	compatibility        domain.CompatibilityResult
	compatibilityRequest domain.CompatibilityRequest
	compatibilityCalls   int
}

func (c *coreSpy) ListAdapters(context.Context) ([]domain.AdapterInfo, error) {
	return c.adapters, nil
}

func (*coreSpy) ListManufacturers(context.Context) ([]domain.Manufacturer, error) {
	return []domain.Manufacturer{}, nil
}

func (*coreSpy) ListVehicleModels(context.Context, string) ([]domain.VehicleModel, error) {
	return []domain.VehicleModel{}, nil
}

func (*coreSpy) ListInfotainmentPlatforms(context.Context, string) ([]domain.InfotainmentPlatform, error) {
	return []domain.InfotainmentPlatform{}, nil
}

func (c *coreSpy) CheckCompatibility(_ context.Context, _ string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	c.compatibilityCalls++
	c.compatibilityRequest = request
	return c.compatibility, nil
}

func (*coreSpy) Detect(context.Context, string, domain.DetectRequest) (domain.DetectionResult, error) {
	return domain.DetectionResult{}, nil
}

func (*coreSpy) InspectResources(context.Context, string, domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	return domain.ResourceInspection{Resources: []domain.TranslatableResource{}}, nil
}

type runtimeSpy struct {
	pingErr error
	closed  bool
}

type localizationCoreSpy struct {
	*coreSpy
	importRequest application.ImportLocalizationRequest
	updateCalls   int
	projectID     string
}

type simulatorCoreSpy struct {
	*coreSpy
	profileCalls  int
	runCalls      int
	profileID     string
	context       context.Context
	waitForCancel bool
	started       chan struct{}
}

type simulatorContextKey struct{}

func (c *simulatorCoreSpy) ListSimulatorProfiles(ctx context.Context) ([]domain.SimulatorProfile, error) {
	c.profileCalls++
	c.context = ctx
	return []domain.SimulatorProfile{{ID: "demo-profile"}}, nil
}

func (c *simulatorCoreSpy) RunSimulatorScenario(ctx context.Context, profileID string) (domain.SimulatorRunResult, error) {
	c.runCalls++
	c.context = ctx
	c.profileID = profileID
	if c.waitForCancel {
		close(c.started)
		<-ctx.Done()
		return domain.SimulatorRunResult{RunID: "run-cancelled", Status: domain.SimulatorRunFailed, ErrorMessage: ctx.Err().Error()}, nil
	}
	return domain.SimulatorRunResult{RunID: "run-1"}, nil
}

func (c *localizationCoreSpy) ListLocalizationProjects(context.Context) ([]domain.LocalizationProjectSummary, error) {
	return []domain.LocalizationProjectSummary{}, nil
}
func (c *localizationCoreSpy) GetLocalizationProject(context.Context, string) (domain.LocalizationProject, error) {
	return domain.LocalizationProject{}, nil
}
func (c *localizationCoreSpy) ImportLocalizationProject(_ context.Context, request application.ImportLocalizationRequest) (domain.LocalizationProject, error) {
	c.importRequest = request
	return domain.LocalizationProject{ID: "local-project"}, nil
}
func (*localizationCoreSpy) ReplaceLocalizationSource(context.Context, string, string, []byte) (domain.LocalizationProject, error) {
	return domain.LocalizationProject{}, nil
}
func (c *localizationCoreSpy) UpdateLocalizationTranslation(_ context.Context, projectID, _, _ string) (domain.LocalizationTranslation, error) {
	c.updateCalls++
	c.projectID = projectID
	return domain.LocalizationTranslation{}, nil
}
func (c *localizationCoreSpy) ApproveLocalizationTranslation(context.Context, string, string) (domain.LocalizationTranslation, error) {
	return domain.LocalizationTranslation{}, nil
}
func (c *localizationCoreSpy) ImportLocalizationCSV(context.Context, string, []byte) (domain.LocalizationProject, error) {
	return domain.LocalizationProject{}, nil
}
func (c *localizationCoreSpy) ExportLocalizationCSV(context.Context, string) (domain.LocalizationArtifact, error) {
	return domain.LocalizationArtifact{}, nil
}
func (c *localizationCoreSpy) ExportLocalizationResources(context.Context, string) (domain.LocalizationArtifact, error) {
	return domain.LocalizationArtifact{}, nil
}
func (c *localizationCoreSpy) ValidateLocalizationProject(context.Context, string) (domain.LocalizationValidationReport, error) {
	return domain.LocalizationValidationReport{}, nil
}
func (c *localizationCoreSpy) BuildLocalizationPackage(context.Context, string) (domain.LocalizationPackage, error) {
	return domain.LocalizationPackage{}, nil
}

func (r *runtimeSpy) Ping(context.Context) error { return r.pingErr }
func (r *runtimeSpy) Close()                     { r.closed = true }
