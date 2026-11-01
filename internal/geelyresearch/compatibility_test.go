package geelyresearch

import (
	"testing"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestCheckCompatibilityRequiresExactFullProfile(t *testing.T) {
	profile := testProfile()
	record := CompatibilityRecord{
		ID:         "test-only-record",
		Key:        profile.CompatibilityKey(),
		Status:     sdk.StatusSupported,
		Reason:     "synthetic unit-test record only",
		Evidence:   []string{"test-source"},
		ReviewedAt: "2026-11-01",
	}

	if got := CheckCompatibility(profile, []CompatibilityRecord{record}); got.Status != sdk.StatusSupported {
		t.Fatalf("exact match status = %q, want supported; reason=%q", got.Status, got.Reason)
	}

	mutations := map[string]func(*Profile){
		"manufacturer":        func(p *Profile) { p.Vehicle.Manufacturer += " changed" },
		"model":               func(p *Profile) { p.Vehicle.Model += " changed" },
		"model year":          func(p *Profile) { p.Vehicle.ModelYear++ },
		"market":              func(p *Profile) { p.Vehicle.Market += " changed" },
		"trim":                func(p *Profile) { p.Vehicle.Trim += " changed" },
		"platform":            func(p *Profile) { p.Infotainment.Platform += " changed" },
		"OS name":             func(p *Profile) { p.Infotainment.OSName += " changed" },
		"OS version":          func(p *Profile) { p.Infotainment.OSVersion += " changed" },
		"hardware identifier": func(p *Profile) { p.Infotainment.HardwareIdentifier += " changed" },
		"hardware revision":   func(p *Profile) { p.Infotainment.HardwareRevision += " changed" },
		"firmware version":    func(p *Profile) { p.Infotainment.FirmwareVersion += " changed" },
		"firmware build":      func(p *Profile) { p.Infotainment.FirmwareBuildID += " changed" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := profile
			mutate(&changed)
			if got := CheckCompatibility(changed, []CompatibilityRecord{record}); got.Status != sdk.StatusUnknown {
				t.Fatalf("changed profile status = %q, want unknown", got.Status)
			}
		})
	}
}

func TestCheckCompatibilityReturnsUnknownWithoutRuleOrRequiredObservation(t *testing.T) {
	profile := testProfile()

	if got := CheckCompatibility(profile, nil); got.Status != sdk.StatusUnknown {
		t.Fatalf("empty catalog status = %q, want unknown", got.Status)
	}

	record := CompatibilityRecord{
		ID:         "test-only-record",
		Key:        profile.CompatibilityKey(),
		Status:     sdk.StatusSupported,
		Reason:     "synthetic unit-test record only",
		Evidence:   []string{"test-source"},
		ReviewedAt: "2026-11-01",
	}
	profile.Infotainment.OSVersion = ""
	if got := CheckCompatibility(profile, []CompatibilityRecord{record}); got.Status != sdk.StatusUnknown {
		t.Fatalf("incomplete profile status = %q, want unknown", got.Status)
	}
}

func TestCheckCompatibilityOnlyUsesExplicitExactExclusion(t *testing.T) {
	profile := testProfile()
	record := CompatibilityRecord{
		ID:         "test-only-exclusion",
		Key:        profile.CompatibilityKey(),
		Status:     sdk.StatusUnsupported,
		Reason:     "synthetic unit-test exclusion only",
		Evidence:   []string{"test-source"},
		ReviewedAt: "2026-11-01",
	}

	if got := CheckCompatibility(profile, []CompatibilityRecord{record}); got.Status != sdk.StatusUnsupported {
		t.Fatalf("exact exclusion status = %q, want unsupported; reason=%q", got.Status, got.Reason)
	}

	profile.Infotainment.FirmwareBuildID = "different test build"
	if got := CheckCompatibility(profile, []CompatibilityRecord{record}); got.Status != sdk.StatusUnknown {
		t.Fatalf("non-matching profile status = %q, want unknown", got.Status)
	}
}

func TestLoadCompatibilityRecordsStartsEmpty(t *testing.T) {
	records, err := LoadCompatibilityRecords()
	if err != nil {
		t.Fatalf("LoadCompatibilityRecords() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("initial compatibility record count = %d, want 0", len(records))
	}
}

func TestValidateCompatibilityRecordRequiresEvidenceForExactTuple(t *testing.T) {
	profile := testProfile()
	record := CompatibilityRecord{
		ID:         "test-only-record",
		Key:        profile.CompatibilityKey(),
		Status:     sdk.StatusSupported,
		Reason:     "synthetic unit-test record only",
		Evidence:   []string{"geely-xingyue-l-monjaro-naming-2026"},
		ReviewedAt: "2026-11-01",
	}

	if err := validateRecord(record, evidenceIDs()); err == nil {
		t.Fatal("validateRecord() accepted broad naming evidence that does not establish an exact compatibility tuple")
	}
}
