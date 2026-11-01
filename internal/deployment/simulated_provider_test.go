package deployment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestSimulatedProviderDeclaresNoInstallationCapability(t *testing.T) {
	provider := NewSimulatedProvider()
	capabilities := provider.Capabilities()
	if provider.ID() != "synthetic-local-simulator" || capabilities.Mode != domain.DeploymentProviderModeSimulation {
		t.Fatalf("provider identity/capabilities = %q, %+v", provider.ID(), capabilities)
	}
	if capabilities.CanInstall || !capabilities.CanSimulate || !capabilities.CanVerifyResult || !capabilities.CanCheckRecovery {
		t.Fatalf("provider capabilities = %+v, want simulation only with separate verification/recovery checks", capabilities)
	}
}

func TestSimulatedProviderRunsOnlyKnownSupportedSyntheticProfile(t *testing.T) {
	provider := NewSimulatedProvider()
	for _, test := range []struct {
		name    string
		mode    domain.DeploymentMode
		profile string
	}{
		{name: "production mode", mode: domain.DeploymentModeProduction, profile: "demo-android-platform-a"},
		{name: "unknown profile", mode: domain.DeploymentModeTest, profile: "not-in-catalog"},
		{name: "unknown firmware", mode: domain.DeploymentModeTest, profile: "unknown-firmware"},
		{name: "unsupported hardware", mode: domain.DeploymentModeTest, profile: "incompatible-hardware"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := provider.Simulate(context.Background(), deploymentTestJob(test.mode, test.profile), []byte("package"), func(domain.DeploymentEvent) error {
				t.Fatal("provider emitted progress for an invalid test target")
				return nil
			})
			if err == nil {
				t.Fatal("Simulate() accepted an invalid or production target")
			}
		})
	}
}

func TestSimulatedProviderEmitsBoundedProgressAndPropagatesErrors(t *testing.T) {
	provider := NewSimulatedProvider()
	job := deploymentTestJob(domain.DeploymentModeTest, "demo-android-platform-a")
	var events []domain.DeploymentEvent
	err := provider.Simulate(context.Background(), job, []byte("package"), func(event domain.DeploymentEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("Simulate() error = %v", err)
	}
	if len(events) < 1 || len(events) > 3 {
		t.Fatalf("simulation emitted %d events, want a bounded progress sequence", len(events))
	}
	for _, event := range events {
		if event.State != domain.DeploymentStateSimulating || event.JobID != job.ID || event.Stage == "" || event.Message == "" {
			t.Fatalf("simulation emitted invalid progress event: %+v", event)
		}
	}
	callbackErr := errors.New("event persistence failed")
	if err := provider.Simulate(context.Background(), job, []byte("package"), func(domain.DeploymentEvent) error { return callbackErr }); !errors.Is(err, callbackErr) {
		t.Fatalf("Simulate() callback error = %v, want %v", err, callbackErr)
	}
}

func TestSimulatedProviderChecksPackageHashAndRecoveryWithoutClaimingInstallation(t *testing.T) {
	provider := NewSimulatedProvider()
	job := deploymentTestJob(domain.DeploymentModeTest, "demo-android-platform-a")
	packageBytes := []byte("synthetic package bytes")
	job.PackageSHA256 = packageSHA256(packageBytes)

	verification, err := provider.VerifyResult(context.Background(), job, job.PackageSHA256)
	if err != nil {
		t.Fatalf("VerifyResult() error = %v", err)
	}
	if !verification.Verified || !verification.Simulated || strings.Contains(strings.ToLower(verification.Message), "installed") {
		t.Fatalf("verification = %+v, want explicitly simulated result", verification)
	}
	if _, err := provider.VerifyResult(context.Background(), job, strings.Repeat("0", 64)); err == nil {
		t.Fatal("VerifyResult() accepted a mismatched package hash")
	}
	recovery, err := provider.CheckRecovery(context.Background(), job)
	if err != nil {
		t.Fatalf("CheckRecovery() error = %v", err)
	}
	if !recovery.Available || !recovery.Simulated || strings.Contains(strings.ToLower(recovery.Message), "rollback performed") {
		t.Fatalf("recovery = %+v, want simulated-only recovery availability", recovery)
	}
}

func TestSimulatedProviderHonorsCancellation(t *testing.T) {
	provider := NewSimulatedProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := provider.Simulate(ctx, deploymentTestJob(domain.DeploymentModeTest, "demo-android-platform-a"), []byte("package"), func(domain.DeploymentEvent) error {
		t.Fatal("cancelled simulation emitted progress")
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Simulate() error = %v, want context.Canceled", err)
	}
}

func deploymentTestJob(mode domain.DeploymentMode, profileID string) domain.DeploymentJob {
	return domain.DeploymentJob{
		ID: "deployment-test-id", ProjectID: "project-test-id", TargetProfileID: profileID,
		Mode: mode, ProviderID: "synthetic-local-simulator", PackageSHA256: packageSHA256([]byte("package")),
		State: domain.DeploymentStateSimulating,
	}
}
