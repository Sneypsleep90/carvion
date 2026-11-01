// Package v1 defines the stable CARVION adapter contract for API version v1.
package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	APIVersion      = "v1"
	SDKVersion      = "1.0.0"
	SDKMajorVersion = 1
	ManifestVersion = 2
)

type AdapterInputType string

const (
	InputVehicleObservationJSON AdapterInputType = "vehicle-observation-json"
	InputCompatibilityProfile   AdapterInputType = "compatibility-profile"
	InputResourceDocument       AdapterInputType = "resource-document"
)

var (
	adapterIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	semverPattern    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	formatPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9.+-]{0,63}$`)
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
	SDKVersion      string              `json:"sdk_version"`
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Version         string              `json:"version"`
	Executable      string              `json:"executable"`
	Capabilities    AdapterCapabilities `json:"capabilities"`
	InputTypes      []AdapterInputType  `json:"input_types"`
	ResourceFormats []string            `json:"resource_formats"`
}

// ValidateManifest checks the public SDK contract and capability declarations.
// Plugin managers must additionally validate executable paths against a trusted
// local plugin directory before launching the adapter.
func ValidateManifest(manifest AdapterManifest) error {
	if manifest.ManifestVersion != ManifestVersion {
		return fmt.Errorf("unsupported manifest version %d", manifest.ManifestVersion)
	}
	if manifest.APIVersion != APIVersion {
		return fmt.Errorf("unsupported adapter API version %q", manifest.APIVersion)
	}
	if !semverPattern.MatchString(manifest.SDKVersion) {
		return errors.New("adapter SDK version must use semantic versioning")
	}
	major, err := strconv.Atoi(strings.SplitN(manifest.SDKVersion, ".", 2)[0])
	if err != nil || major != SDKMajorVersion {
		return fmt.Errorf("unsupported adapter SDK major version %q", manifest.SDKVersion)
	}
	if !adapterIDPattern.MatchString(manifest.ID) {
		return errors.New("adapter ID must contain lowercase letters, digits, and hyphens")
	}
	if name := strings.TrimSpace(manifest.Name); name == "" || len(name) > 128 {
		return errors.New("adapter name must be between 1 and 128 characters")
	}
	if !semverPattern.MatchString(manifest.Version) {
		return errors.New("adapter version must use semantic versioning")
	}
	if strings.TrimSpace(manifest.Executable) == "" {
		return errors.New("adapter executable is required")
	}
	if manifest.InputTypes == nil {
		return errors.New("input_types must be declared as an array")
	}
	if manifest.ResourceFormats == nil {
		return errors.New("resource_formats must be declared as an array")
	}
	if !manifest.Capabilities.Detect && !manifest.Capabilities.CheckCompatibility && !manifest.Capabilities.InspectResources {
		return errors.New("adapter must declare at least one capability")
	}

	inputTypes := make(map[AdapterInputType]struct{}, len(manifest.InputTypes))
	for _, inputType := range manifest.InputTypes {
		if _, exists := inputTypes[inputType]; exists {
			return fmt.Errorf("duplicate adapter input type %q", inputType)
		}
		switch inputType {
		case InputVehicleObservationJSON, InputCompatibilityProfile, InputResourceDocument:
			inputTypes[inputType] = struct{}{}
		default:
			return fmt.Errorf("unsupported adapter input type %q", inputType)
		}
	}
	if manifest.Capabilities.Detect && !hasInputType(inputTypes, InputVehicleObservationJSON) {
		return errors.New("Detect capability requires a vehicle observation input type")
	}
	if manifest.Capabilities.CheckCompatibility && !hasInputType(inputTypes, InputCompatibilityProfile) {
		return errors.New("CheckCompatibility capability requires a compatibility profile input type")
	}
	if manifest.Capabilities.InspectResources && !hasInputType(inputTypes, InputResourceDocument) {
		return errors.New("InspectResources requires a resource document input type")
	}
	if !manifest.Capabilities.Detect && hasInputType(inputTypes, InputVehicleObservationJSON) {
		return errors.New("vehicle observation input type requires Detect capability")
	}
	if !manifest.Capabilities.CheckCompatibility && hasInputType(inputTypes, InputCompatibilityProfile) {
		return errors.New("compatibility profile input type requires CheckCompatibility capability")
	}
	if !manifest.Capabilities.InspectResources && (hasInputType(inputTypes, InputResourceDocument) || len(manifest.ResourceFormats) > 0) {
		return errors.New("resource input types and formats require InspectResources capability")
	}

	resourceFormats := make(map[string]struct{}, len(manifest.ResourceFormats))
	for _, format := range manifest.ResourceFormats {
		if !formatPattern.MatchString(format) {
			return fmt.Errorf("invalid resource format %q", format)
		}
		if _, exists := resourceFormats[format]; exists {
			return fmt.Errorf("duplicate resource format %q", format)
		}
		resourceFormats[format] = struct{}{}
	}
	if manifest.Capabilities.InspectResources && len(resourceFormats) == 0 {
		return errors.New("InspectResources capability requires at least one resource format")
	}
	return nil
}

func hasInputType(inputTypes map[AdapterInputType]struct{}, inputType AdapterInputType) bool {
	_, ok := inputTypes[inputType]
	return ok
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
