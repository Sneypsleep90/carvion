package geelyresearch

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	research "github.com/Sneypsleep90/carvion/internal/geelyresearch"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestGetInfoDeclaresResearchContract(t *testing.T) {
	adapter := newTestAdapter(t)
	manifest, err := adapter.GetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}
	if err := sdk.ValidateManifest(manifest); err != nil {
		t.Fatalf("ValidateManifest() error = %v", err)
	}
	if manifest.ID != "adapter-geely-research" || manifest.SDKVersion != "1.1.0" ||
		!manifest.Capabilities.Detect || !manifest.Capabilities.CheckCompatibility || !manifest.Capabilities.InspectResources {
		t.Fatalf("GetInfo() manifest = %+v, want Geely research SDK 1.1 manifest", manifest)
	}
	if !containsInput(manifest.InputTypes, sdk.InputVehicleResearchProfileJSON) {
		t.Fatalf("manifest input types = %v, missing research profile", manifest.InputTypes)
	}
	if len(manifest.ResourceFormats) != 2 || !containsFormat(manifest.ResourceFormats, "android-strings-xml") || !containsFormat(manifest.ResourceFormats, "carvion-json-v1") {
		t.Fatalf("manifest resource formats = %v, want Android XML and CARVION JSON", manifest.ResourceFormats)
	}
}

func TestDetectKeepsImportedProfileUnknown(t *testing.T) {
	adapter := newTestAdapter(t)
	profileJSON := testProfileJSON(t)

	got, err := adapter.Detect(context.Background(), sdk.DetectRequest{Observation: profileJSON})
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got.Status != sdk.StatusUnknown || got.Identity.Manufacturer != "TEST ONLY" || got.Identity.Model != "TEST MODEL" {
		t.Fatalf("Detect() = %+v, want unverified identity with unknown status", got)
	}
	if !strings.Contains(got.Reason, "file") || !strings.Contains(got.Reason, "connection") {
		t.Fatalf("Detect() reason = %q, want explicit file-only/no-connection explanation", got.Reason)
	}
}

func TestCheckCompatibilityRequiresFullExactProfileAndStaysUnknownWithoutRules(t *testing.T) {
	adapter := newTestAdapter(t)
	profileJSON := testProfileJSON(t)
	profile, err := research.ParseProfile(profileJSON)
	if err != nil {
		t.Fatal(err)
	}

	request := requestFromProfile(profile, profileJSON)
	got, err := adapter.CheckCompatibility(context.Background(), request)
	if err != nil {
		t.Fatalf("CheckCompatibility() error = %v", err)
	}
	if got.Status != sdk.StatusUnknown {
		t.Fatalf("CheckCompatibility() = %+v, want unknown with empty Geely catalog", got)
	}

	request.VehicleProfile = nil
	got, err = adapter.CheckCompatibility(context.Background(), request)
	if err != nil || got.Status != sdk.StatusUnknown {
		t.Fatalf("CheckCompatibility() without full profile = %+v, %v; want unknown", got, err)
	}

	request = requestFromProfile(profile, profileJSON)
	request.Firmware.BuildID += "-mismatch"
	got, err = adapter.CheckCompatibility(context.Background(), request)
	if err != nil || got.Status != sdk.StatusUnknown {
		t.Fatalf("CheckCompatibility() with conflicting SDK fields = %+v, %v; want unknown", got, err)
	}
}

func TestCheckCompatibilityRejectsMalformedProfile(t *testing.T) {
	adapter := newTestAdapter(t)
	request := sdk.CompatibilityRequest{VehicleProfile: json.RawMessage(`{"schema":"wrong"}`)}
	if _, err := adapter.CheckCompatibility(context.Background(), request); err == nil {
		t.Fatal("CheckCompatibility() error = nil, want invalid profile error")
	}
}

func TestInspectResourcesUsesSharedLocalizationParser(t *testing.T) {
	adapter := newTestAdapter(t)
	tests := []struct {
		name     string
		format   string
		filename string
		content  string
		want     []sdk.TranslatableResource
	}{
		{
			name:     "Android XML strings arrays plurals and flags",
			format:   "android-strings-xml",
			filename: "strings.xml",
			content:  `<resources><string name="title">Hello</string><string name="fixed" translatable="false">Fixed</string><string-array name="items"><item>One</item><item>Two</item></string-array><plurals name="count"><item quantity="one">One</item><item quantity="other">Many</item></plurals></resources>`,
			want: []sdk.TranslatableResource{
				{ID: "title", Text: "Hello"},
				{ID: "items[0]", Text: "One"},
				{ID: "items[1]", Text: "Two"},
				{ID: "count#one", Text: "One"},
				{ID: "count#other", Text: "Many"},
			},
		},
		{
			name:     "CARVION JSON resource schema",
			format:   "carvion-json-v1",
			filename: "resources.json",
			content:  `{"schema":"carvion.localization/v1","metadata":{"source":"test"},"resources":[{"key":"title","kind":"string","text":"Hello"},{"key":"fixed","kind":"string","text":"Fixed","translatable":false}]}`,
			want:     []sdk.TranslatableResource{{ID: "title", Text: "Hello"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := adapter.InspectResources(context.Background(), sdk.InspectResourcesRequest{
				Format: test.format, Filename: test.filename, Content: test.content,
			})
			if err != nil {
				t.Fatalf("InspectResources() error = %v", err)
			}
			if !equalResources(got.Resources, test.want) {
				t.Fatalf("InspectResources() = %+v, want %+v", got.Resources, test.want)
			}
		})
	}
}

func TestInspectResourcesRejectsMalformedAndUnsafeFiles(t *testing.T) {
	adapter := newTestAdapter(t)
	tests := []struct {
		name    string
		request sdk.InspectResourcesRequest
	}{
		{
			name:    "malformed XML",
			request: sdk.InspectResourcesRequest{Format: "android-strings-xml", Filename: "strings.xml", Content: `<resources><string`},
		},
		{
			name:    "unknown JSON schema",
			request: sdk.InspectResourcesRequest{Format: "carvion-json-v1", Filename: "resources.json", Content: `{"schema":"other","resources":[]}`},
		},
		{
			name:    "unsupported format",
			request: sdk.InspectResourcesRequest{Format: "raw-firmware", Filename: "data.bin", Content: "test"},
		},
		{
			name:    "path is not accepted",
			request: sdk.InspectResourcesRequest{Format: "android-strings-xml", Filename: "../strings.xml", Content: `<resources/>`},
		},
		{
			name:    "oversized resource",
			request: sdk.InspectResourcesRequest{Format: "android-strings-xml", Filename: "strings.xml", Content: strings.Repeat("x", research.MaxResourceContentBytes+1)},
		},
		{
			name:    "invalid UTF-8",
			request: sdk.InspectResourcesRequest{Format: "android-strings-xml", Filename: "strings.xml", Content: string([]byte{0xff})},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := adapter.InspectResources(context.Background(), test.request); err == nil {
				t.Fatal("InspectResources() error = nil, want validation failure")
			}
		})
	}
}

func TestAdapterOperationsRespectCanceledContext(t *testing.T) {
	adapter := newTestAdapter(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.GetInfo(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetInfo() error = %v, want context.Canceled", err)
	}
	if _, err := adapter.Detect(ctx, sdk.DetectRequest{Observation: testProfileJSON(t)}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Detect() error = %v, want context.Canceled", err)
	}
	if _, err := adapter.InspectResources(ctx, sdk.InspectResourcesRequest{Format: "android-strings-xml", Filename: "strings.xml", Content: `<resources/>`}); !errors.Is(err, context.Canceled) {
		t.Fatalf("InspectResources() error = %v, want context.Canceled", err)
	}
}

func newTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	adapter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return adapter
}

func testProfileJSON(t *testing.T) []byte {
	t.Helper()
	profile := research.Profile{
		Schema:    research.ProfileSchema,
		ProfileID: "test-profile",
		Vehicle: research.VehicleProfile{
			Manufacturer: "TEST ONLY",
			Model:        "TEST MODEL",
			ModelYear:    2023,
			Market:       "TEST MARKET",
			Trim:         "TEST TRIM",
		},
		Infotainment: research.InfotainmentProfile{
			Platform:           "TEST PLATFORM",
			OSName:             "TEST OS",
			OSVersion:          "TEST OS VERSION",
			HardwareIdentifier: "TEST HARDWARE",
			HardwareRevision:   "TEST REVISION",
			FirmwareVersion:    "TEST FIRMWARE VERSION",
			FirmwareBuildID:    "TEST BUILD ID",
		},
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func requestFromProfile(profile research.Profile, profileJSON []byte) sdk.CompatibilityRequest {
	return sdk.CompatibilityRequest{
		Identity: sdk.VehicleIdentity{
			Manufacturer: profile.Vehicle.Manufacturer,
			Model:        profile.Vehicle.Model,
			PlatformID:   profile.Infotainment.Platform,
		},
		Hardware:       sdk.HardwareInfo{PlatformID: profile.Infotainment.HardwareIdentifier, Revision: profile.Infotainment.HardwareRevision},
		Firmware:       sdk.FirmwareInfo{BuildID: profile.Infotainment.FirmwareBuildID, Version: profile.Infotainment.FirmwareVersion},
		VehicleProfile: json.RawMessage(profileJSON),
	}
}

func containsInput(values []sdk.AdapterInputType, target sdk.AdapterInputType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsFormat(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func equalResources(left, right []sdk.TranslatableResource) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
