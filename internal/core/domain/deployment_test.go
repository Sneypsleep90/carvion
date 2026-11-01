package domain

import "testing"

func TestValidateDeploymentTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    DeploymentState
		to      DeploymentState
		wantErr bool
	}{
		{name: "queue enters preflight", from: DeploymentStateQueued, to: DeploymentStatePreflighting},
		{name: "preflight blocks production", from: DeploymentStatePreflighting, to: DeploymentStateBlocked},
		{name: "preflight clears for simulation", from: DeploymentStatePreflighting, to: DeploymentStatePreflightPassed},
		{name: "passed preflight starts simulation", from: DeploymentStatePreflightPassed, to: DeploymentStateSimulating},
		{name: "simulation verifies result", from: DeploymentStateSimulating, to: DeploymentStateVerifying},
		{name: "verification completes simulation", from: DeploymentStateVerifying, to: DeploymentStateSimulated},
		{name: "simulation can fail", from: DeploymentStateSimulating, to: DeploymentStateFailed},
		{name: "verification can fail", from: DeploymentStateVerifying, to: DeploymentStateFailed},
		{name: "simulation can be cancelled", from: DeploymentStateSimulating, to: DeploymentStateCancelled},
		{name: "queue cannot skip preflight", from: DeploymentStateQueued, to: DeploymentStateSimulating, wantErr: true},
		{name: "blocked is terminal", from: DeploymentStateBlocked, to: DeploymentStateSimulated, wantErr: true},
		{name: "simulated is terminal", from: DeploymentStateSimulated, to: DeploymentStateFailed, wantErr: true},
		{name: "same state is not a transition", from: DeploymentStateSimulating, to: DeploymentStateSimulating, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDeploymentTransition(test.from, test.to)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateDeploymentTransition(%q, %q) error = %v, wantErr %t", test.from, test.to, err, test.wantErr)
			}
		})
	}
}

func TestDeploymentStateTerminal(t *testing.T) {
	if !DeploymentStateBlocked.Terminal() || !DeploymentStateSimulated.Terminal() || !DeploymentStateFailed.Terminal() || !DeploymentStateCancelled.Terminal() {
		t.Fatal("blocked, simulated, failed and cancelled states must be terminal")
	}
	if DeploymentStateQueued.Terminal() || DeploymentStatePreflighting.Terminal() || DeploymentStateSimulating.Terminal() {
		t.Fatal("active deployment states must not be terminal")
	}
}
