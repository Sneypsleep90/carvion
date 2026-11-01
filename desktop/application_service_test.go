package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/config"
	"github.com/Sneypsleep90/carvion/internal/core/domain"
	wailsapp "github.com/wailsapp/wails/v3/pkg/application"
)

func TestServiceStartupKeepsWindowAvailableWhenCoreCannotStart(t *testing.T) {
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, errors.New("DATABASE_URL is required")
		},
	}

	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() error = %v, want degraded startup", err)
	}

	state := service.GetApplicationState()
	if state.CoreStatus != "unavailable" {
		t.Errorf("CoreStatus = %q, want unavailable", state.CoreStatus)
	}
	if !strings.Contains(state.Message, "DATABASE_URL is required") {
		t.Errorf("Message = %q, want configuration error", state.Message)
	}
	if _, err := service.ListAdapters(); !errors.Is(err, ErrCoreUnavailable) {
		t.Errorf("ListAdapters() error = %v, want ErrCoreUnavailable", err)
	}
}

func TestServiceDelegatesToCoreAndClosesRuntime(t *testing.T) {
	core := &coreSpy{
		adapters: []domain.AdapterInfo{{
			ID:      "adapter-android-demo",
			Name:    "CARVION Synthetic Android Demo",
			Version: "0.1.0",
		}},
		compatibility: domain.CompatibilityResult{
			Status: domain.StatusSupported,
			Reason: "exact synthetic demo profile match",
		},
	}
	runtime := &runtimeSpy{}
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) {
			return config.Config{}, nil
		},
		startRuntime: func(context.Context, config.Config, *slog.Logger) (coreService, managedRuntime, error) {
			return core, runtime, nil
		},
	}

	if err := service.ServiceStartup(context.Background(), wailsapp.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() error = %v", err)
	}

	request := domain.CompatibilityRequest{
		Identity: domain.VehicleIdentity{Manufacturer: "CARVION TEST LAB", Model: "Android Demo Unit"},
	}
	result, err := service.CheckCompatibility("adapter-android-demo", request)
	if err != nil {
		t.Fatalf("CheckCompatibility() error = %v", err)
	}
	if result.Status != domain.StatusSupported {
		t.Errorf("Status = %q, want %q", result.Status, domain.StatusSupported)
	}
	if core.compatibilityCalls != 1 || core.compatibilityRequest.Identity.Model != request.Identity.Model {
		t.Errorf("Core compatibility call = %+v, want one delegated request", core.compatibilityRequest)
	}

	state := service.GetApplicationState()
	if state.CoreStatus != "ready" || state.DatabaseStatus != "connected" {
		t.Errorf("GetApplicationState() = %+v, want ready local Core", state)
	}
	if state.AdapterCount != 1 {
		t.Errorf("AdapterCount = %d, want 1", state.AdapterCount)
	}

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown() error = %v", err)
	}
	if !runtime.closed {
		t.Error("ServiceShutdown() did not close the local runtime")
	}
}

func TestServiceStartupRejectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &ApplicationService{
		loadConfig: func() (config.Config, error) {
			t.Fatal("configuration loaded after startup context was cancelled")
			return config.Config{}, nil
		},
	}

	if err := service.ServiceStartup(ctx, wailsapp.ServiceOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ServiceStartup() error = %v, want context.Canceled", err)
	}
}

type coreSpy struct {
	adapters             []domain.AdapterInfo
	compatibility        domain.CompatibilityResult
	compatibilityRequest domain.CompatibilityRequest
	compatibilityCalls   int
}

func (c *coreSpy) ListAdapters(context.Context) ([]domain.AdapterInfo, error) {
	return c.adapters, nil
}

func (*coreSpy) ListManufacturers(context.Context) ([]domain.Manufacturer, error) {
	return []domain.Manufacturer{}, nil
}

func (*coreSpy) ListVehicleModels(context.Context, string) ([]domain.VehicleModel, error) {
	return []domain.VehicleModel{}, nil
}

func (*coreSpy) ListInfotainmentPlatforms(context.Context, string) ([]domain.InfotainmentPlatform, error) {
	return []domain.InfotainmentPlatform{}, nil
}

func (c *coreSpy) CheckCompatibility(_ context.Context, _ string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	c.compatibilityCalls++
	c.compatibilityRequest = request
	return c.compatibility, nil
}

func (*coreSpy) Detect(context.Context, string, domain.DetectRequest) (domain.DetectionResult, error) {
	return domain.DetectionResult{}, nil
}

func (*coreSpy) InspectResources(context.Context, string, domain.InspectResourcesRequest) (domain.ResourceInspection, error) {
	return domain.ResourceInspection{Resources: []domain.TranslatableResource{}}, nil
}

type runtimeSpy struct {
	pingErr error
	closed  bool
}

func (r *runtimeSpy) Ping(context.Context) error { return r.pingErr }
func (r *runtimeSpy) Close()                     { r.closed = true }
