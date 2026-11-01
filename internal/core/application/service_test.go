package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

type adapterGatewayStub struct {
	items      []domain.AdapterInfo
	detection  domain.DetectionResult
	compat     domain.CompatibilityResult
	inspection domain.ResourceInspection
	err        error
	calls      []string
	gotContext context.Context
	gotID      string
	gotRequest any
}

func (s *adapterGatewayStub) List(ctx context.Context) ([]domain.AdapterInfo, error) {
	s.calls = append(s.calls, "list")
	s.gotContext = ctx
	return s.items, s.err
}

func (s *adapterGatewayStub) GetInfo(ctx context.Context, adapterID string) (domain.AdapterInfo, error) {
	s.calls = append(s.calls, "get_info")
	s.gotContext, s.gotID = ctx, adapterID
	return domain.AdapterInfo{}, s.err
}

func (s *adapterGatewayStub) Detect(ctx context.Context, adapterID string, request domain.DetectRequest) (domain.DetectionResult, error) {
	s.calls = append(s.calls, "detect")
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	return s.detection, s.err
}

func (s *adapterGatewayStub) CheckCompatibility(ctx context.Context, adapterID string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	s.calls = append(s.calls, "compatibility")
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	return s.compat, s.err
}

func (s *adapterGatewayStub) InspectResources(ctx context.Context, adapterID string, request domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	s.calls = append(s.calls, "inspect_resources")
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	return s.inspection, s.err
}

type catalogRepositoryStub struct {
	manufacturers []domain.Manufacturer
	models        []domain.VehicleModel
	platforms     []domain.InfotainmentPlatform
	err           error
	calls         []string
	gotContext    context.Context
	gotFilter     string
}

func (s *catalogRepositoryStub) ListManufacturers(ctx context.Context) ([]domain.Manufacturer, error) {
	s.calls = append(s.calls, "manufacturers")
	s.gotContext = ctx
	return s.manufacturers, s.err
}

func (s *catalogRepositoryStub) ListVehicleModels(ctx context.Context, manufacturerID string) ([]domain.VehicleModel, error) {
	s.calls = append(s.calls, "models")
	s.gotContext, s.gotFilter = ctx, manufacturerID
	return s.models, s.err
}

func (s *catalogRepositoryStub) ListInfotainmentPlatforms(ctx context.Context, manufacturerID string) ([]domain.InfotainmentPlatform, error) {
	s.calls = append(s.calls, "platforms")
	s.gotContext, s.gotFilter = ctx, manufacturerID
	return s.platforms, s.err
}

func TestServiceDelegatesAdapterOperationsAndPropagatesContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "request-context")
	gateway := &adapterGatewayStub{
		compat: domain.CompatibilityResult{Status: domain.StatusSupported, Reason: "verified demo profile"},
	}
	service := newTestService(t, gateway, &catalogRepositoryStub{})
	request := domain.CompatibilityRequest{
		Identity: domain.VehicleIdentity{Manufacturer: "Test Lab", Model: "Test Unit", PlatformID: "platform-a"},
		Hardware: domain.HardwareInfo{PlatformID: "platform-a", Revision: "rev-a"},
		Firmware: domain.FirmwareInfo{BuildID: "build-a", Version: "1.0.0"},
	}

	got, err := service.CheckCompatibility(ctx, "adapter-demo", request)
	if err != nil {
		t.Fatalf("CheckCompatibility() error = %v", err)
	}
	if got != gateway.compat {
		t.Fatalf("CheckCompatibility() = %+v, want %+v", got, gateway.compat)
	}
	if gateway.gotContext != ctx || gateway.gotID != "adapter-demo" || gateway.gotRequest != request {
		t.Fatalf("gateway received context=%v adapter=%q request=%+v", gateway.gotContext, gateway.gotID, gateway.gotRequest)
	}
}

func TestServiceRejectsInvalidCompatibilityBeforeCallingAdapter(t *testing.T) {
	gateway := &adapterGatewayStub{}
	service := newTestService(t, gateway, &catalogRepositoryStub{})

	_, err := service.CheckCompatibility(context.Background(), "adapter-demo", domain.CompatibilityRequest{})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("CheckCompatibility() error = %v, want ErrInvalidArgument", err)
	}
	if len(gateway.calls) != 0 {
		t.Fatalf("adapter calls = %v, want none for invalid request", gateway.calls)
	}
}

func TestServiceExposesDetectionAndResourceInspectionToDesktopCallers(t *testing.T) {
	gateway := &adapterGatewayStub{
		detection:  domain.DetectionResult{Status: domain.StatusUnknown, Reason: "no synthetic match"},
		inspection: domain.ResourceInspection{Resources: []domain.TranslatableResource{{ID: "title", Text: "Hello"}}},
	}
	service := newTestService(t, gateway, &catalogRepositoryStub{})
	ctx := context.Background()
	request := domain.DetectRequest{Observation: []byte(`{"model":"synthetic"}`)}

	gotDetection, err := service.Detect(ctx, "adapter-demo", request)
	if err != nil || gotDetection != gateway.detection {
		t.Fatalf("Detect() = %+v, %v; want %+v, nil", gotDetection, err, gateway.detection)
	}
	if !reflect.DeepEqual(gateway.gotRequest, request) || gateway.gotID != "adapter-demo" {
		t.Fatalf("Detect() gateway request = %+v adapter=%q", gateway.gotRequest, gateway.gotID)
	}

	resourceRequest := domain.InspectResourcesRequest{Format: "android-strings-xml", Content: "<resources/>"}
	gotInspection, err := service.InspectResources(ctx, "adapter-demo", resourceRequest)
	if err != nil || !reflect.DeepEqual(gotInspection, gateway.inspection) {
		t.Fatalf("InspectResources() = %+v, %v; want %+v, nil", gotInspection, err, gateway.inspection)
	}
	if !reflect.DeepEqual(gateway.gotRequest, resourceRequest) || gateway.gotID != "adapter-demo" {
		t.Fatalf("InspectResources() gateway request = %+v adapter=%q", gateway.gotRequest, gateway.gotID)
	}
}

func TestServiceReadsCatalogThroughRepositoryPort(t *testing.T) {
	repository := &catalogRepositoryStub{
		platforms: []domain.InfotainmentPlatform{{ID: "platform-a", Name: "Synthetic Platform"}},
	}
	service := newTestService(t, &adapterGatewayStub{}, repository)
	ctx := context.Background()

	got, err := service.ListInfotainmentPlatforms(ctx, "manufacturer-a")
	if err != nil {
		t.Fatalf("ListInfotainmentPlatforms() error = %v", err)
	}
	if !reflect.DeepEqual(got, repository.platforms) {
		t.Fatalf("ListInfotainmentPlatforms() = %+v, want %+v", got, repository.platforms)
	}
	if repository.gotContext != ctx || repository.gotFilter != "manufacturer-a" {
		t.Fatalf("repository received context=%v filter=%q", repository.gotContext, repository.gotFilter)
	}
}

func TestServiceHonorsCanceledContextBeforeCallingDependencies(t *testing.T) {
	gateway := &adapterGatewayStub{}
	repository := &catalogRepositoryStub{}
	service := newTestService(t, gateway, repository)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.ListAdapters(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListAdapters() error = %v, want context.Canceled", err)
	}
	if _, err := service.ListManufacturers(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListManufacturers() error = %v, want context.Canceled", err)
	}
	if len(gateway.calls) != 0 || len(repository.calls) != 0 {
		t.Fatalf("dependencies called after cancellation: adapters=%v repositories=%v", gateway.calls, repository.calls)
	}
}

func newTestService(t *testing.T, adapters AdapterGateway, catalog CatalogRepository) *Service {
	t.Helper()
	service, err := NewService(adapters, catalog)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}
