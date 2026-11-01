package geelyresearch

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	research "github.com/Sneypsleep90/carvion/internal/geelyresearch"
	"github.com/Sneypsleep90/carvion/internal/localization"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const (
	adapterID      = "adapter-geely-research"
	adapterName    = "CARVION Geely Research Adapter"
	adapterVersion = "0.1.0"
)

type Adapter struct {
	records []research.CompatibilityRecord
}

func New() (*Adapter, error) {
	records, err := research.LoadCompatibilityRecords()
	if err != nil {
		return nil, fmt.Errorf("load reviewed Geely compatibility catalog: %w", err)
	}
	return &Adapter{records: records}, nil
}

func (a *Adapter) GetInfo(ctx context.Context) (sdk.AdapterManifest, error) {
	if err := ctx.Err(); err != nil {
		return sdk.AdapterManifest{}, err
	}
	return sdk.AdapterManifest{
		ManifestVersion: sdk.ManifestVersion,
		APIVersion:      sdk.APIVersion,
		SDKVersion:      sdk.SDKVersion,
		ID:              adapterID,
		Name:            adapterName,
		Version:         adapterVersion,
		Executable:      "bin/adapter-geely-research",
		Capabilities: sdk.AdapterCapabilities{
			Detect:             true,
			CheckCompatibility: true,
			InspectResources:   true,
		},
		InputTypes: []sdk.AdapterInputType{
			sdk.InputVehicleObservationJSON,
			sdk.InputCompatibilityProfile,
			sdk.InputVehicleResearchProfileJSON,
			sdk.InputResourceDocument,
		},
		ResourceFormats: []string{"android-strings-xml", "carvion-json-v1"},
	}, nil
}

func (a *Adapter) Detect(ctx context.Context, request sdk.DetectRequest) (sdk.DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.DetectionResult{}, err
	}
	profile, err := research.ParseProfile(request.Observation)
	if err != nil {
		return sdk.DetectionResult{}, fmt.Errorf("decode imported Geely profile: %w", err)
	}
	return sdk.DetectionResult{
		Status: sdk.StatusUnknown,
		Identity: sdk.VehicleIdentity{
			Manufacturer: profile.Vehicle.Manufacturer,
			Model:        profile.Vehicle.Model,
			PlatformID:   profile.Infotainment.Platform,
		},
		Reason: "identity values came from an imported file; this is not device detection and no vehicle connection was made",
	}, nil
}

func (a *Adapter) CheckCompatibility(ctx context.Context, request sdk.CompatibilityRequest) (sdk.CompatibilityResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.CompatibilityResult{}, err
	}
	if len(request.VehicleProfile) == 0 {
		return sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "full Geely research profile is required for an exact match"}, nil
	}
	profile, err := research.ParseProfile(request.VehicleProfile)
	if err != nil {
		return sdk.CompatibilityResult{}, fmt.Errorf("decode Geely compatibility profile: %w", err)
	}
	if !requestMatchesProfile(request, profile) {
		return sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "compatibility fields conflict with the imported Geely profile"}, nil
	}
	result := research.CheckCompatibility(profile, a.records)
	return sdk.CompatibilityResult{Status: result.Status, Reason: result.Reason}, nil
}

func (a *Adapter) InspectResources(ctx context.Context, request sdk.InspectResourcesRequest) (sdk.ResourceInspection, error) {
	if err := ctx.Err(); err != nil {
		return sdk.ResourceInspection{}, err
	}
	if !safeFilename(request.Filename) {
		return sdk.ResourceInspection{}, errors.New("resource filename must be a basename without a path")
	}
	if len(request.Content) == 0 || len(request.Content) > research.MaxResourceContentBytes {
		return sdk.ResourceInspection{}, fmt.Errorf("resource content is empty or exceeds %d bytes", research.MaxResourceContentBytes)
	}
	if !utf8.ValidString(request.Content) {
		return sdk.ResourceInspection{}, errors.New("resource document is not valid UTF-8")
	}
	var format domain.LocalizationFormat
	switch request.Format {
	case string(domain.LocalizationFormatAndroidXML):
		format = domain.LocalizationFormatAndroidXML
	case string(domain.LocalizationFormatJSON):
		format = domain.LocalizationFormatJSON
	default:
		return sdk.ResourceInspection{}, fmt.Errorf("unsupported resource format %q", request.Format)
	}
	document, err := localization.Parse(format, request.Filename, []byte(request.Content))
	if err != nil {
		return sdk.ResourceInspection{}, fmt.Errorf("parse localization resource document: %w", err)
	}
	resources := make([]sdk.TranslatableResource, 0, len(document.Resources))
	for _, resource := range document.Resources {
		if !resource.Present || !resource.Translatable {
			continue
		}
		resources = append(resources, sdk.TranslatableResource{ID: resource.Key, Text: resource.SourceText})
	}
	return sdk.ResourceInspection{Resources: resources}, nil
}

func requestMatchesProfile(request sdk.CompatibilityRequest, profile research.Profile) bool {
	return request.Identity.Manufacturer == profile.Vehicle.Manufacturer &&
		request.Identity.Model == profile.Vehicle.Model &&
		request.Identity.PlatformID == profile.Infotainment.Platform &&
		request.Hardware.PlatformID == profile.Infotainment.HardwareIdentifier &&
		request.Hardware.Revision == profile.Infotainment.HardwareRevision &&
		request.Firmware.Version == profile.Infotainment.FirmwareVersion &&
		request.Firmware.BuildID == profile.Infotainment.FirmwareBuildID
}

func safeFilename(filename string) bool {
	if strings.TrimSpace(filename) == "" || strings.ContainsAny(filename, "\r\n\x00") {
		return false
	}
	if strings.ContainsAny(filename, `/\`) || path.Base(filename) != filename || filename == "." || filename == ".." {
		return false
	}
	return len(filename) <= 255
}

var _ sdk.Adapter = (*Adapter)(nil)
