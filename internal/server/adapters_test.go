package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

type testAdapterService struct {
	items   []domain.AdapterInfo
	result  domain.CompatibilityResult
	err     error
	gotID   string
	gotBody domain.CompatibilityRequest
}

func (s *testAdapterService) ListAdapters(context.Context) ([]domain.AdapterInfo, error) {
	return s.items, s.err
}

func (s *testAdapterService) CheckCompatibility(_ context.Context, adapterID string, request domain.CompatibilityRequest) (domain.CompatibilityResult, error) {
	s.gotID = adapterID
	s.gotBody = request
	if request.Identity.Manufacturer == "" {
		return domain.CompatibilityResult{}, domain.ErrInvalidArgument
	}
	return s.result, s.err
}

func TestListAdaptersReturnsPublicAdapterSummaries(t *testing.T) {
	service := &testAdapterService{items: []domain.AdapterInfo{{
		ID: "adapter-android-demo", Name: "Synthetic Demo", Version: "0.1.0", APIVersion: "v1",
		Capabilities: domain.AdapterCapabilities{CheckCompatibility: true},
	}}}
	handler := NewHandler(nil, nil, service)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/adapters", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"id":"adapter-android-demo"`) ||
		strings.Contains(recorder.Body.String(), "executable") {
		t.Errorf("body = %s, want public adapter summary without executable path", recorder.Body.String())
	}
}

func TestCompatibilityCheckReturnsStructuredResult(t *testing.T) {
	service := &testAdapterService{result: domain.CompatibilityResult{Status: domain.StatusSupported, Reason: "exact profile"}}
	handler := NewHandler(nil, nil, service)
	body := `{"adapter_id":"adapter-android-demo","identity":{"manufacturer":"CARVION TEST LAB","model":"Android Demo Unit","platform_id":"android-demo-platform"},"hardware":{"platform_id":"android-demo-platform","revision":"rev-a"},"firmware":{"build_id":"demo-build-a-001","version":"1.0.0-demo"}}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"supported"`) {
		t.Fatalf("response = %d %s, want supported JSON result", recorder.Code, recorder.Body.String())
	}
	if service.gotID != "adapter-android-demo" || service.gotBody.Firmware.BuildID != "demo-build-a-001" {
		t.Errorf("service received adapter=%q request=%+v", service.gotID, service.gotBody)
	}
}

func TestCompatibilityCheckRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"adapter_id":`},
		{name: "unknown field", body: `{"adapter_id":"a","extra":true}`},
		{name: "missing required fields", body: `{"adapter_id":"a"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil, nil, &testAdapterService{})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"error"`) {
				t.Fatalf("response = %d %s, want structured HTTP 400", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestCompatibilityCheckMapsApplicationErrors(t *testing.T) {
	for _, tt := range []struct {
		name    string
		coreErr error
		status  int
		code    string
	}{
		{name: "invalid argument", coreErr: domain.ErrInvalidArgument, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "unknown adapter", coreErr: domain.ErrAdapterNotFound, status: http.StatusNotFound, code: "adapter_not_found"},
		{name: "process failure", coreErr: domain.ErrAdapterUnavailable, status: http.StatusServiceUnavailable, code: "adapter_unavailable"},
		{name: "timeout", coreErr: domain.ErrAdapterTimeout, status: http.StatusServiceUnavailable, code: "adapter_unavailable"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &testAdapterService{err: tt.coreErr}
			handler := NewHandler(nil, nil, service)
			body := `{"adapter_id":"adapter-android-demo","identity":{"manufacturer":"x","model":"y","platform_id":"p"},"hardware":{"platform_id":"p","revision":"r"},"firmware":{"build_id":"b","version":"v"}}`
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/compatibility/check", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")

			handler.ServeHTTP(recorder, request)

			if recorder.Code != tt.status || !strings.Contains(recorder.Body.String(), `"code":"`+tt.code+`"`) {
				t.Fatalf("response = %d %s, want status %d code %q", recorder.Code, recorder.Body.String(), tt.status, tt.code)
			}
		})
	}
}
