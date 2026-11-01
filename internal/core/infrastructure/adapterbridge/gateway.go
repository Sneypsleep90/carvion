package adapterbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/pluginmanager"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

type Manager interface {
	List() []pluginmanager.AdapterSummary
	GetInfo(context.Context, string) (sdk.AdapterManifest, error)
	Detect(context.Context, string, sdk.DetectRequest) (sdk.DetectionResult, error)
	CheckCompatibility(context.Context, string, sdk.CompatibilityRequest) (sdk.CompatibilityResult, error)
	InspectResources(context.Context, string, sdk.InspectResourcesRequest) (sdk.ResourceInspection, error)
}

type Gateway struct {
	manager Manager
}

func New(manager Manager) *Gateway {
	return &Gateway{manager: manager}
}

func (g *Gateway) List(ctx context.Context) ([]domain.AdapterInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := g.manager.List()
	result := make([]domain.AdapterInfo, 0, len(items))
	for _, item := range items {
		result = append(result, domain.AdapterInfo{
			ID: item.ID, Name: item.Name, Version: item.Version, APIVersion: item.APIVersion, SDKVersion: item.SDKVersion,
			InputTypes: inputTypes(item.InputTypes), ResourceFormats: append([]string{}, item.ResourceFormats...),
			Status: domain.AdapterStatus(item.Status),
			Capabilities: domain.AdapterCapabilities{
				Detect:             item.Capabilities.Detect,
				CheckCompatibility: item.Capabilities.CheckCompatibility,
				InspectResources:   item.Capabilities.InspectResources,
			},
		})
	}
	return result, nil
}

func (g *Gateway) GetInfo(ctx context.Context, adapterID string) (domain.AdapterInfo, error) {
	manifest, err := g.manager.GetInfo(ctx, adapterID)
	if err != nil {
		return domain.AdapterInfo{}, mapManagerError(err)
	}
	return adapterInfo(manifest), nil
}

func (g *Gateway) Detect(ctx context.Context, adapterID string, request domain.DetectRequest) (domain.DetectionResult, error) {
	result, err := g.manager.Detect(ctx, adapterID, sdk.DetectRequest{Observation: request.Observation})
	if err != nil {
		return domain.DetectionResult{}, mapManagerError(err)
	}
	return domain.DetectionResult{
		Status: domain.CompatibilityStatus(result.Status),
		Identity: domain.VehicleIdentity{
			Manufacturer: result.Identity.Manufacturer,
			Model:        result.Identity.Model,
			PlatformID:   result.Identity.PlatformID,
		},
		Reason: result.Reason,
	}, nil
}

func (g *Gateway) CheckCompatibility(ctx context.Context, adapterID string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	return g.checkCompatibility(ctx, adapterID, request, nil)
}

// CheckCompatibilityWithVehicleProfile passes an optional versioned research
// profile through the additive SDK v1.1 request field.
func (g *Gateway) CheckCompatibilityWithVehicleProfile(ctx context.Context, adapterID string, request domain.CompatibilityRequest, profile json.RawMessage) (domain.CompatibilityResult, error) {
	return g.checkCompatibility(ctx, adapterID, request, profile)
}

func (g *Gateway) checkCompatibility(ctx context.Context, adapterID string, request domain.CompatibilityRequest, profile json.RawMessage) (domain.CompatibilityResult, error) {
	result, err := g.manager.CheckCompatibility(ctx, adapterID, sdk.CompatibilityRequest{
		Identity:       sdk.VehicleIdentity{Manufacturer: request.Identity.Manufacturer, Model: request.Identity.Model, PlatformID: request.Identity.PlatformID},
		Hardware:       sdk.HardwareInfo{PlatformID: request.Hardware.PlatformID, Revision: request.Hardware.Revision},
		Firmware:       sdk.FirmwareInfo{BuildID: request.Firmware.BuildID, Version: request.Firmware.Version},
		VehicleProfile: append(json.RawMessage(nil), profile...),
	})
	if err != nil {
		return domain.CompatibilityResult{}, mapManagerError(err)
	}
	return domain.CompatibilityResult{Status: domain.CompatibilityStatus(result.Status), Reason: result.Reason}, nil
}

func (g *Gateway) InspectResources(ctx context.Context, adapterID string, request domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	result, err := g.manager.InspectResources(ctx, adapterID, sdk.InspectResourcesRequest{
		Format: request.Format, Filename: request.Filename, Content: request.Content,
	})
	if err != nil {
		return domain.ResourceInspection{}, mapManagerError(err)
	}
	resources := make([]domain.TranslatableResource, 0, len(result.Resources))
	for _, item := range result.Resources {
		resources = append(resources, domain.TranslatableResource{ID: item.ID, Text: item.Text})
	}
	return domain.ResourceInspection{Resources: resources}, nil
}

func adapterInfo(manifest sdk.AdapterManifest) domain.AdapterInfo {
	return domain.AdapterInfo{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version, APIVersion: manifest.APIVersion, SDKVersion: manifest.SDKVersion,
		InputTypes: inputTypes(manifest.InputTypes), ResourceFormats: append([]string{}, manifest.ResourceFormats...),
		Status: domain.AdapterStatusAvailable,
		Capabilities: domain.AdapterCapabilities{
			Detect:             manifest.Capabilities.Detect,
			CheckCompatibility: manifest.Capabilities.CheckCompatibility,
			InspectResources:   manifest.Capabilities.InspectResources,
		},
	}
}

func inputTypes(inputTypes []sdk.AdapterInputType) []string {
	result := make([]string, 0, len(inputTypes))
	for _, inputType := range inputTypes {
		result = append(result, string(inputType))
	}
	return result
}

func mapManagerError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, pluginmanager.ErrAdapterNotFound):
		return fmt.Errorf("%w: %v", domain.ErrAdapterNotFound, err)
	case errors.Is(err, pluginmanager.ErrCapabilityUnavailable):
		return fmt.Errorf("%w: %v", domain.ErrCapabilityUnavailable, err)
	case errors.Is(err, pluginmanager.ErrResourceFormatUnavailable):
		return fmt.Errorf("%w: %v", domain.ErrResourceFormatUnavailable, err)
	case errors.Is(err, pluginmanager.ErrPluginTimeout):
		return fmt.Errorf("%w: %v", domain.ErrAdapterTimeout, err)
	default:
		return fmt.Errorf("%w: %v", domain.ErrAdapterUnavailable, err)
	}
}
