package v1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSDKMinorVersionAndOptionalVehicleResearchInput(t *testing.T) {
	if SDKVersion != "1.1.0" {
		t.Fatalf("SDKVersion = %q, want 1.1.0", SDKVersion)
	}
	manifest := AdapterManifest{
		ManifestVersion: ManifestVersion,
		APIVersion:      APIVersion,
		SDKVersion:      SDKVersion,
		ID:              "adapter-geely-research",
		Name:            "Geely Research Adapter",
		Version:         "0.1.0",
		Executable:      "bin/adapter-geely-research",
		Capabilities: AdapterCapabilities{
			CheckCompatibility: true,
		},
		InputTypes: []AdapterInputType{
			InputCompatibilityProfile,
			InputVehicleResearchProfileJSON,
		},
		ResourceFormats: []string{},
	}
	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v", err)
	}

	manifest.Capabilities = AdapterCapabilities{Detect: true}
	manifest.InputTypes = []AdapterInputType{InputVehicleObservationJSON, InputVehicleResearchProfileJSON}
	if err := ValidateManifest(manifest); err == nil || !strings.Contains(err.Error(), "requires CheckCompatibility") {
		t.Fatalf("ValidateManifest() error = %v, want compatibility capability requirement", err)
	}
}

func TestCompatibilityRequestVehicleProfileIsOptional(t *testing.T) {
	encoded, err := json.Marshal(CompatibilityRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "vehicle_profile") {
		t.Fatalf("empty compatibility request unexpectedly contains vehicle_profile: %s", encoded)
	}

	const input = `{"vehicle_profile":{"schema":"carvion.geely-research/v1","profile_id":"test-profile"}}`
	var request CompatibilityRequest
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(request.VehicleProfile) != `{"schema":"carvion.geely-research/v1","profile_id":"test-profile"}` {
		t.Fatalf("VehicleProfile = %s, want original JSON object", request.VehicleProfile)
	}
}
