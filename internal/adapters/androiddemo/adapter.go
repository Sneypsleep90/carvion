package androiddemo

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Sneypsleep90/carvion/internal/simulator"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const maxResourceDocumentBytes = 1 << 20

type DemoAdapter struct {
	profiles []profile
}

type profile struct {
	Identity         sdk.VehicleIdentity
	PlatformID       string
	HardwareRevision string
	FirmwareBuildID  string
	FirmwareVersion  string
	Status           sdk.CompatibilityStatus
	FaultMode        string
}

func New() (*DemoAdapter, error) {
	profiles, err := simulator.Profiles()
	if err != nil {
		return nil, fmt.Errorf("load synthetic simulator profiles: %w", err)
	}
	if len(profiles) == 0 {
		return nil, errors.New("synthetic simulator profiles are empty")
	}
	demoProfiles := make([]profile, 0, len(profiles))
	for _, item := range profiles {
		loaded, err := simulator.LoadProfile(item.ID)
		if err != nil {
			return nil, fmt.Errorf("load synthetic simulator profile %q: %w", item.ID, err)
		}
		demoProfiles = append(demoProfiles, profile{
			Identity: sdk.VehicleIdentity{
				Manufacturer: item.Identity.Manufacturer,
				Model:        item.Identity.Model,
				PlatformID:   item.Identity.PlatformID,
			},
			PlatformID:       item.Hardware.PlatformID,
			HardwareRevision: item.Hardware.Revision,
			FirmwareBuildID:  item.Firmware.BuildID,
			FirmwareVersion:  item.Firmware.Version,
			Status:           sdk.CompatibilityStatus(item.ExpectedCompatibility),
			FaultMode:        loaded.FaultMode,
		})
	}
	return &DemoAdapter{profiles: demoProfiles}, nil
}

func (a *DemoAdapter) GetInfo(ctx context.Context) (sdk.AdapterManifest, error) {
	if err := ctx.Err(); err != nil {
		return sdk.AdapterManifest{}, err
	}
	return sdk.AdapterManifest{
		ManifestVersion: sdk.ManifestVersion,
		APIVersion:      sdk.APIVersion,
		SDKVersion:      sdk.SDKVersion,
		ID:              "adapter-android-demo",
		Name:            "CARVION Synthetic Android Demo",
		Version:         "0.1.0",
		Executable:      "bin/adapter-android-demo",
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
	}, nil
}

func (a *DemoAdapter) Detect(ctx context.Context, request sdk.DetectRequest) (sdk.DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.DetectionResult{}, err
	}
	var observation vehicleObservation
	if err := decodeStrict(request.Observation, &observation); err != nil {
		return sdk.DetectionResult{}, fmt.Errorf("decode vehicle observation: %w", err)
	}
	identity := observation.identity()
	if observation.ProfileID != "" {
		profile, err := simulator.LoadProfile(observation.ProfileID)
		if err != nil {
			return sdk.DetectionResult{Status: sdk.StatusUnknown, Reason: "profile is not in the local synthetic simulator catalog"}, nil
		}
		if profile.FaultMode == simulator.FaultRPCError {
			return sdk.DetectionResult{}, errors.New("simulated adapter communication failure")
		}
		expectedIdentity := sdk.VehicleIdentity{
			Manufacturer: profile.Public.Identity.Manufacturer,
			Model:        profile.Public.Identity.Model,
			PlatformID:   profile.Public.Identity.PlatformID,
		}
		if !sameIdentity(identity, expectedIdentity) {
			return sdk.DetectionResult{Status: sdk.StatusUnknown, Reason: "observation does not match its synthetic profile"}, nil
		}
	}
	for _, item := range a.profiles {
		if sameIdentity(identity, item.Identity) {
			return sdk.DetectionResult{Status: sdk.StatusSupported, Identity: identity}, nil
		}
	}
	return sdk.DetectionResult{
		Status: sdk.StatusUnknown,
		Reason: "vehicle identity is not present in the synthetic demo profiles",
	}, nil
}

func (a *DemoAdapter) CheckCompatibility(ctx context.Context, request sdk.CompatibilityRequest) (sdk.CompatibilityResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.CompatibilityResult{}, err
	}

	identityKnown := false
	for _, item := range a.profiles {
		if item.Identity.Manufacturer != request.Identity.Manufacturer || item.Identity.Model != request.Identity.Model {
			continue
		}
		identityKnown = true
		if item.FirmwareBuildID != request.Firmware.BuildID {
			continue
		}
		if item.FaultMode == simulator.FaultRPCError {
			return sdk.CompatibilityResult{}, errors.New("simulated adapter communication failure")
		}
		if item.FirmwareVersion != request.Firmware.Version {
			return sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "firmware version does not exactly match a demo profile"}, nil
		}
		if request.Identity.PlatformID != item.PlatformID || request.Hardware.PlatformID != item.PlatformID ||
			request.Hardware.Revision != item.HardwareRevision {
			return sdk.CompatibilityResult{Status: sdk.StatusUnsupported, Reason: "known firmware build does not match the platform or hardware revision"}, nil
		}
		return sdk.CompatibilityResult{Status: item.Status, Reason: "exact synthetic demo profile match"}, nil
	}
	if !identityKnown {
		return sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "vehicle identity is not present in the synthetic demo profiles"}, nil
	}
	return sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "firmware build is not present in the synthetic demo profiles"}, nil
}

func (a *DemoAdapter) InspectResources(ctx context.Context, request sdk.InspectResourcesRequest) (sdk.ResourceInspection, error) {
	if err := ctx.Err(); err != nil {
		return sdk.ResourceInspection{}, err
	}
	if request.Format != "android-strings-xml" {
		return sdk.ResourceInspection{}, fmt.Errorf("unsupported resource format %q", request.Format)
	}
	if len(request.Content) > maxResourceDocumentBytes {
		return sdk.ResourceInspection{}, errors.New("resource document exceeds the size limit")
	}

	var document struct {
		XMLName xml.Name `xml:"resources"`
		Strings []struct {
			ID           string `xml:"name,attr"`
			Translatable string `xml:"translatable,attr"`
			Text         string `xml:",chardata"`
		} `xml:"string"`
	}
	if err := xml.Unmarshal([]byte(request.Content), &document); err != nil {
		return sdk.ResourceInspection{}, fmt.Errorf("parse Android strings XML: %w", err)
	}
	if document.XMLName.Local != "resources" {
		return sdk.ResourceInspection{}, errors.New("Android strings XML root must be resources")
	}

	resources := make([]sdk.TranslatableResource, 0, len(document.Strings))
	seen := make(map[string]struct{}, len(document.Strings))
	for _, item := range document.Strings {
		if strings.EqualFold(item.Translatable, "false") {
			continue
		}
		if item.ID == "" {
			return sdk.ResourceInspection{}, errors.New("Android string resource is missing its name")
		}
		if _, ok := seen[item.ID]; ok {
			return sdk.ResourceInspection{}, fmt.Errorf("duplicate Android string resource %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		resources = append(resources, sdk.TranslatableResource{ID: item.ID, Text: strings.TrimSpace(item.Text)})
	}
	return sdk.ResourceInspection{Resources: resources}, nil
}

func decodeStrict(data []byte, target any) error {
	if len(data) == 0 || len(data) > maxResourceDocumentBytes {
		return errors.New("JSON observation is empty or exceeds the size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("observation contains multiple JSON values")
		}
		return err
	}
	return nil
}

func sameIdentity(a, b sdk.VehicleIdentity) bool {
	return a.Manufacturer == b.Manufacturer && a.Model == b.Model && a.PlatformID == b.PlatformID
}

type vehicleObservation struct {
	ProfileID    string              `json:"profile_id,omitempty"`
	Identity     sdk.VehicleIdentity `json:"identity,omitempty"`
	Manufacturer string              `json:"manufacturer,omitempty"`
	Model        string              `json:"model,omitempty"`
	PlatformID   string              `json:"platform_id,omitempty"`
}

func (o vehicleObservation) identity() sdk.VehicleIdentity {
	if o.Identity.Manufacturer != "" || o.Identity.Model != "" || o.Identity.PlatformID != "" {
		return o.Identity
	}
	return sdk.VehicleIdentity{Manufacturer: o.Manufacturer, Model: o.Model, PlatformID: o.PlatformID}
}
