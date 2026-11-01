package pluginmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestDiscoverRegistersValidAdapterAndChecksSDKInfo(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("test-plugin")
	writePlugin(t, root, "test-plugin", manifest, staticInfoPlugin(t, manifest))
	var logs bytes.Buffer

	manager, err := Discover(root, 5*time.Second, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	got := manager.List()
	if len(got) != 1 || got[0].ID != manifest.ID || got[0].APIVersion != sdk.APIVersion {
		t.Fatalf("List() = %+v, want one registered test plugin; discovery log: %s", got, logs.String())
	}

	info, err := manager.GetInfo(context.Background(), manifest.ID)
	if err != nil {
		t.Fatalf("GetInfo() error = %v", err)
	}
	if info.ID != manifest.ID || info.Version != manifest.Version {
		t.Errorf("GetInfo() = %+v, want manifest %+v", info, manifest)
	}
}

func TestDiscoverSkipsMalformedAndIncompatibleManifests(t *testing.T) {
	root := t.TempDir()
	writeRawPlugin(t, root, "bad-json", "{not json", "")

	incompatible := validManifest("wrong-api")
	incompatible.APIVersion = "v99"
	writePlugin(t, root, "wrong-api", incompatible, staticInfoPlugin(t, incompatible))

	escaping := validManifest("path-escape")
	escaping.Executable = "../outside"
	writePlugin(t, root, "path-escape", escaping, "")

	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if got := manager.List(); len(got) != 0 {
		t.Fatalf("List() = %+v, want no invalid adapters registered", got)
	}
}

func TestDiscoverRegistersManifestWithSDKInputAndResourceMetadata(t *testing.T) {
	root := t.TempDir()
	manifest := versionedManifestJSON
	response := `{"jsonrpc":"2.0","id":"1","result":` + manifest + `}`
	script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '" + response + "'\n"
	writeRawPlugin(t, root, "metadata-plugin", manifest, script)

	manager, err := Discover(root, time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if got := manager.List(); len(got) != 1 {
		t.Fatalf("List() = %+v, want registered versioned adapter", got)
	}
	encoded, err := json.Marshal(manager.List()[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"sdk_version":"1.0.0"`, `"input_types"`, `"resource_formats"`, `"status":"available"`} {
		if !strings.Contains(string(encoded), field) {
			t.Errorf("adapter summary %s is missing %s", encoded, field)
		}
	}
}

func TestReadManifestRejectsUnsupportedSDKMajor(t *testing.T) {
	root := t.TempDir()
	manifest := strings.Replace(versionedManifestJSON, `"sdk_version":"1.0.0"`, `"sdk_version":"2.0.0"`, 1)
	writeRawPlugin(t, root, "future-sdk", manifest, "")

	_, err := readManifest(filepath.Join(root, "future-sdk"))
	if err == nil || !strings.Contains(err.Error(), "SDK major") {
		t.Fatalf("readManifest() error = %v, want unsupported SDK major", err)
	}
}

func TestReadManifestRejectsMalformedVersionedContract(t *testing.T) {
	for _, test := range []struct {
		name     string
		manifest string
	}{
		{name: "unsupported schema", manifest: strings.Replace(versionedManifestJSON, `"manifest_version":2`, `"manifest_version":1`, 1)},
		{name: "missing SDK version", manifest: strings.Replace(versionedManifestJSON, `"sdk_version":"1.0.0",`, "", 1)},
		{name: "missing input declarations", manifest: strings.Replace(versionedManifestJSON, `"input_types":["vehicle-observation-json","compatibility-profile","resource-document"],`, "", 1)},
		{name: "invalid resource format", manifest: strings.Replace(versionedManifestJSON, `"android-strings-xml"`, `"Android XML"`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeRawPlugin(t, root, "malformed", test.manifest, "")
			if _, err := readManifest(filepath.Join(root, "malformed")); err == nil {
				t.Fatal("readManifest() error = nil, want manifest contract validation error")
			}
		})
	}
}

func TestManagerRejectsResourceFormatNotDeclaredByAdapter(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("format-plugin")
	writePlugin(t, root, "format-plugin", manifest, staticInfoPlugin(t, manifest))
	manager, err := Discover(root, time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	_, err = manager.InspectResources(context.Background(), manifest.ID, sdk.InspectResourcesRequest{
		Format: "unknown-xml",
	})
	if err == nil || !strings.Contains(err.Error(), "resource format") {
		t.Fatalf("InspectResources() error = %v, want undeclared resource format", err)
	}
}

func TestManagerMarksMalformedAdapterResponseUnavailable(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("malformed-plugin")
	script := "#!/bin/sh\nread request\ncase \"$request\" in\n" +
		"  *'\"method\":\"GetInfo\"'*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n" +
		"  *) printf '%s\\n' '{not-json' ;;\nesac\n"
	writePlugin(t, root, "malformed-plugin", manifest, script)
	manager, err := Discover(root, time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if _, err := manager.Detect(context.Background(), manifest.ID, sdk.DetectRequest{Observation: json.RawMessage(`{}`)}); !errors.Is(err, ErrPluginUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrPluginUnavailable", err)
	}

	encoded, err := json.Marshal(manager.List()[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"status":"unavailable"`) {
		t.Fatalf("adapter summary = %s, want unavailable status", encoded)
	}
}

func TestManagerRejectsMalformedOperationResults(t *testing.T) {
	for _, test := range []struct {
		name       string
		method     string
		resultJSON string
	}{
		{name: "invalid compatibility state", method: "CheckCompatibility", resultJSON: `{"status":"pending","reason":"not a contract state"}`},
		{name: "unknown result field", method: "CheckCompatibility", resultJSON: `{"status":"unknown","reason":"no profile","extra":true}`},
		{name: "missing resource array", method: "InspectResources", resultJSON: `{"resources":null}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			manifest := validManifest("invalid-result")
			result := rpcResultResponse(t, json.RawMessage(test.resultJSON))
			script := "#!/bin/sh\nread request\ncase \"$request\" in\n" +
				"  *GetInfo*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n" +
				"  *) printf '%s\\n' '" + result + "' ;;\nesac\n"
			writePlugin(t, root, manifest.ID, manifest, script)
			manager, err := Discover(root, time.Second, quietLogger())
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			var callErr error
			switch test.method {
			case "CheckCompatibility":
				_, callErr = manager.CheckCompatibility(context.Background(), manifest.ID, sdk.CompatibilityRequest{})
			case "InspectResources":
				_, callErr = manager.InspectResources(context.Background(), manifest.ID, sdk.InspectResourcesRequest{Format: "android-strings-xml"})
			}
			if !errors.Is(callErr, ErrPluginUnavailable) {
				t.Fatalf("adapter call error = %v, want ErrPluginUnavailable", callErr)
			}
			if got := manager.List()[0].Status; got != AdapterStatusUnavailable {
				t.Errorf("adapter status = %q, want unavailable", got)
			}
		})
	}
}

const versionedManifestJSON = `{"manifest_version":2,"api_version":"v1","sdk_version":"1.0.0","id":"metadata-plugin","name":"Metadata Test Adapter","version":"1.2.3","executable":"bin/plugin","capabilities":{"detect":true,"check_compatibility":true,"inspect_resources":true},"input_types":["vehicle-observation-json","compatibility-profile","resource-document"],"resource_formats":["android-strings-xml"]}`

func TestDiscoverRejectsDuplicateAdapterIDs(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("duplicate")
	writePlugin(t, root, "a-first", manifest, staticInfoPlugin(t, manifest))
	writePlugin(t, root, "b-second", manifest, staticInfoPlugin(t, manifest))

	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if got := manager.List(); len(got) != 1 {
		t.Fatalf("List() count = %d, want 1 after duplicate rejection", len(got))
	}
}

func TestManagerReturnsPluginRPCError(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("rpc-error")
	script := "#!/bin/sh\nread request\ncase \"$request\" in\n  *'\"method\":\"GetInfo\"'*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n  *) printf '%s\\n' '{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"error\":{\"code\":-32000,\"message\":\"synthetic failure\"}}' ;;\nesac\n"
	writePlugin(t, root, "rpc-error", manifest, script)
	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	_, err = manager.Detect(context.Background(), manifest.ID, sdk.DetectRequest{
		Observation: json.RawMessage(`{"manufacturer":"test"}`),
	})
	if !errors.Is(err, ErrPluginUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrPluginUnavailable", err)
	}
}

func TestManagerRoutesCallsToTheSelectedAdapterProcess(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"adapter-one", "adapter-two"} {
		manifest := validManifest(id)
		result := rpcResultResponse(t, sdk.CompatibilityResult{Status: sdk.StatusSupported, Reason: id})
		script := "#!/bin/sh\nread request\ncase \"$request\" in\n" +
			"  *GetInfo*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n" +
			"  *) printf '%s\\n' '" + result + "' ;;\nesac\n"
		writePlugin(t, root, id, manifest, script)
	}
	manager, err := Discover(root, time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	for _, id := range []string{"adapter-one", "adapter-two"} {
		got, err := manager.CheckCompatibility(context.Background(), id, sdk.CompatibilityRequest{})
		if err != nil {
			t.Fatalf("CheckCompatibility(%q) error = %v", id, err)
		}
		if got.Reason != id {
			t.Errorf("CheckCompatibility(%q) reason = %q, want selected process %q", id, got.Reason, id)
		}
	}
}

func TestManagerParentCancellationDoesNotMarkAdapterUnavailable(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("cancelled-plugin")
	script := "#!/bin/sh\nread request\ncase \"$request\" in\n" +
		"  *GetInfo*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n" +
		"  *) exec /bin/sleep 5 ;;\nesac\n"
	writePlugin(t, root, "cancelled-plugin", manifest, script)
	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = manager.CheckCompatibility(ctx, manifest.ID, sdk.CompatibilityRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CheckCompatibility() error = %v, want parent context deadline", err)
	}
	if got := manager.List()[0].Status; got != AdapterStatusAvailable {
		t.Errorf("adapter status = %q, want available after caller cancellation", got)
	}
}

func TestManagerTimesOutAndTerminatesPlugin(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("slow-plugin")
	script := "#!/bin/sh\nread request\ncase \"$request\" in\n  *'\"method\":\"GetInfo\"'*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n  *) exec /bin/sleep 5 ;;\nesac\n"
	writePlugin(t, root, "slow-plugin", manifest, script)
	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	manager.timeout = 100 * time.Millisecond

	start := time.Now()
	_, err = manager.Detect(context.Background(), manifest.ID, sdk.DetectRequest{
		Observation: json.RawMessage(`{"manufacturer":"test"}`),
	})
	if !errors.Is(err, ErrPluginTimeout) {
		t.Fatalf("Detect() error = %v, want ErrPluginTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Detect() took %s, want under one second", elapsed)
	}
	if got := manager.List()[0].Status; got != AdapterStatusUnavailable {
		t.Errorf("adapter status = %q, want %q after timeout", got, AdapterStatusUnavailable)
	}
}

func TestManagerHandlesProcessExitWithoutRPCResponse(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("exiting-plugin")
	script := "#!/bin/sh\nread request\ncase \"$request\" in\n  *'\"method\":\"GetInfo\"'*) printf '%s\\n' '" + responseForManifest(t, manifest) + "' ;;\n  *) exit 9 ;;\nesac\n"
	writePlugin(t, root, "exiting-plugin", manifest, script)
	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	_, err = manager.Detect(context.Background(), manifest.ID, sdk.DetectRequest{
		Observation: json.RawMessage(`{"manufacturer":"test"}`),
	})
	if !errors.Is(err, ErrPluginUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrPluginUnavailable", err)
	}
}

func TestManagerRejectsCapabilityNotDeclaredByManifest(t *testing.T) {
	root := t.TempDir()
	manifest := validManifest("no-detect")
	manifest.Capabilities.Detect = false
	manifest.InputTypes = []sdk.AdapterInputType{sdk.InputCompatibilityProfile, sdk.InputResourceDocument}
	writePlugin(t, root, "no-detect", manifest, staticInfoPlugin(t, manifest))
	manager, err := Discover(root, 5*time.Second, quietLogger())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	_, err = manager.Detect(context.Background(), manifest.ID, sdk.DetectRequest{
		Observation: json.RawMessage(`{"manufacturer":"test"}`),
	})
	if !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatalf("Detect() error = %v, want ErrCapabilityUnavailable", err)
	}
}

func TestDiscoverContextStopsWhenStartupIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := DiscoverContext(ctx, t.TempDir(), time.Second, quietLogger()); !errors.Is(err, context.Canceled) {
		t.Fatalf("DiscoverContext() error = %v, want context.Canceled", err)
	}
}

func validManifest(id string) sdk.AdapterManifest {
	return sdk.AdapterManifest{
		ManifestVersion: sdk.ManifestVersion,
		APIVersion:      sdk.APIVersion,
		SDKVersion:      sdk.SDKVersion,
		ID:              id,
		Name:            "Test Adapter",
		Version:         "0.1.0",
		Executable:      "bin/plugin",
		Capabilities: sdk.AdapterCapabilities{
			Detect:             true,
			CheckCompatibility: true,
			InspectResources:   true,
		},
		InputTypes: []sdk.AdapterInputType{
			sdk.InputVehicleObservationJSON,
			sdk.InputCompatibilityProfile,
			sdk.InputResourceDocument,
		},
		ResourceFormats: []string{"android-strings-xml"},
	}
}

func staticInfoPlugin(t *testing.T, manifest sdk.AdapterManifest) string {
	t.Helper()
	return "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '" + responseForManifest(t, manifest) + "'\n"
}

func responseForManifest(t *testing.T, manifest sdk.AdapterManifest) string {
	t.Helper()
	result, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      "1",
		"result":  json.RawMessage(result),
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(response)
}

func rpcResultResponse(t *testing.T, result any) string {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      "1",
		"result":  json.RawMessage(encoded),
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(response)
}

func writePlugin(t *testing.T, root, directory string, manifest sdk.AdapterManifest, script string) {
	t.Helper()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeRawPlugin(t, root, directory, string(encoded), script)
}

func writeRawPlugin(t *testing.T, root, directory, manifest, script string) {
	t.Helper()
	pluginDir := filepath.Join(root, directory)
	if err := os.MkdirAll(filepath.Join(pluginDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if script != "" {
		if err := os.WriteFile(filepath.Join(pluginDir, "bin", "plugin"), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
}
