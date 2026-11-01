package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

const maxInputFieldBytes = 256

type Service struct {
	adapters AdapterGateway
	catalog  CatalogRepository
}

func NewService(adapters AdapterGateway, catalog CatalogRepository) (*Service, error) {
	if adapters == nil {
		return nil, errorsMissingDependency("adapter gateway")
	}
	if catalog == nil {
		return nil, errorsMissingDependency("catalog repository")
	}
	return &Service{adapters: adapters, catalog: catalog}, nil
}

func (s *Service) ListAdapters(ctx context.Context) ([]domain.AdapterInfo, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return s.adapters.List(ctx)
}

func (s *Service) GetAdapterInfo(ctx context.Context, adapterID string) (domain.AdapterInfo, error) {
	if err := contextError(ctx); err != nil {
		return domain.AdapterInfo{}, err
	}
	if err := validateField("adapter_id", adapterID); err != nil {
		return domain.AdapterInfo{}, err
	}
	return s.adapters.GetInfo(ctx, adapterID)
}

func (s *Service) Detect(ctx context.Context, adapterID string, request domain.DetectRequest) (domain.DetectionResult, error) {
	if err := contextError(ctx); err != nil {
		return domain.DetectionResult{}, err
	}
	if err := validateField("adapter_id", adapterID); err != nil {
		return domain.DetectionResult{}, err
	}
	if len(request.Observation) == 0 {
		return domain.DetectionResult{}, invalidArgument("observation is required")
	}
	if len(request.Observation) > 1<<20 {
		return domain.DetectionResult{}, invalidArgument("observation exceeds the size limit")
	}
	return s.adapters.Detect(ctx, adapterID, request)
}

func (s *Service) CheckCompatibility(ctx context.Context, adapterID string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	if err := contextError(ctx); err != nil {
		return domain.CompatibilityResult{}, err
	}
	if err := validateField("adapter_id", adapterID); err != nil {
		return domain.CompatibilityResult{}, err
	}
	fields := []struct {
		name  string
		value string
	}{
		{name: "identity.manufacturer", value: request.Identity.Manufacturer},
		{name: "identity.model", value: request.Identity.Model},
		{name: "identity.platform_id", value: request.Identity.PlatformID},
		{name: "hardware.platform_id", value: request.Hardware.PlatformID},
		{name: "hardware.revision", value: request.Hardware.Revision},
		{name: "firmware.build_id", value: request.Firmware.BuildID},
		{name: "firmware.version", value: request.Firmware.Version},
	}
	for _, field := range fields {
		if err := validateField(field.name, field.value); err != nil {
			return domain.CompatibilityResult{}, err
		}
	}
	return s.adapters.CheckCompatibility(ctx, adapterID, request)
}

func (s *Service) InspectResources(ctx context.Context, adapterID string, request domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	if err := contextError(ctx); err != nil {
		return domain.ResourceInspection{}, err
	}
	if err := validateField("adapter_id", adapterID); err != nil {
		return domain.ResourceInspection{}, err
	}
	if err := validateField("format", request.Format); err != nil {
		return domain.ResourceInspection{}, err
	}
	if len(request.Content) > 1<<20 {
		return domain.ResourceInspection{}, invalidArgument("resource document exceeds the size limit")
	}
	return s.adapters.InspectResources(ctx, adapterID, request)
}

func (s *Service) ListManufacturers(ctx context.Context) ([]domain.Manufacturer, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	items, err := s.catalog.ListManufacturers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list manufacturers: %w", err)
	}
	return items, nil
}

func (s *Service) ListVehicleModels(ctx context.Context, manufacturerID string) ([]domain.VehicleModel, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	items, err := s.catalog.ListVehicleModels(ctx, manufacturerID)
	if err != nil {
		return nil, fmt.Errorf("list vehicle models: %w", err)
	}
	return items, nil
}

func (s *Service) ListInfotainmentPlatforms(ctx context.Context, manufacturerID string) ([]domain.InfotainmentPlatform, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	items, err := s.catalog.ListInfotainmentPlatforms(ctx, manufacturerID)
	if err != nil {
		return nil, fmt.Errorf("list infotainment platforms: %w", err)
	}
	return items, nil
}

func validateField(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidArgument("%s is required", name)
	}
	if len(value) > maxInputFieldBytes {
		return invalidArgument("%s must not exceed %d bytes", name, maxInputFieldBytes)
	}
	return nil
}

func invalidArgument(format string, args ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidArgument, fmt.Sprintf(format, args...))
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return invalidArgument("context is required")
	}
	return ctx.Err()
}

func errorsMissingDependency(name string) error {
	return fmt.Errorf("application service requires %s", name)
}
