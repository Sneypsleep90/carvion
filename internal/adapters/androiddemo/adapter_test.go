package androiddemo

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/simulator"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestDetectRecognizesBothSyntheticSimulatorProfiles(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, profileID := range []string{"demo-android-platform-a", "demo-android-platform-b"} {
		t.Run(profileID, func(t *testing.T) {
			profile, err := simulator.LoadProfile(profileID)
			if err != nil {
				t.Fatal(err)
			}
			observation, err := json.Marshal(struct {
				ProfileID string              `json:"profile_id"`
				Identity  sdk.VehicleIdentity `json:"identity"`
			}{ProfileID: profileID, Identity: sdk.VehicleIdentity{
				Manufacturer: profile.Public.Identity.Manufacturer,
				Model:        profile.Public.Identity.Model,
				PlatformID:   profile.Public.Identity.PlatformID,
			}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := adapter.Detect(context.Background(), sdk.DetectRequest{Observation: observation})
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}
			if got.Status != sdk.StatusSupported {
				t.Fatalf("Detect() status = %q, want %q", got.Status, sdk.StatusSupported)
			}
		})
	}
}

func TestSimulatorProfilesUseExactBuildAndControlledRPCFault(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name       string
		profileID  string
		wantStatus sdk.CompatibilityStatus
		wantErr    bool
	}{
		{name: "platform B exact build", profileID: "demo-android-platform-b", wantStatus: sdk.StatusSupported},
		{name: "unknown firmware", profileID: "unknown-firmware", wantStatus: sdk.StatusUnknown},
		{name: "incompatible hardware", profileID: "incompatible-hardware", wantStatus: sdk.StatusUnsupported},
		{name: "controlled RPC error", profileID: "simulated-communication-failure", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := simulator.LoadProfile(tt.profileID)
			if err != nil {
				t.Fatal(err)
			}
			request := sdk.CompatibilityRequest{
				Identity: sdk.VehicleIdentity{
					Manufacturer: profile.Public.Identity.Manufacturer,
					Model:        profile.Public.Identity.Model,
					PlatformID:   profile.Public.Identity.PlatformID,
				},
				Hardware: sdk.HardwareInfo{PlatformID: profile.Public.Hardware.PlatformID, Revision: profile.Public.Hardware.Revision},
				Firmware: sdk.FirmwareInfo{BuildID: profile.Public.Firmware.BuildID, Version: profile.Public.Firmware.Version},
			}
			got, err := adapter.CheckCompatibility(context.Background(), request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckCompatibility() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got.Status != tt.wantStatus {
				t.Fatalf("CheckCompatibility() status = %q, want %q", got.Status, tt.wantStatus)
			}
		})
	}
}

func TestDetectRecognizesSyntheticJSONObservation(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	observation, err := os.ReadFile(filepath.Join("testdata", "vehicle.json"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := adapter.Detect(context.Background(), sdk.DetectRequest{Observation: observation})
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if result.Status != sdk.StatusSupported {
		t.Fatalf("status = %q, want %q", result.Status, sdk.StatusSupported)
	}
	if result.Identity.Manufacturer != "CARVION TEST LAB" || result.Identity.Model != "Android Demo Unit" {
		t.Errorf("identity = %+v, want synthetic demo identity", result.Identity)
	}
}

func TestCheckCompatibilityRequiresExactFirmwareAndHardwareMatch(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := supportedRequest()

	got, err := adapter.CheckCompatibility(context.Background(), request)
	if err != nil {
		t.Fatalf("CheckCompatibility() error = %v", err)
	}
	if got.Status != sdk.StatusSupported {
		t.Fatalf("matching profile status = %q, want %q", got.Status, sdk.StatusSupported)
	}

	request.Firmware.Version = "1.0.1-test"
	got, err = adapter.CheckCompatibility(context.Background(), request)
	if err != nil {
		t.Fatalf("CheckCompatibility() with mismatched version error = %v", err)
	}
	if got.Status != sdk.StatusUnknown {
		t.Errorf("mismatched firmware status = %q, want %q", got.Status, sdk.StatusUnknown)
	}

	request = supportedRequest()
	request.Hardware.Revision = "rev-b"
	got, err = adapter.CheckCompatibility(context.Background(), request)
	if err != nil {
		t.Fatalf("CheckCompatibility() with mismatched revision error = %v", err)
	}
	if got.Status != sdk.StatusUnsupported {
		t.Errorf("mismatched hardware status = %q, want %q", got.Status, sdk.StatusUnsupported)
	}
}

func TestCheckCompatibilityRefusesUnknownBuildAndVehicle(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	request := supportedRequest()
	request.Firmware.BuildID = "unlisted-build"
	got, err := adapter.CheckCompatibility(context.Background(), request)
	if err != nil {
		t.Fatalf("CheckCompatibility() unknown build error = %v", err)
	}
	if got.Status != sdk.StatusUnknown {
		t.Errorf("unknown build status = %q, want %q", got.Status, sdk.StatusUnknown)
	}

	request = supportedRequest()
	request.Identity.Model = "Unknown Vehicle"
	got, err = adapter.CheckCompatibility(context.Background(), request)
	if err != nil {
		t.Fatalf("CheckCompatibility() unknown vehicle error = %v", err)
	}
	if got.Status != sdk.StatusUnknown {
		t.Errorf("unknown vehicle status = %q, want %q", got.Status, sdk.StatusUnknown)
	}
}

func TestInspectResourcesReturnsTranslatableAndroidStrings(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join("testdata", "strings.xml"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := adapter.InspectResources(context.Background(), sdk.InspectResourcesRequest{
		Format:   "android-strings-xml",
		Filename: "strings.xml",
		Content:  string(content),
	})
	if err != nil {
		t.Fatalf("InspectResources() error = %v", err)
	}
	want := []sdk.TranslatableResource{
		{ID: "app_name", Text: "CARVION Demo"},
		{ID: "welcome_message", Text: "Welcome & explore"},
	}
	if len(got.Resources) != len(want) {
		t.Fatalf("resources = %+v, want %+v", got.Resources, want)
	}
	for i := range want {
		if got.Resources[i] != want[i] {
			t.Errorf("resource[%d] = %+v, want %+v", i, got.Resources[i], want[i])
		}
	}
}

func TestInspectResourcesRejectsMalformedXML(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = adapter.InspectResources(context.Background(), sdk.InspectResourcesRequest{
		Format:  "android-strings-xml",
		Content: "<resources><string name=\"broken\">",
	})
	if err == nil {
		t.Fatal("InspectResources() error = nil, want malformed XML error")
	}
}

func TestGetInfoReturnsVersionedManifest(t *testing.T) {
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	manifest, err := adapter.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}
	if manifest.APIVersion != sdk.APIVersion || manifest.ManifestVersion != sdk.ManifestVersion || manifest.SDKVersion != sdk.SDKVersion {
		t.Errorf("manifest versions = (%q, %d, %q), want (%q, %d, %q)", manifest.APIVersion, manifest.ManifestVersion, manifest.SDKVersion, sdk.APIVersion, sdk.ManifestVersion, sdk.SDKVersion)
	}
	if len(manifest.InputTypes) != 3 || len(manifest.ResourceFormats) != 1 || manifest.ResourceFormats[0] != "android-strings-xml" {
		t.Errorf("manifest capabilities metadata = inputs %v, formats %v; want declared SDK inputs and Android XML format", manifest.InputTypes, manifest.ResourceFormats)
	}
}

func supportedRequest() sdk.CompatibilityRequest {
	return sdk.CompatibilityRequest{
		Identity: sdk.VehicleIdentity{
			Manufacturer: "CARVION TEST LAB",
			Model:        "Android Demo Unit",
			PlatformID:   "android-demo-platform",
		},
		Hardware: sdk.HardwareInfo{PlatformID: "android-demo-platform", Revision: "rev-a"},
		Firmware: sdk.FirmwareInfo{BuildID: "demo-build-a-001", Version: "1.0.0-demo"},
	}
}
