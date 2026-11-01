package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestRunDeploymentJobSimulatesOnlyAfterPersistedPreflight(t *testing.T) {
	core, deploymentRepository, provider, verifier, projectID := newDeploymentTestCore(t, nil)
	job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
		ProjectID: projectID, TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest,
	})
	if err != nil {
		t.Fatalf("StartDeploymentJob() error = %v", err)
	}
	var published []domain.DeploymentEvent
	result, err := core.RunDeploymentJob(context.Background(), job.ID, func(event domain.DeploymentEvent) error {
		persisted, getErr := deploymentRepository.GetDeploymentJob(context.Background(), job.ID)
		if getErr != nil {
			return getErr
		}
		if len(persisted.Events) == 0 || persisted.Events[len(persisted.Events)-1].Sequence != event.Sequence {
			return errors.New("progress was published before it was persisted")
		}
		published = append(published, event)
		return nil
	})
	if err != nil {
		t.Fatalf("RunDeploymentJob() error = %v", err)
	}
	if result.State != domain.DeploymentStateSimulated || !result.Preflight.Ready || !result.ResultVerification.Verified || !result.ResultVerification.Simulated {
		t.Fatalf("deployment result = %+v, want verified synthetic completion", result)
	}
	if result.Recovery.Available != true || !result.Recovery.Simulated {
		t.Fatalf("recovery result = %+v, want simulated capability check", result.Recovery)
	}
	if !verifier.called || provider.simulateCalls != 1 || provider.verifyCalls != 1 || provider.recoveryCalls != 1 {
		t.Fatalf("calls: verifier=%t provider simulate=%d verify=%d recovery=%d", verifier.called, provider.simulateCalls, provider.verifyCalls, provider.recoveryCalls)
	}
	if len(result.Events) != len(published)+1 || len(result.Events) < 6 {
		t.Fatalf("persisted/published events = %d/%d, want initial queued record plus published lifecycle", len(result.Events), len(published))
	}
	for i, event := range result.Events {
		if event.Sequence != int64(i+1) {
			t.Fatalf("event sequence %d = %d", i, event.Sequence)
		}
	}
}

func TestProductionDeploymentIsPersistedBlockedAndNeverCallsProvider(t *testing.T) {
	core, repository, provider, _, projectID := newDeploymentTestCore(t, nil)
	job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
		ProjectID: projectID, Mode: domain.DeploymentModeProduction,
	})
	if err != nil {
		t.Fatalf("StartDeploymentJob() error = %v", err)
	}
	result, err := core.RunDeploymentJob(context.Background(), job.ID, nil)
	if err != nil {
		t.Fatalf("RunDeploymentJob() error = %v", err)
	}
	if result.State != domain.DeploymentStateBlocked || result.Preflight.Ready {
		t.Fatalf("production job = %+v, want blocked preflight", result)
	}
	productionBlockers := map[string]bool{
		"target-profile": false, "compatibility": false, "installation-capability": false,
		"result-verification": false, "recovery-capability": false, "bench-evidence": false, "production-policy": false,
	}
	for _, check := range result.Preflight.Checks {
		if check.Required && check.ID != "package-integrity" && check.Status != domain.DeploymentCheckBlocked && check.Status != domain.DeploymentCheckUnknown {
			t.Fatalf("production required check is not a clear blocker: %+v", check)
		}
		if _, expected := productionBlockers[check.ID]; expected {
			productionBlockers[check.ID] = check.Status == domain.DeploymentCheckBlocked || check.Status == domain.DeploymentCheckUnknown
		}
	}
	for checkID, blocked := range productionBlockers {
		if !blocked {
			t.Errorf("Production Mode preflight is missing blocker %q", checkID)
		}
	}
	if provider.simulateCalls != 0 || provider.verifyCalls != 0 || provider.recoveryCalls != 0 {
		t.Fatalf("production called provider: simulate=%d verify=%d recovery=%d", provider.simulateCalls, provider.verifyCalls, provider.recoveryCalls)
	}
	persisted, err := repository.GetDeploymentJob(context.Background(), job.ID)
	if err != nil || persisted.State != domain.DeploymentStateBlocked {
		t.Fatalf("blocked production job was not persisted: %+v, %v", persisted, err)
	}
}

func TestDeploymentPreflightBlocksUnknownAndUnsupportedSyntheticProfiles(t *testing.T) {
	for _, profileID := range []string{"unknown-firmware", "incompatible-hardware", "profile-outside-catalog"} {
		t.Run(profileID, func(t *testing.T) {
			core, _, provider, _, projectID := newDeploymentTestCore(t, nil)
			job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
				ProjectID: projectID, TargetProfileID: profileID, Mode: domain.DeploymentModeTest,
			})
			if err != nil {
				t.Fatalf("StartDeploymentJob() error = %v", err)
			}
			result, err := core.RunDeploymentJob(context.Background(), job.ID, nil)
			if err != nil {
				t.Fatalf("RunDeploymentJob() error = %v", err)
			}
			if result.State != domain.DeploymentStateBlocked || provider.simulateCalls != 0 {
				t.Fatalf("result = %+v, simulation calls = %d; want blocked without provider execution", result, provider.simulateCalls)
			}
		})
	}
}

func TestDeploymentPreflightBlocksProviderWithUnsafeOrMissingCapabilities(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*DeploymentProviderCapabilities)
	}{
		{name: "installation capability", edit: func(capabilities *DeploymentProviderCapabilities) { capabilities.CanInstall = true }},
		{name: "result verification missing", edit: func(capabilities *DeploymentProviderCapabilities) { capabilities.CanVerifyResult = false }},
		{name: "recovery check missing", edit: func(capabilities *DeploymentProviderCapabilities) { capabilities.CanCheckRecovery = false }},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &deploymentProviderStub{}
			capabilities := provider.Capabilities()
			test.edit(&capabilities)
			provider.capabilities = &capabilities
			core, _, _, _, projectID := newDeploymentTestCore(t, nil, provider)
			job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
				ProjectID: projectID, TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest,
			})
			if err != nil {
				t.Fatalf("StartDeploymentJob() error = %v", err)
			}
			result, err := core.RunDeploymentJob(context.Background(), job.ID, nil)
			if err != nil {
				t.Fatalf("RunDeploymentJob() error = %v", err)
			}
			if result.State != domain.DeploymentStateBlocked || provider.simulateCalls != 0 {
				t.Fatalf("result = %+v, simulate calls = %d; want blocked before execution", result, provider.simulateCalls)
			}
		})
	}
}

func TestDeploymentPackageIntegrityFailureStopsBeforeProvider(t *testing.T) {
	core, _, provider, verifier, projectID := newDeploymentTestCore(t, errors.New("synthetic integrity mismatch"))
	job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
		ProjectID: projectID, TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest,
	})
	if err != nil {
		t.Fatalf("StartDeploymentJob() error = %v", err)
	}
	result, err := core.RunDeploymentJob(context.Background(), job.ID, nil)
	if err != nil {
		t.Fatalf("RunDeploymentJob() error = %v", err)
	}
	if result.State != domain.DeploymentStateFailed || provider.simulateCalls != 0 || !verifier.called {
		t.Fatalf("result = %+v, provider calls = %d, verifier called = %t", result, provider.simulateCalls, verifier.called)
	}
}

func TestDeploymentProviderFailureRecordsRecoveryCheck(t *testing.T) {
	core, _, provider, _, projectID := newDeploymentTestCore(t, nil)
	provider.simulateErr = errors.New("synthetic provider failure")
	job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
		ProjectID: projectID, TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest,
	})
	if err != nil {
		t.Fatalf("StartDeploymentJob() error = %v", err)
	}
	result, err := core.RunDeploymentJob(context.Background(), job.ID, nil)
	if err != nil {
		t.Fatalf("RunDeploymentJob() error = %v", err)
	}
	if result.State != domain.DeploymentStateFailed || !result.Recovery.Available || !result.Recovery.Simulated || provider.recoveryCalls < 2 {
		t.Fatalf("failed result = %+v, recovery calls = %d; want an explicit post-failure simulation check", result, provider.recoveryCalls)
	}
}

func TestCancelDeploymentJobCancelsActiveProviderAndPersistsTerminalState(t *testing.T) {
	started := make(chan struct{})
	provider := &deploymentProviderStub{started: started, waitForCancel: true}
	core, _, _, _, projectID := newDeploymentTestCore(t, nil, provider)
	job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
		ProjectID: projectID, TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest,
	})
	if err != nil {
		t.Fatalf("StartDeploymentJob() error = %v", err)
	}
	finished := make(chan domain.DeploymentJob, 1)
	go func() {
		result, _ := core.RunDeploymentJob(context.Background(), job.ID, nil)
		finished <- result
	}()
	<-started
	if _, err := core.CancelDeploymentJob(context.Background(), job.ID); err != nil {
		t.Fatalf("CancelDeploymentJob() error = %v", err)
	}
	select {
	case result := <-finished:
		if result.State != domain.DeploymentStateCancelled {
			t.Fatalf("cancelled result state = %q", result.State)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled deployment did not finish")
	}
}

func TestDeploymentProviderTimeoutFailsAndChecksRecovery(t *testing.T) {
	provider := &deploymentProviderStub{waitForCancel: true}
	core, _, _, _, projectID := newDeploymentTestCore(t, nil, provider)
	core.deployment.runTimeout = 10 * time.Millisecond
	job, err := core.StartDeploymentJob(context.Background(), domain.DeploymentRequest{
		ProjectID: projectID, TargetProfileID: "demo-android-platform-a", Mode: domain.DeploymentModeTest,
	})
	if err != nil {
		t.Fatalf("StartDeploymentJob() error = %v", err)
	}
	result, err := core.RunDeploymentJob(context.Background(), job.ID, nil)
	if err != nil {
		t.Fatalf("RunDeploymentJob() error = %v", err)
	}
	if result.State != domain.DeploymentStateFailed || provider.recoveryCalls < 2 {
		t.Fatalf("timed out result = %+v, recovery calls = %d", result, provider.recoveryCalls)
	}
}

func TestStartDeploymentJobValidatesModeAndRequiredTestTarget(t *testing.T) {
	core, _, _, _, projectID := newDeploymentTestCore(t, nil)
	for _, request := range []domain.DeploymentRequest{
		{ProjectID: projectID, Mode: domain.DeploymentModeTest},
		{ProjectID: projectID, Mode: "unknown"},
	} {
		if _, err := core.StartDeploymentJob(context.Background(), request); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("StartDeploymentJob(%+v) error = %v, want invalid argument", request, err)
		}
	}
}

func newDeploymentTestCore(t *testing.T, verifierErr error, providers ...*deploymentProviderStub) (*Service, *deploymentRepositoryStub, *deploymentProviderStub, *deploymentVerifierStub, string) {
	t.Helper()
	localizationRepository := &localizationRepositoryFake{}
	localizationService, err := NewLocalizationService(localizationRepository)
	if err != nil {
		t.Fatal(err)
	}
	project, err := localizationService.ImportProject(context.Background(), ImportLocalizationRequest{
		Name: "Synthetic deployment", Locale: "ru-RU", Format: domain.LocalizationFormatAndroidXML,
		Filename: "strings.xml", Content: []byte(`<resources><string name="title">Home</string></resources>`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := localizationService.UpdateTranslation(context.Background(), project.ID, "title", "Главная"); err != nil {
		t.Fatal(err)
	}
	if _, err := localizationService.ApproveTranslation(context.Background(), project.ID, "title"); err != nil {
		t.Fatal(err)
	}
	repository := &deploymentRepositoryStub{jobs: make(map[string]domain.DeploymentJob)}
	provider := &deploymentProviderStub{}
	if len(providers) > 0 {
		provider = providers[0]
	}
	verifier := &deploymentVerifierStub{err: verifierErr}
	core, err := NewServiceWithDeployment(
		&simulatorGatewayStub{}, &catalogRepositoryStub{}, localizationRepository,
		repository, verifier, provider,
	)
	if err != nil {
		t.Fatal(err)
	}
	return core, repository, provider, verifier, project.ID
}

type deploymentRepositoryStub struct {
	mu   sync.Mutex
	jobs map[string]domain.DeploymentJob
	next int
}

func (r *deploymentRepositoryStub) CreateDeploymentJob(_ context.Context, job domain.DeploymentJob) (domain.DeploymentJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	job.ID = fmt.Sprintf("deployment-%d", r.next)
	job.CreatedAt, job.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	job.Events = []domain.DeploymentEvent{{JobID: job.ID, Sequence: 1, At: time.Now().UTC(), State: job.State, Stage: "queued", Message: "Deployment job created"}}
	r.jobs[job.ID] = cloneDeploymentJob(job)
	return cloneDeploymentJob(job), nil
}

func (r *deploymentRepositoryStub) TransitionDeploymentJob(_ context.Context, job domain.DeploymentJob, event domain.DeploymentEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.jobs[job.ID]
	if !ok {
		return domain.ErrDeploymentJobNotFound
	}
	if err := domain.ValidateDeploymentTransition(current.State, job.State); err != nil {
		return err
	}
	event.JobID, event.Sequence, event.State, event.At = job.ID, int64(len(current.Events)+1), job.State, time.Now().UTC()
	job.Events = append(append([]domain.DeploymentEvent(nil), current.Events...), event)
	job.CreatedAt, job.UpdatedAt = current.CreatedAt, event.At
	r.jobs[job.ID] = cloneDeploymentJob(job)
	return nil
}

func (r *deploymentRepositoryStub) AppendDeploymentEvent(_ context.Context, jobID string, expectedState domain.DeploymentState, event domain.DeploymentEvent) (domain.DeploymentEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[jobID]
	if !ok {
		return domain.DeploymentEvent{}, domain.ErrDeploymentJobNotFound
	}
	if job.State != expectedState || expectedState.Terminal() {
		return domain.DeploymentEvent{}, domain.ErrDeploymentStateConflict
	}
	event.JobID, event.Sequence, event.State, event.At = jobID, int64(len(job.Events)+1), expectedState, time.Now().UTC()
	job.Events = append(job.Events, event)
	job.UpdatedAt = event.At
	r.jobs[jobID] = cloneDeploymentJob(job)
	return event, nil
}

func (r *deploymentRepositoryStub) GetDeploymentJob(_ context.Context, id string) (domain.DeploymentJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok {
		return domain.DeploymentJob{}, domain.ErrDeploymentJobNotFound
	}
	return cloneDeploymentJob(job), nil
}

func (r *deploymentRepositoryStub) ListDeploymentJobs(context.Context, int) ([]domain.DeploymentJobSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	jobs := make([]domain.DeploymentJobSummary, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, domain.DeploymentJobSummary{ID: job.ID, ProjectID: job.ProjectID, TargetProfileID: job.TargetProfileID, Mode: job.Mode, PackageSHA256: job.PackageSHA256, State: job.State, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt})
	}
	return jobs, nil
}

func cloneDeploymentJob(job domain.DeploymentJob) domain.DeploymentJob {
	job.Events = append([]domain.DeploymentEvent(nil), job.Events...)
	job.Preflight.Checks = append([]domain.DeploymentCheck(nil), job.Preflight.Checks...)
	return job
}

type deploymentVerifierStub struct {
	called bool
	err    error
}

func (v *deploymentVerifierStub) VerifyLanguagePackage([]byte, string) error {
	v.called = true
	return v.err
}

type deploymentProviderStub struct {
	simulateCalls int
	verifyCalls   int
	recoveryCalls int
	simulateErr   error
	verifyErr     error
	started       chan struct{}
	waitForCancel bool
	capabilities  *DeploymentProviderCapabilities
}

func (p *deploymentProviderStub) ID() string { return "synthetic-local-simulator" }

func (p *deploymentProviderStub) Capabilities() DeploymentProviderCapabilities {
	if p.capabilities != nil {
		return *p.capabilities
	}
	return DeploymentProviderCapabilities{Mode: domain.DeploymentProviderModeSimulation, CanInstall: false, CanVerifyResult: true, CanCheckRecovery: true, CanSimulate: true}
}

func (p *deploymentProviderStub) Simulate(ctx context.Context, job domain.DeploymentJob, _ []byte, publish func(domain.DeploymentEvent) error) error {
	p.simulateCalls++
	if p.started != nil {
		select {
		case <-p.started:
		default:
			close(p.started)
		}
	}
	if p.waitForCancel {
		<-ctx.Done()
		return ctx.Err()
	}
	if p.simulateErr != nil {
		return p.simulateErr
	}
	return publish(domain.DeploymentEvent{JobID: job.ID, State: domain.DeploymentStateSimulating, Stage: "simulation.progress", Message: "Synthetic provider progress"})
}

func (p *deploymentProviderStub) VerifyResult(context.Context, domain.DeploymentJob, string) (domain.DeploymentResultVerification, error) {
	p.verifyCalls++
	if p.verifyErr != nil {
		return domain.DeploymentResultVerification{}, p.verifyErr
	}
	return domain.DeploymentResultVerification{Verified: true, Simulated: true, Message: "Synthetic result check passed."}, nil
}

func (p *deploymentProviderStub) CheckRecovery(context.Context, domain.DeploymentJob) (domain.RecoveryCapabilityResult, error) {
	p.recoveryCalls++
	return domain.RecoveryCapabilityResult{Available: true, Simulated: true, Message: "Synthetic recovery check only."}, nil
}
