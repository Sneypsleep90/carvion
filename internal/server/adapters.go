package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

const maxCompatibilityRequestBytes = 64 << 10

type compatibilityCheckRequest struct {
	AdapterID string                 `json:"adapter_id"`
	Identity  vehicleIdentityRequest `json:"identity"`
	Hardware  hardwareInfoRequest    `json:"hardware"`
	Firmware  firmwareInfoRequest    `json:"firmware"`
}

type compatibilityCheckResponse struct {
	AdapterID string                         `json:"adapter_id"`
	Result    compatibilityResultAPIResponse `json:"result"`
}

type vehicleIdentityRequest struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	PlatformID   string `json:"platform_id"`
}

type hardwareInfoRequest struct {
	PlatformID string `json:"platform_id"`
	Revision   string `json:"revision"`
}

type firmwareInfoRequest struct {
	BuildID string `json:"build_id"`
	Version string `json:"version"`
}

type adapterCapabilitiesResponse struct {
	Detect             bool `json:"detect"`
	CheckCompatibility bool `json:"check_compatibility"`
	InspectResources   bool `json:"inspect_resources"`
}

type adapterInfoResponse struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	Version      string                      `json:"version"`
	APIVersion   string                      `json:"api_version"`
	Capabilities adapterCapabilitiesResponse `json:"capabilities"`
}

type compatibilityResultAPIResponse struct {
	Status domain.CompatibilityStatus `json:"status"`
	Reason string                     `json:"reason"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

func adaptersListHandler(core Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if core == nil {
			writeAPIError(w, http.StatusServiceUnavailable, "core_unavailable", "application core is unavailable")
			return
		}
		items, err := core.ListAdapters(r.Context())
		if err != nil {
			writeApplicationError(w, err)
			return
		}
		adapters := make([]adapterInfoResponse, 0, len(items))
		for _, item := range items {
			adapters = append(adapters, adapterInfoResponse{
				ID: item.ID, Name: item.Name, Version: item.Version, APIVersion: item.APIVersion,
				Capabilities: adapterCapabilitiesResponse{
					Detect:             item.Capabilities.Detect,
					CheckCompatibility: item.Capabilities.CheckCompatibility,
					InspectResources:   item.Capabilities.InspectResources,
				},
			})
		}
		writeJSON(w, http.StatusOK, struct {
			Adapters []adapterInfoResponse `json:"adapters"`
		}{Adapters: adapters})
	}
}

func compatibilityCheckHandler(core Core) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if core == nil {
			writeAPIError(w, http.StatusServiceUnavailable, "core_unavailable", "application core is unavailable")
			return
		}
		if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
			writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "content type must be application/json")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxCompatibilityRequestBytes)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var body compatibilityCheckRequest
		if err := decoder.Decode(&body); err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				writeAPIError(w, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds the size limit")
				return
			}
			writeAPIError(w, http.StatusBadRequest, "invalid_json", "request body must be valid compatibility-check JSON")
			return
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			writeAPIError(w, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON object")
			return
		}
		request := domain.CompatibilityRequest{
			Identity: domain.VehicleIdentity{
				Manufacturer: body.Identity.Manufacturer,
				Model:        body.Identity.Model,
				PlatformID:   body.Identity.PlatformID,
			},
			Hardware: domain.HardwareInfo{PlatformID: body.Hardware.PlatformID, Revision: body.Hardware.Revision},
			Firmware: domain.FirmwareInfo{BuildID: body.Firmware.BuildID, Version: body.Firmware.Version},
		}
		result, err := core.CheckCompatibility(r.Context(), body.AdapterID, request)
		if err != nil {
			writeApplicationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, compatibilityCheckResponse{
			AdapterID: body.AdapterID,
			Result:    compatibilityResultAPIResponse{Status: result.Status, Reason: result.Reason},
		})
	}
}

func writeApplicationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		message := strings.TrimPrefix(err.Error(), domain.ErrInvalidArgument.Error()+": ")
		writeAPIError(w, http.StatusBadRequest, "invalid_request", message)
	case errors.Is(err, domain.ErrAdapterNotFound):
		writeAPIError(w, http.StatusNotFound, "adapter_not_found", "adapter is not registered")
	case errors.Is(err, domain.ErrCapabilityUnavailable):
		writeAPIError(w, http.StatusUnprocessableEntity, "capability_unavailable", "adapter does not support compatibility checks")
	case errors.Is(err, domain.ErrAdapterTimeout):
		writeAPIError(w, http.StatusServiceUnavailable, "adapter_unavailable", "adapter could not complete the compatibility check")
	case errors.Is(err, domain.ErrCatalogUnavailable):
		writeAPIError(w, http.StatusServiceUnavailable, "catalog_unavailable", "catalog data is unavailable")
	default:
		writeAPIError(w, http.StatusServiceUnavailable, "adapter_unavailable", "application core could not complete the request")
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorResponse{Error: apiError{Code: code, Message: message}})
}
