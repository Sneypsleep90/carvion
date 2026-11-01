package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Sneypsleep90/carvion/internal/composition"
	"github.com/Sneypsleep90/carvion/internal/config"
	"github.com/Sneypsleep90/carvion/internal/core/application"
	"github.com/Sneypsleep90/carvion/internal/core/domain"
	wailsapp "github.com/wailsapp/wails/v3/pkg/application"
)

const (
	applicationVersion   = "0.1.0"
	statusReady          = "ready"
	statusDegraded       = "degraded"
	statusUnavailable    = "unavailable"
	statusUnknown        = "unknown"
	statusConnected      = "connected"
	statusNoAdapters     = "none"
	statusAdaptersLoaded = "available"
	statusCheckTimeout   = time.Second
)

var ErrCoreUnavailable = errors.New("CARVION Core is unavailable")

type coreService interface {
	ListAdapters(context.Context) ([]domain.AdapterInfo, error)
	ListManufacturers(context.Context) ([]domain.Manufacturer, error)
	ListVehicleModels(context.Context, string) ([]domain.VehicleModel, error)
	ListInfotainmentPlatforms(context.Context, string) ([]domain.InfotainmentPlatform, error)
	CheckCompatibility(context.Context, string, domain.CompatibilityRequest) (domain.CompatibilityResult, error)
	Detect(context.Context, string, domain.DetectRequest) (domain.DetectionResult, error)
	InspectResources(context.Context, string, domain.InspectResourcesRequest) (domain.ResourceInspection, error)
}

type localizationCore interface {
	ListLocalizationProjects(context.Context) ([]domain.LocalizationProjectSummary, error)
	GetLocalizationProject(context.Context, string) (domain.LocalizationProject, error)
	ImportLocalizationProject(context.Context, application.ImportLocalizationRequest) (domain.LocalizationProject, error)
	ReplaceLocalizationSource(context.Context, string, string, []byte) (domain.LocalizationProject, error)
	UpdateLocalizationTranslation(context.Context, string, string, string) (domain.LocalizationTranslation, error)
	ApproveLocalizationTranslation(context.Context, string, string) (domain.LocalizationTranslation, error)
	ImportLocalizationCSV(context.Context, string, []byte) (domain.LocalizationProject, error)
	ExportLocalizationCSV(context.Context, string) (domain.LocalizationArtifact, error)
	ExportLocalizationResources(context.Context, string) (domain.LocalizationArtifact, error)
	ValidateLocalizationProject(context.Context, string) (domain.LocalizationValidationReport, error)
	BuildLocalizationPackage(context.Context, string) (domain.LocalizationPackage, error)
}

type simulatorCore interface {
	ListSimulatorProfiles(context.Context) ([]domain.SimulatorProfile, error)
	RunSimulatorScenario(context.Context, string) (domain.SimulatorRunResult, error)
}

type geelyResearchCore interface {
	ListGeelyResearchEvidence(context.Context) ([]domain.GeelyResearchEvidence, error)
	AnalyzeGeelyProfile(context.Context, []byte) (domain.GeelyResearchAnalysis, error)
	InspectGeelyResources(context.Context, domain.GeelyResourceRequest) (domain.ResourceInspection, error)
	ImportGeelyLocalizationProject(context.Context, domain.GeelyLocalizationImportRequest) (domain.GeelyLocalizationImportResult, error)
	BuildGeelyDiagnosticReport(context.Context, domain.GeelyDiagnosticReportRequest) (domain.GeelyDiagnosticReport, error)
}

type managedRuntime interface {
	Ping(context.Context) error
	Close()
}

type runtimeStarter func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error)

// ApplicationState reports live local runtime health without implying a
// connection to a vehicle.
type ApplicationState struct {
	Version        string `json:"version"`
	CoreStatus     string `json:"coreStatus"`
	DatabaseStatus string `json:"databaseStatus"`
	AdapterStatus  string `json:"adapterStatus"`
	AdapterCount   int    `json:"adapterCount"`
	Message        string `json:"message"`
}

// ApplicationService is the typed Wails boundary over the transport-neutral
// Core. It owns local runtime lifecycle and passes the Wails context to Core.
type ApplicationService struct {
	mu sync.RWMutex

	ctx           context.Context
	core          coreService
	runtime       managedRuntime
	startupError  string
	loadConfig    func() (config.Config, error)
	startRuntime  runtimeStarter
	simulatorRuns map[string]context.CancelFunc
}

func NewApplicationService() *ApplicationService {
	return &ApplicationService{
		loadConfig: config.Load,
		startRuntime: func(ctx context.Context, cfg config.Config, logger *slog.Logger) (coreService, managedRuntime, error) {
			runtime, err := composition.NewLocal(ctx, cfg, logger)
			if err != nil {
				return nil, nil, err
			}
			return runtime.Core, runtime, nil
		},
	}
}

func (s *ApplicationService) ServiceStartup(ctx context.Context, _ wailsapp.ServiceOptions) error {
	if ctx == nil {
		return errors.New("Wails application context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.ctx = ctx
	loadConfig := s.loadConfig
	startRuntime := s.startRuntime
	s.mu.Unlock()

	if loadConfig == nil {
		loadConfig = config.Load
	}
	if startRuntime == nil {
		startRuntime = NewApplicationService().startRuntime
	}

	cfg, err := loadConfig()
	if err != nil {
		s.markUnavailable(err)
		return nil
	}
	core, runtime, err := startRuntime(ctx, cfg, slog.Default())
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.markUnavailable(err)
		return nil
	}
	if core == nil || runtime == nil {
		s.markUnavailable(errors.New("local runtime returned incomplete dependencies"))
		return nil
	}
	if err := ctx.Err(); err != nil {
		runtime.Close()
		return err
	}

	s.mu.Lock()
	s.core = core
	s.runtime = runtime
	s.startupError = ""
	s.mu.Unlock()
	return nil
}

func (s *ApplicationService) ServiceShutdown() error {
	s.mu.Lock()
	runtime := s.runtime
	simulatorRuns := s.simulatorRuns
	s.simulatorRuns = nil
	s.runtime = nil
	s.core = nil
	s.ctx = nil
	s.mu.Unlock()
	for _, cancel := range simulatorRuns {
		cancel()
	}

	if runtime != nil {
		runtime.Close()
	}
	return nil
}

func (s *ApplicationService) GetApplicationState() ApplicationState {
	ctx, core, runtime, startupError := s.snapshot()
	state := ApplicationState{
		Version:        applicationVersion,
		CoreStatus:     statusUnavailable,
		DatabaseStatus: statusUnknown,
		AdapterStatus:  statusUnknown,
	}
	if core == nil || runtime == nil {
		if startupError != "" {
			state.Message = startupError
			return state
		}
		state.CoreStatus = "starting"
		return state
	}

	state.CoreStatus = statusReady
	state.DatabaseStatus = "checking"
	state.AdapterStatus = "checking"
	pingCtx, cancel := context.WithTimeout(ctx, statusCheckTimeout)
	err := runtime.Ping(pingCtx)
	cancel()
	if err != nil {
		state.DatabaseStatus = statusUnavailable
		state.CoreStatus = statusDegraded
		state.Message = appendMessage(state.Message, fmt.Sprintf("Локальная PostgreSQL недоступна: %v", err))
	}
	if err == nil {
		state.DatabaseStatus = statusConnected
	}

	adapters, err := core.ListAdapters(ctx)
	if err != nil {
		state.AdapterStatus = statusUnavailable
		state.CoreStatus = statusDegraded
		state.Message = appendMessage(state.Message, fmt.Sprintf("Не удалось получить список адаптеров: %v", err))
		return state
	}
	state.AdapterCount = len(adapters)
	state.AdapterStatus = statusAdaptersLoaded
	if len(adapters) == 0 {
		state.AdapterStatus = statusNoAdapters
		state.Message = appendMessage(state.Message, "В доверенном локальном каталоге адаптеров ничего не найдено.")
	}
	return state
}

func (s *ApplicationService) ListAdapters() ([]domain.AdapterInfo, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListAdapters(ctx)
}

func (s *ApplicationService) ListManufacturers() ([]domain.Manufacturer, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListManufacturers(ctx)
}

func (s *ApplicationService) ListVehicleModels(manufacturerID string) ([]domain.VehicleModel, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListVehicleModels(ctx, manufacturerID)
}

func (s *ApplicationService) ListInfotainmentPlatforms(manufacturerID string) ([]domain.InfotainmentPlatform, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListInfotainmentPlatforms(ctx, manufacturerID)
}

func (s *ApplicationService) CheckCompatibility(adapterID string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return domain.CompatibilityResult{}, err
	}
	return core.CheckCompatibility(ctx, adapterID, request)
}

func (s *ApplicationService) DetectVehicle(adapterID string, identity domain.VehicleIdentity) (domain.DetectionResult, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return domain.DetectionResult{}, err
	}
	observation, err := json.Marshal(identity)
	if err != nil {
		return domain.DetectionResult{}, fmt.Errorf("encode vehicle observation: %w", err)
	}
	return core.Detect(ctx, adapterID, domain.DetectRequest{Observation: observation})
}

func (s *ApplicationService) InspectResources(adapterID string, request domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return domain.ResourceInspection{}, err
	}
	return core.InspectResources(ctx, adapterID, request)
}

func (s *ApplicationService) ListLocalizationProjects() ([]domain.LocalizationProjectSummary, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListLocalizationProjects(ctx)
}

func (s *ApplicationService) GetLocalizationProject(projectID string) (domain.LocalizationProject, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return core.GetLocalizationProject(ctx, projectID)
}

func (s *ApplicationService) ReplaceLocalizationSource(projectID, filename, content string) (domain.LocalizationProject, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return core.ReplaceLocalizationSource(ctx, projectID, filename, []byte(content))
}

func (s *ApplicationService) ImportLocalizationProject(name, locale, format, filename, content string) (domain.LocalizationProject, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return core.ImportLocalizationProject(ctx, application.ImportLocalizationRequest{
		Name: name, Locale: locale, Format: domain.LocalizationFormat(format), Filename: filename, Content: []byte(content),
	})
}

func (s *ApplicationService) UpdateLocalizationTranslation(projectID, resourceKey, targetText string) (domain.LocalizationTranslation, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationTranslation{}, err
	}
	return core.UpdateLocalizationTranslation(ctx, projectID, resourceKey, targetText)
}

func (s *ApplicationService) ApproveLocalizationTranslation(projectID, resourceKey string) (domain.LocalizationTranslation, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationTranslation{}, err
	}
	return core.ApproveLocalizationTranslation(ctx, projectID, resourceKey)
}

func (s *ApplicationService) ImportLocalizationCSV(projectID, content string) (domain.LocalizationProject, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationProject{}, err
	}
	return core.ImportLocalizationCSV(ctx, projectID, []byte(content))
}

func (s *ApplicationService) ExportLocalizationCSV(projectID string) (domain.LocalizationArtifact, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	return core.ExportLocalizationCSV(ctx, projectID)
}

func (s *ApplicationService) ExportLocalizationResources(projectID string) (domain.LocalizationArtifact, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationArtifact{}, err
	}
	return core.ExportLocalizationResources(ctx, projectID)
}

func (s *ApplicationService) ValidateLocalizationProject(projectID string) (domain.LocalizationValidationReport, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationValidationReport{}, err
	}
	return core.ValidateLocalizationProject(ctx, projectID)
}

func (s *ApplicationService) BuildLocalizationPackage(projectID string) (domain.LocalizationPackage, error) {
	ctx, core, err := s.localizationCoreSnapshot()
	if err != nil {
		return domain.LocalizationPackage{}, err
	}
	return core.BuildLocalizationPackage(ctx, projectID)
}

func (s *ApplicationService) ListSimulatorProfiles() ([]domain.SimulatorProfile, error) {
	ctx, core, err := s.simulatorCoreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListSimulatorProfiles(ctx)
}

func (s *ApplicationService) RunSimulatorScenario(profileID, requestID string) (domain.SimulatorRunResult, error) {
	if strings.TrimSpace(requestID) == "" || len(requestID) > 128 {
		return domain.SimulatorRunResult{}, fmt.Errorf("%w: simulator request ID must be between 1 and 128 bytes", domain.ErrInvalidArgument)
	}
	ctx, core, err := s.simulatorCoreSnapshot()
	if err != nil {
		return domain.SimulatorRunResult{}, err
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	if s.simulatorRuns == nil {
		s.simulatorRuns = make(map[string]context.CancelFunc)
	}
	if _, exists := s.simulatorRuns[requestID]; exists {
		s.mu.Unlock()
		cancel()
		return domain.SimulatorRunResult{}, fmt.Errorf("%w: simulator request ID is already active", domain.ErrInvalidArgument)
	}
	s.simulatorRuns[requestID] = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.simulatorRuns, requestID)
		s.mu.Unlock()
		cancel()
	}()
	return core.RunSimulatorScenario(runCtx, profileID)
}

func (s *ApplicationService) CancelSimulatorScenario(requestID string) bool {
	s.mu.RLock()
	cancel := s.simulatorRuns[requestID]
	s.mu.RUnlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (s *ApplicationService) ListGeelyResearchEvidence() ([]domain.GeelyResearchEvidence, error) {
	ctx, core, err := s.geelyResearchCoreSnapshot()
	if err != nil {
		return nil, err
	}
	return core.ListGeelyResearchEvidence(ctx)
}

func (s *ApplicationService) AnalyzeGeelyProfile(profileJSON string) (domain.GeelyResearchAnalysis, error) {
	ctx, core, err := s.geelyResearchCoreSnapshot()
	if err != nil {
		return domain.GeelyResearchAnalysis{}, err
	}
	return core.AnalyzeGeelyProfile(ctx, []byte(profileJSON))
}

func (s *ApplicationService) InspectGeelyResources(request domain.GeelyResourceRequest) (domain.ResourceInspection, error) {
	ctx, core, err := s.geelyResearchCoreSnapshot()
	if err != nil {
		return domain.ResourceInspection{}, err
	}
	return core.InspectGeelyResources(ctx, request)
}

func (s *ApplicationService) ImportGeelyLocalizationProject(request domain.GeelyLocalizationImportRequest) (domain.GeelyLocalizationImportResult, error) {
	ctx, core, err := s.geelyResearchCoreSnapshot()
	if err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	return core.ImportGeelyLocalizationProject(ctx, request)
}

func (s *ApplicationService) BuildGeelyDiagnosticReport(request domain.GeelyDiagnosticReportRequest) (domain.GeelyDiagnosticReport, error) {
	ctx, core, err := s.geelyResearchCoreSnapshot()
	if err != nil {
		return domain.GeelyDiagnosticReport{}, err
	}
	return core.BuildGeelyDiagnosticReport(ctx, request)
}

func (s *ApplicationService) coreSnapshot() (context.Context, coreService, error) {
	ctx, core, _, startupError := s.snapshot()
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if core == nil {
		detail := startupError
		if detail == "" {
			detail = "local runtime has not started"
		}
		return nil, nil, fmt.Errorf("%w: %s", ErrCoreUnavailable, detail)
	}
	return ctx, core, nil
}

func (s *ApplicationService) localizationCoreSnapshot() (context.Context, localizationCore, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, nil, err
	}
	localized, ok := core.(localizationCore)
	if !ok {
		return nil, nil, fmt.Errorf("%w: localization service is unavailable", ErrCoreUnavailable)
	}
	return ctx, localized, nil
}

func (s *ApplicationService) simulatorCoreSnapshot() (context.Context, simulatorCore, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, nil, err
	}
	simulator, ok := core.(simulatorCore)
	if !ok {
		return nil, nil, fmt.Errorf("%w: simulator service is unavailable", ErrCoreUnavailable)
	}
	return ctx, simulator, nil
}

func (s *ApplicationService) geelyResearchCoreSnapshot() (context.Context, geelyResearchCore, error) {
	ctx, core, err := s.coreSnapshot()
	if err != nil {
		return nil, nil, err
	}
	research, ok := core.(geelyResearchCore)
	if !ok {
		return nil, nil, fmt.Errorf("%w: Geely research service is unavailable", ErrCoreUnavailable)
	}
	return ctx, research, nil
}

func (s *ApplicationService) snapshot() (context.Context, coreService, managedRuntime, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx, s.core, s.runtime, s.startupError
}

func (s *ApplicationService) markUnavailable(err error) {
	s.mu.Lock()
	s.core = nil
	s.runtime = nil
	s.startupError = err.Error()
	s.mu.Unlock()
}

func appendMessage(current, next string) string {
	if current == "" {
		return next
	}
	return strings.Join([]string{current, next}, " ")
}
