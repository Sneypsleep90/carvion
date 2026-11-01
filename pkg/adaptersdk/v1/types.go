// Package v1 defines the stable CARVION adapter contract for API version v1.
package v1

import (
	"context"
	"encoding/json"
)

const (
	APIVersion      = "v1"
	ManifestVersion = 1
)

type CompatibilityStatus string

const (
	StatusSupported   CompatibilityStatus = "supported"
	StatusUnsupported CompatibilityStatus = "unsupported"
	StatusUnknown     CompatibilityStatus = "unknown"
)

type AdapterManifest struct {
	ManifestVersion int                 `json:"manifest_version"`
	APIVersion      string              `json:"api_version"`
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Version         string              `json:"version"`
	Executable      string              `json:"executable"`
	Capabilities    AdapterCapabilities `json:"capabilities"`
}

type AdapterCapabilities struct {
	Detect             bool `json:"detect"`
	CheckCompatibility bool `json:"check_compatibility"`
	InspectResources   bool `json:"inspect_resources"`
}

type VehicleIdentity struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	PlatformID   string `json:"platform_id"`
}

type FirmwareInfo struct {
	BuildID string `json:"build_id"`
	Version string `json:"version"`
}

type HardwareInfo struct {
	PlatformID string `json:"platform_id"`
	Revision   string `json:"revision"`
}

type CompatibilityRequest struct {
	Identity VehicleIdentity `json:"identity"`
	Hardware HardwareInfo    `json:"hardware"`
	Firmware FirmwareInfo    `json:"firmware"`
}

type CompatibilityResult struct {
	Status CompatibilityStatus `json:"status"`
	Reason string              `json:"reason"`
}

type DetectRequest struct {
	// Observation is JSON data collected and supplied by the CARVION core.
	Observation json.RawMessage `json:"observation"`
}

type DetectionResult struct {
	Status   CompatibilityStatus `json:"status"`
	Identity VehicleIdentity     `json:"identity"`
	Reason   string              `json:"reason,omitempty"`
}

type InspectResourcesRequest struct {
	Format   string `json:"format"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type TranslatableResource struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type ResourceInspection struct {
	Resources []TranslatableResource `json:"resources"`
}

type Adapter interface {
	GetInfo(context.Context) (AdapterManifest, error)
	Detect(context.Context, DetectRequest) (DetectionResult, error)
	CheckCompatibility(context.Context, CompatibilityRequest) (CompatibilityResult, error)
	InspectResources(context.Context, InspectResourcesRequest) (ResourceInspection, error)
}
