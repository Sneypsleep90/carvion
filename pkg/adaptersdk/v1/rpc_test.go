package v1

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type testAdapter struct {
	getInfoErr       error
	invalidDetection bool
	sdkVersion       string
	vehicleProfile   string
}

func (a testAdapter) GetInfo(context.Context) (AdapterManifest, error) {
	if a.getInfoErr != nil {
		return AdapterManifest{}, a.getInfoErr
	}
	sdkVersion := a.sdkVersion
	if sdkVersion == "" {
		sdkVersion = SDKVersion
	}
	return AdapterManifest{
		ManifestVersion: ManifestVersion,
		APIVersion:      APIVersion,
		SDKVersion:      sdkVersion,
		ID:              "test-adapter",
		Name:            "Test Adapter",
		Version:         "1.0.0",
		Executable:      "bin/test-adapter",
		Capabilities: AdapterCapabilities{
			Detect:             true,
			CheckCompatibility: true,
			InspectResources:   true,
		},
		InputTypes: []AdapterInputType{
			InputVehicleObservationJSON,
			InputCompatibilityProfile,
			InputResourceDocument,
		},
		ResourceFormats: []string{"test-xml"},
	}, nil
}

func (a testAdapter) manifest() AdapterManifest {
	manifest, _ := a.GetInfo(context.Background())
	return manifest
}

func TestServeRejectsManifestFromUnsupportedSDKMajor(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"1","method":"GetInfo","params":{}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{sdkVersion: "2.0.0"}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != CodeAdapterFailure {
		t.Fatalf("response error = %+v, want adapter failure for unsupported SDK major", response.Error)
	}
}

func TestValidateManifestRequiresInputsAndResourceFormats(t *testing.T) {
	manifest := testAdapter{}.manifest()
	manifest.InputTypes = []AdapterInputType{InputVehicleObservationJSON, InputCompatibilityProfile}
	if err := ValidateManifest(manifest); err == nil || !strings.Contains(err.Error(), "resource document input") {
		t.Fatalf("ValidateManifest() error = %v, want required resource document input", err)
	}

	manifest = testAdapter{}.manifest()
	manifest.ResourceFormats = []string{}
	if err := ValidateManifest(manifest); err == nil || !strings.Contains(err.Error(), "resource format") {
		t.Fatalf("ValidateManifest() error = %v, want required resource format", err)
	}
}

func (a testAdapter) Detect(_ context.Context, request DetectRequest) (DetectionResult, error) {
	if string(request.Observation) != `{"source":"synthetic"}` {
		return DetectionResult{}, errors.New("observation was not passed to adapter")
	}
	if a.invalidDetection {
		return DetectionResult{Status: "invalid"}, nil
	}
	return DetectionResult{
		Status: StatusSupported,
		Identity: VehicleIdentity{
			Manufacturer: "Test Lab",
			Model:        "Test Unit",
			PlatformID:   "test-platform",
		},
	}, nil
}

func (a testAdapter) CheckCompatibility(_ context.Context, request CompatibilityRequest) (CompatibilityResult, error) {
	if request.Firmware.BuildID != "build-a" {
		return CompatibilityResult{}, errors.New("firmware request was not passed to adapter")
	}
	if a.vehicleProfile != "" && string(request.VehicleProfile) != a.vehicleProfile {
		return CompatibilityResult{}, errors.New("vehicle profile was not passed to adapter")
	}
	return CompatibilityResult{Status: StatusSupported}, nil
}

func (testAdapter) InspectResources(_ context.Context, request InspectResourcesRequest) (ResourceInspection, error) {
	if request.Format != "test-xml" || !strings.Contains(request.Content, "hello") {
		return ResourceInspection{}, errors.New("resource contents were not passed to adapter")
	}
	return ResourceInspection{Resources: []TranslatableResource{{ID: "title", Text: "Hello"}}}, nil
}

func TestServeDispatchesGetInfo(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"1","method":"GetInfo","params":{}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.JSONRPC != "2.0" || response.ID != "1" || response.Error != nil {
		t.Fatalf("response = %+v, want successful JSON-RPC response", response)
	}
	var manifest AdapterManifest
	if err := json.Unmarshal(response.Result, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.ID != "test-adapter" || manifest.APIVersion != APIVersion {
		t.Errorf("manifest = %+v, want test-adapter with API %q", manifest, APIVersion)
	}
}

func TestServeDispatchesDetect(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"5","method":"Detect","params":{"observation":{"source":"synthetic"}}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var result DetectionResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode detection result: %v", err)
	}
	if result.Status != StatusSupported || result.Identity.Model != "Test Unit" {
		t.Errorf("detection result = %+v, want supported Test Unit", result)
	}
}

func TestServeRejectsInvalidDetectionStatus(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"8","method":"Detect","params":{"observation":{"source":"synthetic"}}}` + "\n"
	adapter := testAdapter{invalidDetection: true}

	if err := Serve(context.Background(), strings.NewReader(request), &output, adapter); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != CodeAdapterFailure {
		t.Fatalf("response error = %+v, want adapter-failure", response.Error)
	}
}

func TestServeDispatchesCheckCompatibility(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"6","method":"CheckCompatibility","params":{"identity":{"manufacturer":"Test Lab","model":"Test Unit","platform_id":"test-platform"},"hardware":{"platform_id":"test-platform","revision":"rev-a"},"firmware":{"build_id":"build-a","version":"1.0.0"}}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var result CompatibilityResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode compatibility result: %v", err)
	}
	if result.Status != StatusSupported {
		t.Errorf("compatibility result = %+v, want supported", result)
	}
}

func TestServeDispatchesCompatibilityVehicleProfile(t *testing.T) {
	var output strings.Builder
	const profile = `{"schema":"carvion.geely-research/v1","profile_id":"test-profile"}`
	request := `{"jsonrpc":"2.0","id":"14","method":"CheckCompatibility","params":{"firmware":{"build_id":"build-a"},"vehicle_profile":` + profile + `}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{vehicleProfile: profile}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("response error = %+v, want successful vehicle profile dispatch", response.Error)
	}
	var result CompatibilityResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode compatibility result: %v", err)
	}
	if result.Status != StatusSupported {
		t.Fatalf("compatibility result = %+v, want supported", result)
	}
}

func TestServeDispatchesInspectResources(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"7","method":"InspectResources","params":{"format":"test-xml","filename":"strings.xml","content":"<resources>hello</resources>"}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var result ResourceInspection
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode resource result: %v", err)
	}
	if len(result.Resources) != 1 || result.Resources[0] != (TranslatableResource{ID: "title", Text: "Hello"}) {
		t.Errorf("resource result = %+v, want title=Hello", result)
	}
}

func TestServeRejectsUnknownMethod(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"2","method":"DeleteVehicle","params":{}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != CodeMethodNotFound {
		t.Fatalf("response error = %+v, want method-not-found", response.Error)
	}
}

func TestServeRejectsInvalidParameters(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"3","method":"CheckCompatibility","params":{"extra":true}}` + "\n"

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != CodeInvalidParams {
		t.Fatalf("response error = %+v, want invalid-params", response.Error)
	}
}

func TestServeReturnsAdapterErrorAsRPCError(t *testing.T) {
	var output strings.Builder
	request := `{"jsonrpc":"2.0","id":"4","method":"GetInfo","params":{}}` + "\n"
	adapterErr := errors.New("adapter unavailable")

	if err := Serve(context.Background(), strings.NewReader(request), &output, testAdapter{getInfoErr: adapterErr}); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	var response RPCResponse
	if err := json.Unmarshal([]byte(output.String()), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != CodeAdapterFailure || response.Error.Message != adapterErr.Error() {
		t.Fatalf("response error = %+v, want adapter failure", response.Error)
	}
}
