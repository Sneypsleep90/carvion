package geelyresearch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestBuildDiagnosticReportOmitsPathsAndResourceContents(t *testing.T) {
	profileBytes, err := json.Marshal(testProfile())
	if err != nil {
		t.Fatal(err)
	}
	resourceContent := []byte("private test resource contents")
	generatedAt := time.Date(2026, 11, 1, 12, 30, 0, 0, time.UTC)

	report, data, err := BuildDiagnosticReport(
		profileBytes,
		CompatibilityResult{Status: sdk.StatusUnknown, Reason: "no exact test record"},
		"/Users/test-user/private-device/strings.xml",
		"android-strings-xml",
		resourceContent,
		[]string{"resource.one"},
		generatedAt,
	)
	if err != nil {
		t.Fatalf("BuildDiagnosticReport() error = %v", err)
	}
	if report.Resource == nil || report.Resource.Filename != "strings.xml" {
		t.Fatalf("report resource filename = %+v, want sanitized basename", report.Resource)
	}
	if !bytes.Contains(data, []byte(report.Schema)) || !bytes.Contains(data, []byte("resource.one")) {
		t.Fatalf("report JSON is missing schema or resource keys: %s", data)
	}
	if !bytes.Contains(data, []byte(`"origin": "imported-file-unverified"`)) {
		t.Fatalf("report JSON is missing file provenance: %s", data)
	}
	for _, secret := range []string{"/Users/test-user", "private-device", string(resourceContent), "vin"} {
		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(secret)) {
			t.Errorf("report contains excluded data %q: %s", secret, data)
		}
	}
}

func TestBuildDiagnosticReportRejectsUnsafeInput(t *testing.T) {
	status := CompatibilityResult{Status: sdk.StatusSupported, Reason: "test only"}

	profileBytes, marshalErr := json.Marshal(testProfile())
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	_, _, err := BuildDiagnosticReport(profileBytes, status, "../", "unknown-format", []byte("source"), nil, time.Time{})
	if err == nil {
		t.Fatal("BuildDiagnosticReport() succeeded with invalid file data")
	}

	for _, test := range []struct {
		name        string
		reason      string
		filename    string
		resourceKey string
	}{
		{name: "identifier in filename", reason: "test only", filename: "ABCDEFGHJKLMNPRST.xml", resourceKey: "safe.key"},
		{name: "credential in compatibility reason", reason: "token=TESTSECRET", filename: "strings.xml", resourceKey: "safe.key"},
		{name: "identifier in resource key", reason: "test only", filename: "strings.xml", resourceKey: "ABCDEFGHJKLMNPRST"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := BuildDiagnosticReport(profileBytes, CompatibilityResult{Status: sdk.StatusUnknown, Reason: test.reason}, test.filename, "android-strings-xml", []byte("source"), []string{test.resourceKey}, time.Time{})
			if err == nil {
				t.Fatal("BuildDiagnosticReport() accepted a diagnostic field with sensitive material")
			}
		})
	}
}
