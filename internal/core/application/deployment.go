package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/simulator"
)

const (
	productionDeploymentEnabled = false
	deploymentRunTimeout        = 2 * time.Minute
	deploymentCleanupTimeout    = 5 * time.Second
	maxDeploymentPackageBytes   = 16 << 20
)

type DeploymentService struct {
	localization *LocalizationService
	repository   DeploymentRepository
	verifier     DeploymentPackageVerifier
	provider     DeploymentProvider
	runTimeout   time.Duration

	activeMu sync.Mutex
	active   map[string]context.CancelFunc
}

func NewServiceWithDeployment(
	adapters AdapterGateway,
	catalog CatalogRepository,
	localizationRepository LocalizationRepository,
	deploymentRepository DeploymentRepository,
	verifier DeploymentPackageVerifier,
	provider DeploymentProvider,
) (*Service, error) {
	service, err := NewServiceWithLocalization(adapters, catalog, localizationRepository)
	if err != nil {
		return nil, err
	}
	if deploymentRepository == nil {
		return nil, errorsMissingDependency("deployment repository")
	}
	if verifier == nil {
		return nil, errorsMissingDependency("deployment package verifier")
	}
	if provider == nil {
		return nil, errorsMissingDependency("synthetic deployment provider")
	}
	service.deployment = &DeploymentService{
		localization: service.localization,
		repository:   deploymentRepository,
		verifier:     verifier,
		provider:     provider,
		runTimeout:   deploymentRunTimeout,
		active:       make(map[string]context.CancelFunc),
	}
	return service, nil
}

func (s *Service) StartDeploymentJob(ctx context.Context, request domain.DeploymentRequest) (domain.DeploymentJob, error) {
	service, err := s.deploymentService()
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return service.StartJob(ctx, request)
}

func (s *Service) RunDeploymentJob(ctx context.Context, jobID string, publish DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	service, err := s.deploymentService()
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return service.RunJob(ctx, jobID, publish)
}

func (s *Service) CancelDeploymentJob(ctx context.Context, jobID string) (domain.DeploymentJob, error) {
	service, err := s.deploymentService()
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return service.CancelJob(ctx, jobID)
}

func (s *Service) GetDeploymentJob(ctx context.Context, jobID string) (domain.DeploymentJob, error) {
	service, err := s.deploymentService()
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	if err := contextError(ctx); err != nil {
		return domain.DeploymentJob{}, err
	}
	if err := validateField("deployment_job_id", jobID); err != nil {
		return domain.DeploymentJob{}, err
	}
	return service.repository.GetDeploymentJob(ctx, jobID)
}

func (s *Service) ListDeploymentJobs(ctx context.Context, limit int) ([]domain.DeploymentJobSummary, error) {
	service, err := s.deploymentService()
	if err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if limit == 0 {
		limit = 50
	}
	return service.repository.ListDeploymentJobs(ctx, limit)
}

func (s *Service) deploymentService() (*DeploymentService, error) {
	if s == nil || s.deployment == nil {
		return nil, fmt.Errorf("%w: deployment engine is not configured", domain.ErrDeploymentUnavailable)
	}
	return s.deployment, nil
}

func (s *DeploymentService) StartJob(ctx context.Context, request domain.DeploymentRequest) (domain.DeploymentJob, error) {
	if err := contextError(ctx); err != nil {
		return domain.DeploymentJob{}, err
	}
	if err := validateField("project_id", request.ProjectID); err != nil {
		return domain.DeploymentJob{}, err
	}
	switch request.Mode {
	case domain.DeploymentModeTest:
		if err := validateField("target_profile_id", request.TargetProfileID); err != nil {
			return domain.DeploymentJob{}, err
		}
	case domain.DeploymentModeProduction:
		// Production requests are retained for an auditable, blocked preflight.
	default:
		return domain.DeploymentJob{}, invalidArgument("deployment mode must be test or production")
	}
	if request.TargetProfileID != "" {
		if err := validateField("target_profile_id", request.TargetProfileID); err != nil {
			return domain.DeploymentJob{}, err
		}
	}
	packageItem, err := s.localization.BuildLanguagePackage(ctx, request.ProjectID)
	if err != nil {
		return domain.DeploymentJob{}, fmt.Errorf("build CARVION language package for deployment: %w", err)
	}
	if packageItem.Size < 1 || packageItem.Size > maxDeploymentPackageBytes || !isDeploymentSHA256(packageItem.SHA256) {
		return domain.DeploymentJob{}, invalidArgument("generated language package has invalid size or SHA-256")
	}
	providerID := ""
	if request.Mode == domain.DeploymentModeTest {
		providerID = s.provider.ID()
	}
	return s.repository.CreateDeploymentJob(ctx, domain.DeploymentJob{
		ProjectID: request.ProjectID, TargetProfileID: request.TargetProfileID,
		Mode: request.Mode, ProviderID: providerID, PackageSHA256: packageItem.SHA256,
		State: domain.DeploymentStateQueued, Preflight: domain.DeploymentPreflight{Checks: []domain.DeploymentCheck{}},
	})
}

func (s *DeploymentService) RunJob(ctx context.Context, jobID string, publish DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	if err := contextError(ctx); err != nil {
		return domain.DeploymentJob{}, err
	}
	if err := validateField("deployment_job_id", jobID); err != nil {
		return domain.DeploymentJob{}, err
	}
	job, err := s.repository.GetDeploymentJob(ctx, jobID)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	if job.State != domain.DeploymentStateQueued {
		return domain.DeploymentJob{}, fmt.Errorf("%w: deployment job is not queued", domain.ErrDeploymentStateConflict)
	}
	runTimeout := s.runTimeout
	if runTimeout <= 0 {
		runTimeout = deploymentRunTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	if err := s.registerActive(jobID, cancel); err != nil {
		cancel()
		return domain.DeploymentJob{}, err
	}
	defer func() {
		cancel()
		s.unregisterActive(jobID)
	}()
	result, runErr := s.runJob(runCtx, job, publish)
	if runCtx.Err() == nil {
		return result, runErr
	}
	latest, getErr := s.getFinalJob(jobID)
	if getErr != nil {
		return result, errors.Join(runErr, getErr)
	}
	if latest.State.Terminal() {
		return latest, nil
	}
	if errors.Is(runCtx.Err(), context.Canceled) {
		return s.persistCancellation(latest, publish)
	}
	return s.failJob(runCtx, latest, latest.Preflight, "Deployment exceeded its time limit.", publish)
}

func (s *DeploymentService) runJob(ctx context.Context, job domain.DeploymentJob, publish DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	if ctx.Err() != nil {
		return s.persistCancellation(job, publish)
	}
	var err error
	job, err = s.transition(ctx, job, domain.DeploymentStatePreflighting, "preflight.started", "Deployment preflight started.", publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	artifact, err := s.localization.GetPackageContent(ctx, job.ProjectID, job.PackageSHA256)
	if err != nil {
		if ctx.Err() != nil {
			return s.persistCancellation(job, publish)
		}
		return s.failJob(ctx, job, domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{{
			ID: "package-integrity", Status: domain.DeploymentCheckFailed, Required: true, Message: "Stored CARVION package could not be loaded.",
		}}}, "Deployment preflight failed because the stored package is unavailable.", publish)
	}
	if err := s.verifier.VerifyLanguagePackage(artifact, job.PackageSHA256); err != nil {
		preflight := domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{{
			ID: "package-integrity", Status: domain.DeploymentCheckFailed, Required: true, Message: "CARVION package integrity verification failed.",
		}}}
		return s.failJob(ctx, job, preflight, "Deployment preflight failed package integrity verification.", publish)
	}
	packageCheck := domain.DeploymentCheck{ID: "package-integrity", Status: domain.DeploymentCheckPassed, Required: true, Message: "CARVION package structure and SHA-256 integrity verified."}
	if job.Mode == domain.DeploymentModeProduction {
		if productionDeploymentEnabled {
			return s.failJob(ctx, job, domain.DeploymentPreflight{Checks: []domain.DeploymentCheck{packageCheck}}, "Production deployment policy is not configured.", publish)
		}
		job.Preflight = productionBlockedPreflight(packageCheck)
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "Production Mode is blocked because required platform and safety evidence is absent.", publish)
	}
	profile, profileErr := simulator.LoadProfile(job.TargetProfileID)
	if profileErr != nil {
		job.Preflight = domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{
			packageCheck,
			{ID: "target-profile", Status: domain.DeploymentCheckBlocked, Required: true, Message: "Selected target is not in the embedded synthetic profile catalog."},
			{ID: "compatibility", Status: domain.DeploymentCheckUnknown, Required: true, Message: "Compatibility is unknown for the selected target."},
		}}
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "The selected target is not a known synthetic test profile.", publish)
	}
	if !profile.Public.Synthetic {
		job.Preflight = domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{
			packageCheck,
			{ID: "target-profile", Status: domain.DeploymentCheckBlocked, Required: true, Message: "Only explicitly synthetic profiles are accepted in Test Mode."},
			{ID: "compatibility", Status: domain.DeploymentCheckUnknown, Required: true, Message: "Compatibility is unknown for this profile."},
		}}
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "The selected target is not marked synthetic.", publish)
	}
	if profile.Public.ExpectedCompatibility != domain.StatusSupported {
		compatibilityStatus := domain.DeploymentCheckBlocked
		compatibilityMessage := "The selected synthetic profile is explicitly unsupported."
		if profile.Public.ExpectedCompatibility == domain.StatusUnknown {
			compatibilityStatus = domain.DeploymentCheckUnknown
			compatibilityMessage = "The exact synthetic firmware build is unknown."
		}
		job.Preflight = domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{
			packageCheck,
			{ID: "target-profile", Status: domain.DeploymentCheckPassed, Required: true, Message: "Target is present in the embedded synthetic profile catalog."},
			{ID: "compatibility", Status: compatibilityStatus, Required: true, Message: compatibilityMessage},
		}}
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "The exact target profile is not supported by the synthetic compatibility catalog.", publish)
	}
	if err := ctx.Err(); err != nil {
		return s.persistCancellation(job, publish)
	}
	if job.ProviderID != s.provider.ID() {
		job.Preflight = domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{
			packageCheck,
			{ID: "target-profile", Status: domain.DeploymentCheckPassed, Required: true, Message: "Target is synthetic."},
			{ID: "compatibility", Status: domain.DeploymentCheckPassed, Required: true, Message: "Exact synthetic profile is supported."},
			{ID: "provider-capability", Status: domain.DeploymentCheckBlocked, Required: true, Message: "No registered synthetic provider matches this job."},
		}}
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "No registered synthetic provider matches this job.", publish)
	}
	capabilities := s.provider.Capabilities()
	checks := []domain.DeploymentCheck{
		packageCheck,
		{ID: "target-profile", Status: domain.DeploymentCheckPassed, Required: true, Message: "Target is present in the embedded synthetic profile catalog."},
		{ID: "compatibility", Status: domain.DeploymentCheckPassed, Required: true, Message: "Exact synthetic platform, hardware revision and firmware build are supported."},
	}
	providerReady := capabilities.Mode == domain.DeploymentProviderModeSimulation && capabilities.CanSimulate && !capabilities.CanInstall
	providerStatus, providerMessage := domain.DeploymentCheckBlocked, "Registered provider is not limited to simulation."
	if providerReady {
		providerStatus, providerMessage = domain.DeploymentCheckPassed, "Provider is simulation-only and has no installation capability."
	}
	checks = append(checks,
		domain.DeploymentCheck{ID: "provider-capability", Status: providerStatus, Required: true, Message: providerMessage},
	)
	verificationStatus, verificationMessage := capabilityCheck(capabilities.CanVerifyResult, "Provider declares separate synthetic result verification.")
	recoveryStatus, recoveryMessage := capabilityCheck(capabilities.CanCheckRecovery, "Provider declares a synthetic recovery capability check.")
	checks = append(checks,
		domain.DeploymentCheck{ID: "result-verification", Status: verificationStatus, Required: true, Message: verificationMessage},
		domain.DeploymentCheck{ID: "recovery-capability", Status: recoveryStatus, Required: true, Message: recoveryMessage},
		domain.DeploymentCheck{ID: "bench-evidence", Status: domain.DeploymentCheckUnknown, Required: false, Message: "No physical bench evidence exists; it is not required for synthetic testing."},
	)
	if !providerReady || !capabilities.CanVerifyResult || !capabilities.CanCheckRecovery {
		job.Preflight = domain.DeploymentPreflight{Ready: false, Checks: checks}
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "The registered provider does not meet the simulation-only capability contract.", publish)
	}
	recovery, err := s.provider.CheckRecovery(ctx, job)
	if err != nil {
		checks = append(checks, domain.DeploymentCheck{ID: "recovery-check", Status: domain.DeploymentCheckFailed, Required: true, Message: "Synthetic recovery capability check failed."})
		job.Preflight = domain.DeploymentPreflight{Ready: false, Checks: checks}
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "Synthetic recovery capability could not be checked.", publish)
	}
	job.Recovery = recovery
	recoveryCheckStatus := domain.DeploymentCheckPassed
	recoveryCheckMessage := "Synthetic recovery capability check passed; no rollback was performed."
	if !recovery.Available || !recovery.Simulated {
		recoveryCheckStatus = domain.DeploymentCheckBlocked
		recoveryCheckMessage = "Provider did not confirm an explicitly simulated recovery capability."
	}
	checks = append(checks, domain.DeploymentCheck{ID: "recovery-check", Status: recoveryCheckStatus, Required: true, Message: recoveryCheckMessage})
	job.Preflight = domain.DeploymentPreflight{Ready: recoveryCheckStatus == domain.DeploymentCheckPassed, Checks: checks}
	if !job.Preflight.Ready {
		return s.transition(ctx, job, domain.DeploymentStateBlocked, "preflight.blocked", "Synthetic recovery preflight did not pass.", publish)
	}
	job, err = s.transition(ctx, job, domain.DeploymentStatePreflightPassed, "preflight.passed", "All required synthetic preflight checks passed.", publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	job, err = s.transition(ctx, job, domain.DeploymentStateSimulating, "simulation.started", "Synthetic simulation started; no vehicle will be changed.", publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	progress := func(event domain.DeploymentEvent) error {
		if event.JobID != job.ID || (event.State != "" && event.State != domain.DeploymentStateSimulating) || event.Sequence != 0 {
			return fmt.Errorf("%w: provider progress identity is invalid", domain.ErrInvalidArgument)
		}
		event.Stage = sanitizeDeploymentStage(event.Stage)
		event.Message = sanitizeDeploymentMessage(event.Message)
		persisted, appendErr := s.repository.AppendDeploymentEvent(ctx, job.ID, domain.DeploymentStateSimulating, event)
		if appendErr != nil {
			return appendErr
		}
		job.Events = append(job.Events, persisted)
		publishDeploymentEvent(publish, persisted)
		return nil
	}
	if err := s.provider.Simulate(ctx, job, artifact, progress); err != nil {
		if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
			return s.persistCancellation(job, publish)
		}
		recoveryCtx, cancel := context.WithTimeout(context.Background(), deploymentCleanupTimeout)
		recovery, recoveryErr := s.provider.CheckRecovery(recoveryCtx, job)
		cancel()
		if recoveryErr == nil {
			job.Recovery = recovery
		}
		return s.failJob(ctx, job, job.Preflight, "Synthetic provider failed before completing the simulation.", publish)
	}
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return s.persistCancellation(job, publish)
		}
		return s.failJob(ctx, job, job.Preflight, "Synthetic simulation exceeded its time limit.", publish)
	}
	job, err = s.transition(ctx, job, domain.DeploymentStateVerifying, "result-verification.started", "Synthetic result verification started.", publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	verification, err := s.provider.VerifyResult(ctx, job, job.PackageSHA256)
	if err != nil || !verification.Verified || !verification.Simulated {
		if err == nil {
			err = errors.New("provider did not confirm a simulated result")
		}
		recoveryCtx, cancel := context.WithTimeout(context.Background(), deploymentCleanupTimeout)
		recovery, recoveryErr := s.provider.CheckRecovery(recoveryCtx, job)
		cancel()
		if recoveryErr == nil {
			job.Recovery = recovery
		}
		return s.failJob(ctx, job, job.Preflight, "Synthetic result verification failed.", publish)
	}
	job.ResultVerification = verification
	job, err = s.transition(ctx, job, domain.DeploymentStateSimulated, "simulation.completed", "Simulation and synthetic result checks passed; no vehicle was changed.", publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return s.getFinalJob(job.ID)
}

func productionBlockedPreflight(packageCheck domain.DeploymentCheck) domain.DeploymentPreflight {
	return domain.DeploymentPreflight{Ready: false, Checks: []domain.DeploymentCheck{
		packageCheck,
		{ID: "target-profile", Status: domain.DeploymentCheckBlocked, Required: true, Message: "No reviewed production platform profile is available."},
		{ID: "compatibility", Status: domain.DeploymentCheckUnknown, Required: true, Message: "No exact reviewed production compatibility record exists."},
		{ID: "installation-capability", Status: domain.DeploymentCheckBlocked, Required: true, Message: "No documented and authorized production installation method is available."},
		{ID: "result-verification", Status: domain.DeploymentCheckBlocked, Required: true, Message: "Production result verification criteria and capability are not established."},
		{ID: "recovery-capability", Status: domain.DeploymentCheckBlocked, Required: true, Message: "No confirmed production recovery procedure or capability exists."},
		{ID: "bench-evidence", Status: domain.DeploymentCheckBlocked, Required: true, Message: "No hardware bench-test evidence or safety review exists."},
		{ID: "production-policy", Status: domain.DeploymentCheckBlocked, Required: true, Message: "Production Mode is disabled by Core policy."},
	}}
}

func capabilityCheck(available bool, successMessage string) (domain.DeploymentCheckStatus, string) {
	if available {
		return domain.DeploymentCheckPassed, successMessage
	}
	return domain.DeploymentCheckBlocked, "Required provider capability is unavailable."
}

func (s *DeploymentService) transition(ctx context.Context, job domain.DeploymentJob, next domain.DeploymentState, stage, message string, publish DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	if err := domain.ValidateDeploymentTransition(job.State, next); err != nil {
		return domain.DeploymentJob{}, fmt.Errorf("%w: %v", domain.ErrDeploymentStateConflict, err)
	}
	updated := job
	updated.State = next
	event := domain.DeploymentEvent{JobID: job.ID, At: time.Now().UTC(), State: next, Stage: stage, Message: message}
	if err := s.repository.TransitionDeploymentJob(ctx, updated, event); err != nil {
		return domain.DeploymentJob{}, err
	}
	event.Sequence = int64(len(job.Events) + 1)
	updated.Events = append(append([]domain.DeploymentEvent(nil), job.Events...), event)
	updated.UpdatedAt = event.At
	publishDeploymentEvent(publish, event)
	return updated, nil
}

func (s *DeploymentService) failJob(ctx context.Context, job domain.DeploymentJob, preflight domain.DeploymentPreflight, message string, publish DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
		return s.persistCancellation(job, publish)
	}
	job.Preflight = preflight
	job.ErrorMessage = message
	cleanupCtx, cancel := context.WithTimeout(context.Background(), deploymentCleanupTimeout)
	defer cancel()
	failed, err := s.transition(cleanupCtx, job, domain.DeploymentStateFailed, "deployment.failed", message, publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return s.getFinalJob(failed.ID)
}

func (s *DeploymentService) persistCancellation(job domain.DeploymentJob, publish DeploymentProgressPublisher) (domain.DeploymentJob, error) {
	if job.State.Terminal() {
		return s.getFinalJob(job.ID)
	}
	job.ErrorMessage = "Deployment simulation was cancelled; no vehicle was changed."
	cleanupCtx, cancel := context.WithTimeout(context.Background(), deploymentCleanupTimeout)
	defer cancel()
	cancelled, err := s.transition(cleanupCtx, job, domain.DeploymentStateCancelled, "deployment.cancelled", job.ErrorMessage, publish)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	return s.getFinalJob(cancelled.ID)
}

func (s *DeploymentService) getFinalJob(jobID string) (domain.DeploymentJob, error) {
	ctx, cancel := context.WithTimeout(context.Background(), deploymentCleanupTimeout)
	defer cancel()
	return s.repository.GetDeploymentJob(ctx, jobID)
}

func (s *DeploymentService) CancelJob(ctx context.Context, jobID string) (domain.DeploymentJob, error) {
	if err := contextError(ctx); err != nil {
		return domain.DeploymentJob{}, err
	}
	if err := validateField("deployment_job_id", jobID); err != nil {
		return domain.DeploymentJob{}, err
	}
	job, err := s.repository.GetDeploymentJob(ctx, jobID)
	if err != nil {
		return domain.DeploymentJob{}, err
	}
	if job.State.Terminal() {
		return job, nil
	}
	s.activeMu.Lock()
	cancel := s.active[jobID]
	s.activeMu.Unlock()
	if cancel != nil {
		cancel()
		return job, nil
	}
	if job.State == domain.DeploymentStateQueued {
		job.ErrorMessage = "Deployment job was cancelled before execution."
		return s.transition(ctx, job, domain.DeploymentStateCancelled, "deployment.cancelled", job.ErrorMessage, nil)
	}
	return domain.DeploymentJob{}, fmt.Errorf("%w: job is not active in this process", domain.ErrDeploymentStateConflict)
}

func (s *DeploymentService) registerActive(jobID string, cancel context.CancelFunc) error {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	if _, exists := s.active[jobID]; exists {
		return fmt.Errorf("%w: deployment job is already running", domain.ErrDeploymentStateConflict)
	}
	s.active[jobID] = cancel
	return nil
}

func (s *DeploymentService) unregisterActive(jobID string) {
	s.activeMu.Lock()
	delete(s.active, jobID)
	s.activeMu.Unlock()
}

func publishDeploymentEvent(publish DeploymentProgressPublisher, event domain.DeploymentEvent) {
	if publish != nil {
		_ = publish(event)
	}
}

func sanitizeDeploymentStage(stage string) string {
	if stage == "" || len(stage) > 80 {
		return "provider.progress"
	}
	for _, char := range stage {
		if !(char >= 'a' && char <= 'z') && !(char >= '0' && char <= '9') && char != '.' && char != '_' && char != '-' {
			return "provider.progress"
		}
	}
	return stage
}

func sanitizeDeploymentMessage(message string) string {
	message = strings.Map(func(char rune) rune {
		if unicode.IsControl(char) {
			return ' '
		}
		return char
	}, message)
	message = strings.TrimSpace(message)
	if message == "" {
		return "Synthetic provider progress."
	}
	runes := []rune(message)
	if len(runes) > 512 {
		message = string(runes[:512])
	}
	return message
}

func isDeploymentSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9') && !(char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}
