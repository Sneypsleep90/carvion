package adapterbridge

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/pluginmanager"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

type managerStub struct {
	items      []pluginmanager.AdapterSummary
	info       sdk.AdapterManifest
	detection  sdk.DetectionResult
	result     sdk.CompatibilityResult
	inspection sdk.ResourceInspection
	err        error
	gotContext context.Context
	gotID      string
	gotRequest any
}

func TestGatewayPassesOptionalVehicleResearchProfileWithoutChangingOtherFields(t *testing.T) {
	manager := &managerStub{result: sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "no exact record"}}
	gateway := New(manager)
	profile := json.RawMessage(`{"schema":"carvion.geely-research/v1","profile_id":"test-only"}`)
	request := domain.CompatibilityRequest{
		Identity: domain.VehicleIdentity{Manufacturer: "TEST ONLY", Model: "TEST ONLY", PlatformID: "TEST ONLY"},
		Hardware: domain.HardwareInfo{PlatformID: "TEST ONLY", Revision: "TEST ONLY"},
		Firmware: domain.FirmwareInfo{BuildID: "TEST ONLY", Version: "TEST ONLY"},
	}

	got, err := gateway.CheckCompatibilityWithVehicleProfile(context.Background(), "adapter-geely-research", request, profile)
	if err != nil || got.Status != domain.StatusUnknown {
		t.Fatalf("CheckCompatibilityWithVehicleProfile() = %+v, %v", got, err)
	}
	want := sdk.CompatibilityRequest{
		Identity:       sdk.VehicleIdentity{Manufacturer: "TEST ONLY", Model: "TEST ONLY", PlatformID: "TEST ONLY"},
		Hardware:       sdk.HardwareInfo{PlatformID: "TEST ONLY", Revision: "TEST ONLY"},
		Firmware:       sdk.FirmwareInfo{BuildID: "TEST ONLY", Version: "TEST ONLY"},
		VehicleProfile: profile,
	}
	if got := manager.gotRequest.(sdk.CompatibilityRequest); !reflect.DeepEqual(got, want) {
		t.Fatalf("manager request = %+v, want %+v", got, want)
	}
}

func (s *managerStub) List() []pluginmanager.AdapterSummary { return s.items }

func (s *managerStub) GetInfo(ctx context.Context, adapterID string) (sdk.AdapterManifest, error) {
	s.gotContext, s.gotID = ctx, adapterID
	return s.info, s.err
}

func (s *managerStub) Detect(ctx context.Context, adapterID string, request sdk.DetectRequest) (sdk.DetectionResult, error) {
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	return s.detection, s.err
}

func (s *managerStub) CheckCompatibility(ctx context.Context, adapterID string, request sdk.CompatibilityRequest) (sdk.CompatibilityResult, error) {
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	return s.result, s.err
}

func (s *managerStub) InspectResources(ctx context.Context, adapterID string, request sdk.InspectResourcesRequest) (sdk.ResourceInspection, error) {
	s.gotContext, s.gotID, s.gotRequest = ctx, adapterID, request
	return s.inspection, s.err
}

func TestGatewayMapsAdapterListAndCompatibilityResults(t *testing.T) {
	manager := &managerStub{
		items: []pluginmanager.AdapterSummary{{
			ID: "adapter-demo", Name: "Synthetic Demo", Version: "1.2.3", APIVersion: "v1",
			SDKVersion: sdk.SDKVersion, InputTypes: []sdk.AdapterInputType{sdk.InputCompatibilityProfile},
			ResourceFormats: []string{}, Status: pluginmanager.AdapterStatusAvailable,
			Capabilities: sdk.AdapterCapabilities{CheckCompatibility: true},
		}},
		result: sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "profile not found"},
	}
	gateway := New(manager)
	ctx := context.Background()

	items, err := gateway.List(ctx)
	if err != nil || len(items) != 1 || items[0].ID != "adapter-demo" || !items[0].Capabilities.CheckCompatibility ||
		items[0].SDKVersion != sdk.SDKVersion || items[0].Status != domain.AdapterStatusAvailable || len(items[0].InputTypes) != 1 {
		t.Fatalf("List() = %+v, %v", items, err)
	}
	request := domain.CompatibilityRequest{Firmware: domain.FirmwareInfo{BuildID: "build-x", Version: "1.0"}}
	got, err := gateway.CheckCompatibility(ctx, "adapter-demo", request)
	if err != nil || got.Status != domain.StatusUnknown || got.Reason != "profile not found" {
		t.Fatalf("CheckCompatibility() = %+v, %v", got, err)
	}
	wantRequest := sdk.CompatibilityRequest{Firmware: sdk.FirmwareInfo{BuildID: "build-x", Version: "1.0"}}
	if manager.gotContext != ctx || manager.gotID != "adapter-demo" || !reflect.DeepEqual(manager.gotRequest, wantRequest) {
		t.Fatalf("manager received context=%v id=%q request=%+v", manager.gotContext, manager.gotID, manager.gotRequest)
	}
}

func TestGatewayMapsManagerErrorsToDomainErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: pluginmanager.ErrAdapterNotFound, want: domain.ErrAdapterNotFound},
		{name: "capability", err: pluginmanager.ErrCapabilityUnavailable, want: domain.ErrCapabilityUnavailable},
		{name: "resource format", err: pluginmanager.ErrResourceFormatUnavailable, want: domain.ErrResourceFormatUnavailable},
		{name: "timeout", err: pluginmanager.ErrPluginTimeout, want: domain.ErrAdapterTimeout},
		{name: "process", err: pluginmanager.ErrPluginUnavailable, want: domain.ErrAdapterUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := New(&managerStub{err: tt.err})
			_, err := gateway.GetInfo(context.Background(), "adapter-demo")
			if !errors.Is(err, tt.want) {
				t.Fatalf("GetInfo() error = %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}
