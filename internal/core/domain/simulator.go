package domain

import "time"

type SimulatorProfile struct {
	ID                    string              `json:"id"`
	Name                  string              `json:"name"`
	Description           string              `json:"description"`
	Synthetic             bool                `json:"synthetic"`
	Identity              VehicleIdentity     `json:"identity"`
	Hardware              HardwareInfo        `json:"hardware"`
	Firmware              FirmwareInfo        `json:"firmware"`
	ExpectedCompatibility CompatibilityStatus `json:"expectedCompatibility"`
}

type SimulatorRunStatus string

const (
	SimulatorRunCompleted   SimulatorRunStatus = "completed"
	SimulatorRunUnsupported SimulatorRunStatus = "unsupported"
	SimulatorRunUnknown     SimulatorRunStatus = "unknown"
	SimulatorRunFailed      SimulatorRunStatus = "failed"
)

type SimulatorEvent struct {
	At      time.Time `json:"at"`
	Step    string    `json:"step"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

type SimulatorRunResult struct {
	RunID                 string              `json:"runId"`
	Profile               SimulatorProfile    `json:"profile"`
	AdapterID             string              `json:"adapterId,omitempty"`
	Status                SimulatorRunStatus  `json:"status"`
	Compatibility         CompatibilityResult `json:"compatibility"`
	LocalizationProjectID string              `json:"localizationProjectId,omitempty"`
	Package               LocalizationPackage `json:"package"`
	PackageVerified       bool                `json:"packageVerified"`
	ErrorMessage          string              `json:"errorMessage,omitempty"`
	Events                []SimulatorEvent    `json:"events"`
}
