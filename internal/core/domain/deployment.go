package domain

import (
	"errors"
	"time"
)

type DeploymentMode string

const (
	DeploymentModeTest       DeploymentMode = "test"
	DeploymentModeProduction DeploymentMode = "production"
)

type DeploymentProviderMode string

const (
	DeploymentProviderModeSimulation DeploymentProviderMode = "simulation"
	DeploymentProviderModeProduction DeploymentProviderMode = "production"
)

type DeploymentState string

const (
	DeploymentStateQueued          DeploymentState = "queued"
	DeploymentStatePreflighting    DeploymentState = "preflighting"
	DeploymentStateBlocked         DeploymentState = "blocked"
	DeploymentStatePreflightPassed DeploymentState = "preflight-passed"
	DeploymentStateSimulating      DeploymentState = "simulating"
	DeploymentStateVerifying       DeploymentState = "verifying"
	DeploymentStateSimulated       DeploymentState = "simulated"
	DeploymentStateFailed          DeploymentState = "failed"
	DeploymentStateCancelled       DeploymentState = "cancelled"
)

type DeploymentCheckStatus string

const (
	DeploymentCheckPassed  DeploymentCheckStatus = "passed"
	DeploymentCheckBlocked DeploymentCheckStatus = "blocked"
	DeploymentCheckFailed  DeploymentCheckStatus = "failed"
	DeploymentCheckUnknown DeploymentCheckStatus = "unknown"
)

type DeploymentJob struct {
	ID                 string                       `json:"id"`
	ProjectID          string                       `json:"projectId"`
	TargetProfileID    string                       `json:"targetProfileId,omitempty"`
	Mode               DeploymentMode               `json:"mode"`
	ProviderID         string                       `json:"providerId,omitempty"`
	PackageSHA256      string                       `json:"packageSha256"`
	State              DeploymentState              `json:"state"`
	Preflight          DeploymentPreflight          `json:"preflight"`
	ResultVerification DeploymentResultVerification `json:"resultVerification"`
	Recovery           RecoveryCapabilityResult     `json:"recovery"`
	ErrorMessage       string                       `json:"errorMessage,omitempty"`
	Events             []DeploymentEvent            `json:"events"`
	CreatedAt          time.Time                    `json:"createdAt,omitempty"`
	UpdatedAt          time.Time                    `json:"updatedAt,omitempty"`
}

type DeploymentJobSummary struct {
	ID              string          `json:"id"`
	ProjectID       string          `json:"projectId"`
	TargetProfileID string          `json:"targetProfileId,omitempty"`
	Mode            DeploymentMode  `json:"mode"`
	PackageSHA256   string          `json:"packageSha256"`
	State           DeploymentState `json:"state"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type DeploymentRequest struct {
	ProjectID       string         `json:"projectId"`
	TargetProfileID string         `json:"targetProfileId,omitempty"`
	Mode            DeploymentMode `json:"mode"`
}

type DeploymentCheck struct {
	ID       string                `json:"id"`
	Status   DeploymentCheckStatus `json:"status"`
	Required bool                  `json:"required"`
	Message  string                `json:"message"`
}

type DeploymentPreflight struct {
	Ready  bool              `json:"ready"`
	Checks []DeploymentCheck `json:"checks"`
}

type DeploymentEvent struct {
	JobID    string          `json:"jobId"`
	Sequence int64           `json:"sequence"`
	At       time.Time       `json:"at"`
	State    DeploymentState `json:"state"`
	Stage    string          `json:"stage"`
	Message  string          `json:"message"`
}

type DeploymentResultVerification struct {
	Verified  bool   `json:"verified"`
	Simulated bool   `json:"simulated"`
	Message   string `json:"message"`
}

type RecoveryCapabilityResult struct {
	Available bool   `json:"available"`
	Simulated bool   `json:"simulated"`
	Message   string `json:"message"`
}

func (state DeploymentState) Terminal() bool {
	switch state {
	case DeploymentStateBlocked, DeploymentStateSimulated, DeploymentStateFailed, DeploymentStateCancelled:
		return true
	default:
		return false
	}
}

func ValidateDeploymentTransition(from, to DeploymentState) error {
	allowed := false
	switch from {
	case DeploymentStateQueued:
		allowed = to == DeploymentStatePreflighting || to == DeploymentStateCancelled
	case DeploymentStatePreflighting:
		allowed = to == DeploymentStateBlocked || to == DeploymentStatePreflightPassed || to == DeploymentStateFailed || to == DeploymentStateCancelled
	case DeploymentStatePreflightPassed:
		allowed = to == DeploymentStateSimulating || to == DeploymentStateFailed || to == DeploymentStateCancelled
	case DeploymentStateSimulating:
		allowed = to == DeploymentStateVerifying || to == DeploymentStateFailed || to == DeploymentStateCancelled
	case DeploymentStateVerifying:
		allowed = to == DeploymentStateSimulated || to == DeploymentStateFailed || to == DeploymentStateCancelled
	}
	if !allowed {
		return errors.New("deployment state transition is not allowed")
	}
	return nil
}
