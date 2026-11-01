package geelyresearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/pluginmanager"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

func TestAdapterRunsThroughPluginManagerJSONRPCProcess(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pluginRoot := t.TempDir()
	pluginDir := filepath.Join(pluginRoot, "adapter-geely-research")
	binDir := filepath.Join(pluginDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(binDir, "adapter-geely-research")
	build := exec.Command("go", "build", "-o", executable, "./cmd/adapter-geely-research")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build adapter process: %v\n%s", err, output)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(repositoryRoot, "adapters", "adapter-geely-research", "manifest.json"))
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	var manifest sdk.AdapterManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}

	var discoveryLogs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&discoveryLogs, nil))
	manager, err := pluginmanager.DiscoverContext(context.Background(), pluginRoot, 10*time.Second, logger)
	if err != nil {
		t.Fatalf("DiscoverContext() error = %v", err)
	}
	if adapters := manager.List(); len(adapters) != 1 || adapters[0].ID != "adapter-geely-research" {
		t.Fatalf("registered adapters = %+v, want Geely research process; discovery logs:\n%s", adapters, discoveryLogs.String())
	}

	profileJSON := testProfileJSON(t)
	detection, err := manager.Detect(context.Background(), manifest.ID, sdk.DetectRequest{Observation: profileJSON})
	if err != nil || detection.Status != sdk.StatusUnknown {
		t.Fatalf("Detect() = %+v, %v; want unknown file observation", detection, err)
	}

	var decodedProfile struct {
		Vehicle struct {
			Manufacturer string `json:"manufacturer"`
			Model        string `json:"model"`
		} `json:"vehicle"`
		Infotainment struct {
			Platform           string `json:"platform"`
			HardwareIdentifier string `json:"hardware_identifier"`
			HardwareRevision   string `json:"hardware_revision"`
			FirmwareVersion    string `json:"firmware_version"`
			FirmwareBuildID    string `json:"firmware_build_id"`
		} `json:"infotainment"`
	}
	if err := json.Unmarshal(profileJSON, &decodedProfile); err != nil {
		t.Fatal(err)
	}
	compatibility, err := manager.CheckCompatibility(context.Background(), manifest.ID, sdk.CompatibilityRequest{
		Identity: sdk.VehicleIdentity{
			Manufacturer: decodedProfile.Vehicle.Manufacturer,
			Model:        decodedProfile.Vehicle.Model,
			PlatformID:   decodedProfile.Infotainment.Platform,
		},
		Hardware:       sdk.HardwareInfo{PlatformID: decodedProfile.Infotainment.HardwareIdentifier, Revision: decodedProfile.Infotainment.HardwareRevision},
		Firmware:       sdk.FirmwareInfo{BuildID: decodedProfile.Infotainment.FirmwareBuildID, Version: decodedProfile.Infotainment.FirmwareVersion},
		VehicleProfile: json.RawMessage(profileJSON),
	})
	if err != nil || compatibility.Status != sdk.StatusUnknown {
		t.Fatalf("CheckCompatibility() = %+v, %v; want unknown with empty rules", compatibility, err)
	}

	inspection, err := manager.InspectResources(context.Background(), manifest.ID, sdk.InspectResourcesRequest{
		Format:   "android-strings-xml",
		Filename: "strings.xml",
		Content:  `<resources><string name="hello">Hello</string></resources>`,
	})
	if err != nil || len(inspection.Resources) != 1 || inspection.Resources[0] != (sdk.TranslatableResource{ID: "hello", Text: "Hello"}) {
		t.Fatalf("InspectResources() = %+v, %v; want hello resource from RPC process", inspection, err)
	}
	if _, err := manager.CheckCompatibility(context.Background(), manifest.ID, sdk.CompatibilityRequest{VehicleProfile: json.RawMessage(`{"schema":"invalid"}`)}); !errors.Is(err, pluginmanager.ErrPluginUnavailable) {
		t.Fatalf("CheckCompatibility() malformed RPC profile error = %v, want ErrPluginUnavailable", err)
	}
}
