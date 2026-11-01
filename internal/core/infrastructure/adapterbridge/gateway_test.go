package adapterbridge

import (
	"context"
	"errors"
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
			Capabilities: sdk.AdapterCapabilities{CheckCompatibility: true},
		}},
		result: sdk.CompatibilityResult{Status: sdk.StatusUnknown, Reason: "profile not found"},
	}
	gateway := New(manager)
	ctx := context.Background()

	items, err := gateway.List(ctx)
	if err != nil || len(items) != 1 || items[0].ID != "adapter-demo" || !items[0].Capabilities.CheckCompatibility {
		t.Fatalf("List() = %+v, %v", items, err)
	}
	request := domain.CompatibilityRequest{Firmware: domain.FirmwareInfo{BuildID: "build-x", Version: "1.0"}}
	got, err := gateway.CheckCompatibility(ctx, "adapter-demo", request)
	if err != nil || got.Status != domain.StatusUnknown || got.Reason != "profile not found" {
		t.Fatalf("CheckCompatibility() = %+v, %v", got, err)
	}
	wantRequest := sdk.CompatibilityRequest{Firmware: sdk.FirmwareInfo{BuildID: "build-x", Version: "1.0"}}
	if manager.gotContext != ctx || manager.gotID != "adapter-demo" || manager.gotRequest != wantRequest {
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
