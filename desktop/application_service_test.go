package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestGeelyResearchMethodsDelegateDirectlyToCoreWithWailsContext(t *testing.T) {
	wailsContext := context.WithValue(context.Background(), geelyContextKey{}, "wails-context")
	core := &geelyResearchCoreSpy{coreSpy: &coreSpy{}}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, &runtimeSpy{}, nil
		},
	}
	if err := service.ServiceStartup(wailsContext, wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}

	evidence, err := service.ListGeelyResearchEvidence()
	if err != nil || len(evidence) != 1 || evidence[0].ID != "evidence-1" {
		t.Fatalf("ListGeelyResearchEvidence() = %+v, %v", evidence, err)
	}
	analysis, err := service.AnalyzeGeelyProfile(`{"schema":"carvion.geely-research/v1"}`)
	if err != nil || analysis.Compatibility.Status != domain.StatusUnknown {
		t.Fatalf("AnalyzeGeelyProfile() = %+v, %v", analysis, err)
	}
	if core.context == nil || core.context.Value(geelyContextKey{}) != "wails-context" || core.profileJSON == "" {
		t.Fatalf("Core received context=%v profile=%q", core.context, core.profileJSON)
	}
	resource := domain.GeelyResourceRequest{Filename: "strings.xml", Format: domain.LocalizationFormatAndroidXML, Content: "<resources/>"}
	if _, err := service.InspectGeelyResources(resource); err != nil {
		t.Fatalf("InspectGeelyResources() error = %v", err)
	}
	if _, err := service.ImportGeelyLocalizationProject(domain.GeelyLocalizationImportRequest{}); err != nil {
		t.Fatalf("ImportGeelyLocalizationProject() error = %v", err)
	}
	if _, err := service.BuildGeelyDiagnosticReport(domain.GeelyDiagnosticReportRequest{}); err != nil {
		t.Fatalf("BuildGeelyDiagnosticReport() error = %v", err)
	}
	if core.evidenceCalls != 1 || core.analyzeCalls != 1 || core.inspectCalls != 1 || core.importCalls != 1 || core.reportCalls != 1 {
		t.Fatalf("Geely Core calls evidence=%d analyze=%d inspect=%d import=%d report=%d", core.evidenceCalls, core.analyzeCalls, core.inspectCalls, core.importCalls, core.reportCalls)
	}
}

func TestGeelyResearchMethodsReportUnavailableCore(t *testing.T) {
	service := &ApplicationService{}
	if _, err := service.ListGeelyResearchEvidence(); !errors.Is(err, ErrCoreUnavailable) {
		t.Fatalf("ListGeelyResearchEvidence() error = %v, want ErrCoreUnavailable", err)
	}
	if _, err := service.AnalyzeGeelyProfile("{}"); !errors.Is(err, ErrCoreUnavailable) {
		t.Fatalf("AnalyzeGeelyProfile() error = %v, want ErrCoreUnavailable", err)
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

func TestDeploymentFacadeStartsAsyncPersistsEventsAndDelegatesQueries(t *testing.T) {
	wailsContext := context.WithValue(context.Background(), deploymentContextKey{}, "wails-context")
	core := newDeploymentCoreSpy()
	core.waitForCancel = true
	core.requireContext = true
	runtime := &runtimeSpy{}
	events := make(chan domain.DeploymentEvent, 4)
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, runtime, nil
		},
		emitDeploymentProgress: func(event domain.DeploymentEvent) error {
			events <- event
			return errors.New("live listener unavailable")
		},
	}
	if err := service.ServiceStartup(wailsContext, wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}

	request := domain.DeploymentRequest{ProjectID: "project-1", TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest}
	job, err := service.StartDeploymentJob(request)
	if err != nil || job.ID != "deployment-1" || job.State != domain.DeploymentStateQueued {
		t.Fatalf("StartDeploymentJob() = %+v, %v", job, err)
	}
	<-core.started
	select {
	case event := <-events:
		if event.JobID != job.ID || event.Sequence != 2 {
			t.Fatalf("deployment event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("Wails event publisher did not receive persisted progress")
	}
	if core.runCalls != 1 || core.startCalls != 1 || core.runContext == nil || core.runContext.Value(deploymentContextKey{}) != "wails-context" {
		t.Fatalf("Core start=%d run=%d context=%v", core.startCalls, core.runCalls, core.runContext)
	}
	if _, err := service.GetDeploymentJob(job.ID); err != nil {
		t.Fatalf("GetDeploymentJob() error = %v", err)
	}
	if summaries, err := service.ListDeploymentJobs(20); err != nil || len(summaries) != 1 {
		t.Fatalf("ListDeploymentJobs() = %+v, %v", summaries, err)
	}
	if _, err := service.CancelDeploymentJob(job.ID); err != nil {
		t.Fatalf("CancelDeploymentJob() error = %v", err)
	}
	select {
	case <-core.runFinished:
	case <-time.After(time.Second):
		t.Fatal("CancelDeploymentJob() did not cancel the active Core run")
	}
	if core.cancelCalls != 1 {
		t.Fatalf("Core cancel calls = %d, want 1", core.cancelCalls)
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentFacadeShutdownCancelsRunsBeforeClosingRuntime(t *testing.T) {
	core := newDeploymentCoreSpy()
	core.waitForCancel = true
	runtime := &runtimeSpy{}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, runtime, nil
		},
	}
	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartDeploymentJob(domain.DeploymentRequest{ProjectID: "project-1", TargetProfileID: "demo-profile", Mode: domain.DeploymentModeTest}); err != nil {
		t.Fatal(err)
	}
	<-core.started
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-core.runFinished:
	default:
		t.Fatal("ServiceShutdown() closed runtime before the deployment worker stopped")
	}
	if !runtime.closed {
		t.Fatal("ServiceShutdown() did not close runtime after deployment cancellation")
	}
}

func TestDeploymentFacadeRejectsDuplicateActiveJobID(t *testing.T) {
	core := newDeploymentCoreSpy()
	core.waitForCancel = true
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, &runtimeSpy{}, nil
		},
	}
	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	request := domain.DeploymentRequest{ProjectID: "project-1", TargetProfileID: "demo-profile", Mode: domain.DeploymentModeTest}
	if _, err := service.StartDeploymentJob(request); err != nil {
		t.Fatal(err)
	}
	<-core.started
	if _, err := service.StartDeploymentJob(request); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("duplicate StartDeploymentJob() error = %v, want invalid argument", err)
	}
	if core.runCalls != 1 {
		t.Fatalf("duplicate job started %d worker runs, want 1", core.runCalls)
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentFacadeReportsUnavailableCore(t *testing.T) {
	service := &ApplicationService{}
	if _, err := service.StartDeploymentJob(domain.DeploymentRequest{}); !errors.Is(err, ErrCoreUnavailable) {
		t.Fatalf("StartDeploymentJob() error = %v, want ErrCoreUnavailable", err)
	}
	if _, err := service.GetDeploymentJob("job-1"); !errors.Is(err, ErrCoreUnavailable) {
		t.Fatalf("GetDeploymentJob() error = %v, want ErrCoreUnavailable", err)
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

type geelyContextKey struct{}

type deploymentContextKey struct{}

type geelyResearchCoreSpy struct {
	*coreSpy
	evidenceCalls int
	analyzeCalls  int
	inspectCalls  int
	importCalls   int
	reportCalls   int
	context       context.Context
	profileJSON   string
}

type deploymentCoreSpy struct {
	*coreSpy
	mu             sync.Mutex
	startCalls     int
	runCalls       int
	cancelCalls    int
	request        domain.DeploymentRequest
	jobID          string
	runContext     context.Context
	waitForCancel  bool
	requireContext bool
	started        chan struct{}
	runFinished    chan struct{}
}

func newDeploymentCoreSpy() *deploymentCoreSpy {
	return &deploymentCoreSpy{coreSpy: &coreSpy{}, started: make(chan struct{}), runFinished: make(chan struct{})}
}

func (c *deploymentCoreSpy) StartDeploymentJob(ctx context.Context, request domain.DeploymentRequest) (domain.DeploymentJob, error) {
	c.mu.Lock()
	c.startCalls++
	c.request = request
	c.mu.Unlock()
	if c.requireContext && ctx.Value(deploymentContextKey{}) == nil {
		return domain.DeploymentJob{}, errors.New("Wails context was not propagated")
	}
	return domain.DeploymentJob{ID: "deployment-1", ProjectID: request.ProjectID, Mode: request.Mode, State: domain.DeploymentStateQueued,
		Events: []domain.DeploymentEvent{{JobID: "deployment-1", Sequence: 1, State: domain.DeploymentStateQueued, Stage: "queued", Message: "queued"}}}, nil
}

func (c *deploymentCoreSpy) RunDeploymentJob(ctx context.Context, id string, publish application.DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	c.mu.Lock()
	c.runCalls++
	c.jobID, c.runContext = id, ctx
	c.mu.Unlock()
	close(c.started)
	defer close(c.runFinished)
	_ = publish(domain.DeploymentEvent{JobID: id, Sequence: 2, State: domain.DeploymentStatePreflighting, Stage: "preflight.started", Message: "progress"})
	if c.waitForCancel {
		<-ctx.Done()
		return domain.DeploymentJob{ID: id, State: domain.DeploymentStateCancelled}, nil
	}
	return domain.DeploymentJob{ID: id, State: domain.DeploymentStateSimulated}, nil
}

func (c *deploymentCoreSpy) CancelDeploymentJob(_ context.Context, id string) (domain.DeploymentJob, error) {
	c.mu.Lock()
	c.cancelCalls++
	c.mu.Unlock()
	return domain.DeploymentJob{ID: id, State: domain.DeploymentStateSimulating}, nil
}

func (c *deploymentCoreSpy) GetDeploymentJob(_ context.Context, id string) (domain.DeploymentJob, error) {
	return domain.DeploymentJob{ID: id, State: domain.DeploymentStateSimulating}, nil
}

func (c *deploymentCoreSpy) ListDeploymentJobs(context.Context, int) ([]domain.DeploymentJobSummary, error) {
	return []domain.DeploymentJobSummary{{ID: "deployment-1"}}, nil
}

func (c *geelyResearchCoreSpy) ListGeelyResearchEvidence(ctx context.Context) ([]domain.GeelyResearchEvidence, error) {
	c.evidenceCalls++
	c.context = ctx
	return []domain.GeelyResearchEvidence{{ID: "evidence-1"}}, nil
}

func (c *geelyResearchCoreSpy) AnalyzeGeelyProfile(ctx context.Context, profileJSON []byte) (domain.GeelyResearchAnalysis, error) {
	c.analyzeCalls++
	c.context = ctx
	c.profileJSON = string(profileJSON)
	return domain.GeelyResearchAnalysis{Compatibility: domain.CompatibilityResult{Status: domain.StatusUnknown}}, nil
}

func (c *geelyResearchCoreSpy) InspectGeelyResources(ctx context.Context, _ domain.GeelyResourceRequest) (domain.ResourceInspection, error) {
	c.inspectCalls++
	c.context = ctx
	return domain.ResourceInspection{}, nil
}

func (c *geelyResearchCoreSpy) ImportGeelyLocalizationProject(ctx context.Context, _ domain.GeelyLocalizationImportRequest) (domain.GeelyLocalizationImportResult, error) {
	c.importCalls++
	c.context = ctx
	return domain.GeelyLocalizationImportResult{}, nil
}

func (c *geelyResearchCoreSpy) BuildGeelyDiagnosticReport(ctx context.Context, _ domain.GeelyDiagnosticReportRequest) (domain.GeelyDiagnosticReport, error) {
	c.reportCalls++
	c.context = ctx
	return domain.GeelyDiagnosticReport{}, nil
}

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
