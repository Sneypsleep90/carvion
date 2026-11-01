package domain

type CompatibilityStatus string

const (
	StatusSupported   CompatibilityStatus = "supported"
	StatusUnsupported CompatibilityStatus = "unsupported"
	StatusUnknown     CompatibilityStatus = "unknown"
)

type AdapterCapabilities struct {
	Detect             bool
	CheckCompatibility bool
	InspectResources   bool
}

type AdapterStatus string

const (
	AdapterStatusAvailable   AdapterStatus = "available"
	AdapterStatusUnavailable AdapterStatus = "unavailable"
)

type AdapterInfo struct {
	ID              string
	Name            string
	Version         string
	APIVersion      string
	SDKVersion      string
	Capabilities    AdapterCapabilities
	InputTypes      []string
	ResourceFormats []string
	Status          AdapterStatus
}

type VehicleIdentity struct {
	Manufacturer string
	Model        string
	PlatformID   string
}

type HardwareInfo struct {
	PlatformID string
	Revision   string
}

type FirmwareInfo struct {
	BuildID string
	Version string
}

type CompatibilityRequest struct {
	Identity VehicleIdentity
	Hardware HardwareInfo
	Firmware FirmwareInfo
}

type CompatibilityResult struct {
	Status CompatibilityStatus
	Reason string
}

type DetectRequest struct {
	Observation []byte
}

type DetectionResult struct {
	Status   CompatibilityStatus
	Identity VehicleIdentity
	Reason   string
}

type InspectResourcesRequest struct {
	Format   string
	Filename string
	Content  string
}

type TranslatableResource struct {
	ID   string
	Text string
}

type ResourceInspection struct {
	Resources []TranslatableResource
}
