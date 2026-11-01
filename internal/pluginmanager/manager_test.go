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
		ID:              id,
		Name:            "Test Adapter",
		Version:         "0.1.0",
		Executable:      "bin/plugin",
		Capabilities: sdk.AdapterCapabilities{
			Detect:             true,
			CheckCompatibility: true,
			InspectResources:   true,
		},
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
