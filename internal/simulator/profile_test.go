package simulator

import (
	"bytes"
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestProfilesContainExplicitSyntheticDemoScenarios(t *testing.T) {
	profiles, err := Profiles()
	if err != nil {
		t.Fatalf("Profiles() error = %v", err)
	}
	wantIDs := []string{
		"demo-android-platform-a",
		"demo-android-platform-b",
		"unknown-firmware",
		"incompatible-hardware",
		"simulated-communication-failure",
	}
	if len(profiles) != len(wantIDs) {
		t.Fatalf("Profiles() returned %d profiles, want %d: %+v", len(profiles), len(wantIDs), profiles)
	}
	gotIDs := make([]string, len(profiles))
	for i, profile := range profiles {
		gotIDs[i] = profile.ID
		if !profile.Synthetic || profile.Name == "" || profile.Description == "" {
			t.Errorf("profile %q must have a name, description, and synthetic marker: %+v", profile.ID, profile)
		}
		if profile.Identity.Manufacturer == "" || profile.Identity.Model == "" ||
			profile.Identity.PlatformID == "" || profile.Hardware.Revision == "" ||
			profile.Firmware.BuildID == "" || profile.Firmware.Version == "" {
			t.Errorf("profile %q is missing test device identity, hardware, or firmware: %+v", profile.ID, profile)
		}
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("profile IDs = %v, want stable order %v", gotIDs, wantIDs)
	}
}

func TestLoadProfileReturnsIsolatedSyntheticResourceFixture(t *testing.T) {
	profile, err := LoadProfile("demo-android-platform-a")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	first := profile.ResourceXML()
	if len(first) == 0 || !utf8.Valid(first) {
		t.Fatalf("ResourceXML() must return non-empty UTF-8, got %q", first)
	}
	for _, expected := range [][]byte{[]byte("<resources"), []byte("zh-CN"), []byte("你好"), []byte("welcome_title")} {
		if !containsBytes(first, expected) {
			t.Errorf("resource fixture %q does not contain %q", first, expected)
		}
	}
	first[0] = 'x'
	second := profile.ResourceXML()
	if len(second) == 0 || second[0] != '<' {
		t.Fatal("ResourceXML() exposed mutable shared fixture bytes")
	}
}

func TestCompatibilityOutcomesAreBoundToExactSyntheticProfiles(t *testing.T) {
	profiles, err := Profiles()
	if err != nil {
		t.Fatalf("Profiles() error = %v", err)
	}
	byID := make(map[string]domain.CompatibilityStatus, len(profiles))
	for _, profile := range profiles {
		byID[profile.ID] = profile.ExpectedCompatibility
	}
	want := map[string]domain.CompatibilityStatus{
		"demo-android-platform-a":         domain.StatusSupported,
		"demo-android-platform-b":         domain.StatusSupported,
		"unknown-firmware":                domain.StatusUnknown,
		"incompatible-hardware":           domain.StatusUnsupported,
		"simulated-communication-failure": domain.StatusUnknown,
	}
	if !reflect.DeepEqual(byID, want) {
		t.Fatalf("expected profile outcomes = %v, want %v", byID, want)
	}
	if _, err := LoadProfile("not-a-demo-profile"); err == nil {
		t.Fatal("LoadProfile() accepted a profile outside the local synthetic catalog")
	}
}

func containsBytes(content, search []byte) bool {
	return bytes.Contains(content, search)
}
