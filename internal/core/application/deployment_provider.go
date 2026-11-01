package application

import (
	"context"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

type DeploymentProviderCapabilities struct {
	Mode             domain.DeploymentProviderMode `json:"mode"`
	CanInstall       bool                          `json:"canInstall"`
	CanVerifyResult  bool                          `json:"canVerifyResult"`
	CanCheckRecovery bool                          `json:"canCheckRecovery"`
	CanSimulate      bool                          `json:"canSimulate"`
}

// DeploymentProvider never receives a database handle, filesystem path or
// hardware connection. The synthetic provider only receives bounded bytes
// already read and verified by Core.
type DeploymentProvider interface {
	ID() string
	Capabilities() DeploymentProviderCapabilities
	Simulate(context.Context, domain.DeploymentJob, []byte, func(domain.DeploymentEvent) error) error
	VerifyResult(context.Context, domain.DeploymentJob, string) (domain.DeploymentResultVerification, error)
	CheckRecovery(context.Context, domain.DeploymentJob) (domain.RecoveryCapabilityResult, error)
}

type DeploymentPackageVerifier interface {
	VerifyLanguagePackage([]byte, string) error
}

type DeploymentRepository interface {
	CreateDeploymentJob(context.Context, domain.DeploymentJob) (domain.DeploymentJob, error)
	TransitionDeploymentJob(context.Context, domain.DeploymentJob, domain.DeploymentEvent) error
	AppendDeploymentEvent(context.Context, string, domain.DeploymentState, domain.DeploymentEvent) (domain.DeploymentEvent, error)
	GetDeploymentJob(context.Context, string) (domain.DeploymentJob, error)
	ListDeploymentJobs(context.Context, int) ([]domain.DeploymentJobSummary, error)
}

type DeploymentProgressPublisher func(domain.DeploymentEvent) error
