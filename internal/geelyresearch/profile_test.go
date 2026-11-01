package geelyresearch

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestParseProfile(t *testing.T) {
	profile := testProfile()
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseProfile(data)
	if err != nil {
		t.Fatalf("ParseProfile() error = %v", err)
	}
	if !reflect.DeepEqual(got, profile) {
		t.Fatalf("ParseProfile() = %#v, want %#v", got, profile)
	}
}

func TestParseProfileRejectsInvalidInput(t *testing.T) {
	profile := testProfile()
	valid, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{name: "invalid UTF-8", data: append(append([]byte(nil), valid...), 0xff)},
		{name: "unknown VIN field", data: []byte(strings.Replace(string(valid), `"profile_id":"test-profile"`, `"profile_id":"test-profile","vin":"must-not-be-accepted"`, 1))},
		{name: "multiple JSON values", data: append(append([]byte(nil), valid...), valid...)},
		{name: "wrong schema", data: []byte(strings.Replace(string(valid), ProfileSchema, "carvion.geely-research/v99", 1))},
		{name: "duplicate profile field", data: []byte(strings.Replace(string(valid), `"profile_id":"test-profile"`, `"profile_id":"test-profile","profile_id":"other-profile"`, 1))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseProfile(test.data); err == nil {
				t.Fatal("ParseProfile() succeeded; want validation error")
			}
		})
	}
}

func TestParseProfileKeepsMissingObservationsUnknown(t *testing.T) {
	profile := testProfile()
	profile.Infotainment.FirmwareBuildID = ""
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseProfile(data)
	if err != nil {
		t.Fatalf("ParseProfile() error = %v", err)
	}
	if got.Infotainment.FirmwareBuildID != "" {
		t.Fatalf("missing firmware build ID = %q, want unknown/empty", got.Infotainment.FirmwareBuildID)
	}
}

func TestParseProfileMarksImportedValuesUnverified(t *testing.T) {
	data, err := json.Marshal(testProfile())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseProfile(data)
	if err != nil {
		t.Fatalf("ParseProfile() error = %v", err)
	}
	if got.Origin != ProfileOriginImportedFile {
		t.Fatalf("profile origin = %q, want imported and unverified", got.Origin)
	}
}

func TestParseProfileRejectsOversizeInput(t *testing.T) {
	data := append([]byte(`{"schema":"`+ProfileSchema+`","profile_id":"test-profile"}`), make([]byte, maxProfileBytes)...)
	if _, err := ParseProfile(data); err == nil {
		t.Fatal("ParseProfile() succeeded for an oversized profile")
	}
}

func TestParseProfileRejectsVehicleIdentifiersAndCredentialMaterial(t *testing.T) {
	vinLikeTestValue := "ABCDEFGHJKLMNPRST"
	for _, test := range []struct {
		name   string
		mutate func(*Profile)
	}{
		{name: "identifier in profile ID", mutate: func(profile *Profile) { profile.ProfileID = "abcdefghjklmnprst" }},
		{name: "identifier in citation ID", mutate: func(profile *Profile) { profile.Sources[0].ID = "abcdefghjklmnprst" }},
		{name: "vehicle identifier", mutate: func(profile *Profile) { profile.Vehicle.Model = vinLikeTestValue }},
		{name: "credential assignment", mutate: func(profile *Profile) { profile.Infotainment.OSVersion = "password=TESTTOKEN" }},
		{name: "credential in citation", mutate: func(profile *Profile) { profile.Sources[0].Scope = "authorization: Bearer TESTTOKEN" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			profile := testProfile()
			test.mutate(&profile)
			data, err := json.Marshal(profile)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseProfile(data); err == nil {
				t.Fatal("ParseProfile() accepted identifier or credential-like text")
			}
		})
	}
}

func TestMaximumProfileAndResourceInputsFitRPCEnvelope(t *testing.T) {
	profilePrefix, profileSuffix := []byte(`{"large":"`), []byte(`"}`)
	profileJSON := append(append(profilePrefix, bytesOf('<', maxProfileBytes-len(profilePrefix)-len(profileSuffix))...), profileSuffix...)
	compatibilityParams, err := json.Marshal(sdk.CompatibilityRequest{VehicleProfile: json.RawMessage(profileJSON)})
	if err != nil {
		t.Fatal(err)
	}
	compatibilityFrame, err := json.Marshal(sdk.RPCRequest{JSONRPC: "2.0", ID: "1", Method: "CheckCompatibility", Params: compatibilityParams})
	if err != nil {
		t.Fatal(err)
	}
	if len(compatibilityFrame)+1 > sdk.MaxRPCMessageBytes {
		t.Fatalf("maximum profile RPC frame = %d bytes, limit = %d", len(compatibilityFrame)+1, sdk.MaxRPCMessageBytes)
	}

	resourceParams, err := json.Marshal(sdk.InspectResourcesRequest{
		Format: "android-strings-xml", Filename: "strings.xml", Content: strings.Repeat("<", MaxResourceContentBytes),
	})
	if err != nil {
		t.Fatal(err)
	}
	resourceFrame, err := json.Marshal(sdk.RPCRequest{JSONRPC: "2.0", ID: "1", Method: "InspectResources", Params: resourceParams})
	if err != nil {
		t.Fatal(err)
	}
	if len(resourceFrame)+1 > sdk.MaxRPCMessageBytes {
		t.Fatalf("maximum resource RPC frame = %d bytes, limit = %d", len(resourceFrame)+1, sdk.MaxRPCMessageBytes)
	}
}

func bytesOf(value byte, count int) []byte {
	return []byte(strings.Repeat(string(value), count))
}

func testProfile() Profile {
	return Profile{
		Schema:    ProfileSchema,
		ProfileID: "test-profile",
		Origin:    ProfileOriginImportedFile,
		Vehicle: VehicleProfile{
			Manufacturer: "TEST ONLY",
			Model:        "TEST MODEL",
			ModelYear:    2023,
			Market:       "TEST MARKET",
			Trim:         "TEST TRIM",
		},
		Infotainment: InfotainmentProfile{
			Platform:           "TEST PLATFORM",
			OSName:             "TEST OS",
			OSVersion:          "TEST OS VERSION",
			HardwareIdentifier: "TEST HARDWARE",
			HardwareRevision:   "TEST REVISION",
			FirmwareVersion:    "TEST FIRMWARE VERSION",
			FirmwareBuildID:    "TEST BUILD ID",
		},
		Sources: []ProfileSource{{
			ID:          "user-source-1",
			Title:       "User-supplied test citation",
			Publisher:   "Test publisher",
			URL:         "https://example.invalid/test",
			PublishedAt: "2023-01-01",
			RetrievedAt: "2026-11-01",
			Scope:       "Test-only profile metadata; not verified evidence.",
		}},
	}
}
