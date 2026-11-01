package deployment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/application"
	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/simulator"
)

const simulatedProviderID = "synthetic-local-simulator"

type SimulatedProvider struct{}

func NewSimulatedProvider() *SimulatedProvider {
	return &SimulatedProvider{}
}

func (p *SimulatedProvider) ID() string { return simulatedProviderID }

func (p *SimulatedProvider) Capabilities() application.DeploymentProviderCapabilities {
	return application.DeploymentProviderCapabilities{
		Mode:             domain.DeploymentProviderModeSimulation,
		CanInstall:       false,
		CanVerifyResult:  true,
		CanCheckRecovery: true,
		CanSimulate:      true,
	}
}

func (p *SimulatedProvider) Simulate(ctx context.Context, job domain.DeploymentJob, artifact []byte, publish func(domain.DeploymentEvent) error) error {
	if err := validateSimulationContext(ctx); err != nil {
		return err
	}
	profile, err := p.validateSyntheticJob(job)
	if err != nil {
		return err
	}
	if job.State != domain.DeploymentStateSimulating {
		return fmt.Errorf("%w: simulated provider requires job state simulating", domain.ErrInvalidArgument)
	}
	if len(artifact) == 0 || packageSHA256(artifact) != job.PackageSHA256 {
		return errors.New("simulated provider package bytes do not match the deployment job hash")
	}
	if publish == nil {
		return fmt.Errorf("%w: progress publisher is required", domain.ErrInvalidArgument)
	}
	events := []domain.DeploymentEvent{
		{Stage: "simulation.started", Message: "Запущена локальная симуляция; автомобиль и оборудование не используются."},
		{Stage: "simulation.completed", Message: fmt.Sprintf("Синтетический сценарий %q завершён без изменения устройства.", profile.Public.Name)},
	}
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		event.JobID = job.ID
		event.At = time.Now().UTC()
		event.State = domain.DeploymentStateSimulating
		if err := publish(event); err != nil {
			return fmt.Errorf("publish simulated deployment progress: %w", err)
		}
	}
	return ctx.Err()
}

func (p *SimulatedProvider) VerifyResult(ctx context.Context, job domain.DeploymentJob, packageSHA256 string) (domain.DeploymentResultVerification, error) {
	if err := validateSimulationContext(ctx); err != nil {
		return domain.DeploymentResultVerification{}, err
	}
	if _, err := p.validateSyntheticJob(job); err != nil {
		return domain.DeploymentResultVerification{}, err
	}
	if !isSHA256(packageSHA256) || packageSHA256 != job.PackageSHA256 {
		return domain.DeploymentResultVerification{}, errors.New("simulated result package SHA-256 does not match the deployment job")
	}
	return domain.DeploymentResultVerification{
		Verified: true, Simulated: true,
		Message: "Synthetic result check passed; no vehicle was changed.",
	}, nil
}

func (p *SimulatedProvider) CheckRecovery(ctx context.Context, job domain.DeploymentJob) (domain.RecoveryCapabilityResult, error) {
	if err := validateSimulationContext(ctx); err != nil {
		return domain.RecoveryCapabilityResult{}, err
	}
	if _, err := p.validateSyntheticJob(job); err != nil {
		return domain.RecoveryCapabilityResult{}, err
	}
	return domain.RecoveryCapabilityResult{
		Available: true, Simulated: true,
		Message: "Recovery capability was checked for the synthetic scenario; no rollback or device change occurred.",
	}, nil
}

func (p *SimulatedProvider) validateSyntheticJob(job domain.DeploymentJob) (simulator.Profile, error) {
	if job.Mode != domain.DeploymentModeTest || job.ProviderID != p.ID() || job.TargetProfileID == "" {
		return simulator.Profile{}, fmt.Errorf("%w: simulated provider accepts only a selected test profile", domain.ErrInvalidArgument)
	}
	profile, err := simulator.LoadProfile(job.TargetProfileID)
	if err != nil {
		return simulator.Profile{}, fmt.Errorf("load synthetic deployment profile: %w", err)
	}
	if !profile.Public.Synthetic || profile.Public.ExpectedCompatibility != domain.StatusSupported {
		return simulator.Profile{}, fmt.Errorf("%w: selected synthetic profile is not exactly supported", domain.ErrCapabilityUnavailable)
	}
	return profile, nil
}

func validateSimulationContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", domain.ErrInvalidArgument)
	}
	return ctx.Err()
}

var _ application.DeploymentProvider = (*SimulatedProvider)(nil)
