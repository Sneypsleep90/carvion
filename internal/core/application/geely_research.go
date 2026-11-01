package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	research "github.com/Sneypsleep90/carvion/internal/geelyresearch"
	"github.com/Sneypsleep90/carvion/internal/localization"
	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const geelyResearchAdapterID = "adapter-geely-research"

type vehicleProfileCompatibilityGateway interface {
	CheckCompatibilityWithVehicleProfile(context.Context, string, domain.CompatibilityRequest, json.RawMessage) (domain.CompatibilityResult, error)
}

func (s *Service) ListGeelyResearchEvidence(ctx context.Context) ([]domain.GeelyResearchEvidence, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	sources := research.PublicEvidence()
	result := make([]domain.GeelyResearchEvidence, 0, len(sources))
	for _, source := range sources {
		result = append(result, domain.GeelyResearchEvidence{
			ID: source.ID, Title: source.Title, Publisher: source.Publisher, URL: source.URL,
			PublishedAt: source.PublishedAt, RetrievedAt: source.RetrievedAt, Scope: source.Scope,
			Confidence: string(source.Confidence),
		})
	}
	return result, nil
}

func (s *Service) AnalyzeGeelyProfile(ctx context.Context, profileJSON []byte) (domain.GeelyResearchAnalysis, error) {
	if err := contextError(ctx); err != nil {
		return domain.GeelyResearchAnalysis{}, err
	}
	profile, err := research.ParseProfile(profileJSON)
	if err != nil {
		return domain.GeelyResearchAnalysis{}, invalidArgument("validate Geely research profile: %v", err)
	}
	if err := s.requireGeelyAdapter(ctx, true, false); err != nil {
		return domain.GeelyResearchAnalysis{}, err
	}
	detection, err := s.adapters.Detect(ctx, geelyResearchAdapterID, domain.DetectRequest{Observation: append([]byte(nil), profileJSON...)})
	if err != nil {
		return domain.GeelyResearchAnalysis{}, err
	}
	analysis := domain.GeelyResearchAnalysis{Profile: summarizeGeelyProfile(profile), Detection: detection}
	if missing := missingCompatibilityValues(profile); len(missing) > 0 {
		analysis.Compatibility = domain.CompatibilityResult{
			Status: domain.StatusUnknown,
			Reason: "exact compatibility cannot be checked because the profile is missing: " + strings.Join(missing, ", "),
		}
		return analysis, nil
	}
	profileGateway, ok := s.adapters.(vehicleProfileCompatibilityGateway)
	if !ok {
		return domain.GeelyResearchAnalysis{}, fmt.Errorf("%w: adapter gateway does not support versioned vehicle research profiles", domain.ErrCapabilityUnavailable)
	}
	compatibilityRequest := domain.CompatibilityRequest{
		Identity: domain.VehicleIdentity{Manufacturer: profile.Vehicle.Manufacturer, Model: profile.Vehicle.Model, PlatformID: profile.Infotainment.Platform},
		Hardware: domain.HardwareInfo{PlatformID: profile.Infotainment.HardwareIdentifier, Revision: profile.Infotainment.HardwareRevision},
		Firmware: domain.FirmwareInfo{Version: profile.Infotainment.FirmwareVersion, BuildID: profile.Infotainment.FirmwareBuildID},
	}
	analysis.Compatibility, err = profileGateway.CheckCompatibilityWithVehicleProfile(ctx, geelyResearchAdapterID, compatibilityRequest, append(json.RawMessage(nil), profileJSON...))
	if err != nil {
		return domain.GeelyResearchAnalysis{}, err
	}
	if analysis.Compatibility.Status != domain.StatusSupported && analysis.Compatibility.Status != domain.StatusUnsupported && analysis.Compatibility.Status != domain.StatusUnknown {
		return domain.GeelyResearchAnalysis{}, fmt.Errorf("%w: Geely adapter returned an invalid compatibility status", domain.ErrAdapterUnavailable)
	}
	return analysis, nil
}

func (s *Service) InspectGeelyResources(ctx context.Context, request domain.GeelyResourceRequest) (domain.ResourceInspection, error) {
	if err := contextError(ctx); err != nil {
		return domain.ResourceInspection{}, err
	}
	content, err := validateGeelyResource(request)
	if err != nil {
		return domain.ResourceInspection{}, err
	}
	if err := s.requireGeelyAdapter(ctx, false, true); err != nil {
		return domain.ResourceInspection{}, err
	}
	inspection, err := s.adapters.InspectResources(ctx, geelyResearchAdapterID, domain.InspectResourcesRequest{
		Format: string(request.Format), Filename: request.Filename, Content: string(content),
	})
	if err != nil {
		return domain.ResourceInspection{}, err
	}
	return inspection, nil
}

func (s *Service) ImportGeelyLocalizationProject(ctx context.Context, request domain.GeelyLocalizationImportRequest) (domain.GeelyLocalizationImportResult, error) {
	if err := contextError(ctx); err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	profileJSON := []byte(request.ProfileJSON)
	analysis, err := s.AnalyzeGeelyProfile(ctx, profileJSON)
	if err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	resourceRequest := domain.GeelyResourceRequest{Filename: request.Filename, Format: request.Format, Content: request.Content}
	inspection, err := s.InspectGeelyResources(ctx, resourceRequest)
	if err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	if err := compareGeelyResourceInspection(request.Format, request.Filename, []byte(request.Content), inspection); err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	localizationService, err := s.localizationService()
	if err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	project, err := localizationService.ImportProject(ctx, ImportLocalizationRequest{
		Name: request.Name, Locale: request.Locale, Format: request.Format, Filename: request.Filename, Content: []byte(request.Content),
	})
	if err != nil {
		return domain.GeelyLocalizationImportResult{}, err
	}
	return domain.GeelyLocalizationImportResult{Project: project, Analysis: analysis, Inspection: inspection}, nil
}

func (s *Service) BuildGeelyDiagnosticReport(ctx context.Context, request domain.GeelyDiagnosticReportRequest) (domain.GeelyDiagnosticReport, error) {
	if err := contextError(ctx); err != nil {
		return domain.GeelyDiagnosticReport{}, err
	}
	analysis, err := s.AnalyzeGeelyProfile(ctx, []byte(request.ProfileJSON))
	if err != nil {
		return domain.GeelyDiagnosticReport{}, err
	}
	var filename, format string
	var content []byte
	var keys []string
	if request.Resource != nil {
		resource := *request.Resource
		inspection, err := s.InspectGeelyResources(ctx, resource)
		if err != nil {
			return domain.GeelyDiagnosticReport{}, err
		}
		if err := compareGeelyResourceInspection(resource.Format, resource.Filename, []byte(resource.Content), inspection); err != nil {
			return domain.GeelyDiagnosticReport{}, err
		}
		filename, format, content = resource.Filename, string(resource.Format), []byte(resource.Content)
		keys = make([]string, 0, len(inspection.Resources))
		for _, item := range inspection.Resources {
			keys = append(keys, item.ID)
		}
	}
	_, reportJSON, err := research.BuildDiagnosticReport(
		[]byte(request.ProfileJSON), research.CompatibilityResult{Status: sdk.CompatibilityStatus(analysis.Compatibility.Status), Reason: analysis.Compatibility.Reason},
		filename, format, content, keys, time.Now().UTC(),
	)
	if err != nil {
		return domain.GeelyDiagnosticReport{}, invalidArgument("build Geely diagnostic report: %v", err)
	}
	return domain.GeelyDiagnosticReport{Filename: "carvion-geely-diagnostic-report.json", Content: string(reportJSON)}, nil
}

func (s *Service) requireGeelyAdapter(ctx context.Context, compatibility, resources bool) error {
	info, err := s.adapters.GetInfo(ctx, geelyResearchAdapterID)
	if err != nil {
		return err
	}
	if info.Status != domain.AdapterStatusAvailable || !info.Capabilities.Detect ||
		(compatibility && !info.Capabilities.CheckCompatibility) || (resources && !info.Capabilities.InspectResources) ||
		!containsGeelyInput(info.InputTypes, "vehicle-research-profile-json") {
		return fmt.Errorf("%w: Geely research adapter is unavailable or does not declare required capabilities", domain.ErrCapabilityUnavailable)
	}
	return nil
}

func missingCompatibilityValues(profile research.Profile) []string {
	fields := []struct{ name, value string }{
		{"vehicle.manufacturer", profile.Vehicle.Manufacturer}, {"vehicle.model", profile.Vehicle.Model},
		{"vehicle.model_year", fmt.Sprint(profile.Vehicle.ModelYear)}, {"vehicle.market", profile.Vehicle.Market},
		{"vehicle.trim", profile.Vehicle.Trim}, {"infotainment.platform", profile.Infotainment.Platform},
		{"infotainment.os_name", profile.Infotainment.OSName}, {"infotainment.os_version", profile.Infotainment.OSVersion},
		{"infotainment.hardware_identifier", profile.Infotainment.HardwareIdentifier}, {"infotainment.hardware_revision", profile.Infotainment.HardwareRevision},
		{"infotainment.firmware_version", profile.Infotainment.FirmwareVersion}, {"infotainment.firmware_build_id", profile.Infotainment.FirmwareBuildID},
	}
	var missing []string
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" || field.value == "0" {
			missing = append(missing, field.name)
		}
	}
	return missing
}

func summarizeGeelyProfile(profile research.Profile) domain.GeelyResearchProfile {
	return domain.GeelyResearchProfile{
		Schema: profile.Schema, ProfileID: profile.ProfileID, Manufacturer: profile.Vehicle.Manufacturer, Model: profile.Vehicle.Model,
		ModelYear: profile.Vehicle.ModelYear, Market: profile.Vehicle.Market, Trim: profile.Vehicle.Trim,
		Platform: profile.Infotainment.Platform, OSName: profile.Infotainment.OSName, OSVersion: profile.Infotainment.OSVersion,
		HardwareIdentifier: profile.Infotainment.HardwareIdentifier, HardwareRevision: profile.Infotainment.HardwareRevision,
		FirmwareVersion: profile.Infotainment.FirmwareVersion, FirmwareBuildID: profile.Infotainment.FirmwareBuildID,
		Origin: string(profile.Origin),
	}
}

func validateGeelyResource(request domain.GeelyResourceRequest) ([]byte, error) {
	if strings.TrimSpace(request.Filename) == "" || len(request.Filename) > 255 || strings.ContainsAny(request.Filename, "/\\\r\n\x00") || request.Filename == "." || request.Filename == ".." {
		return nil, invalidArgument("resource filename must be a basename without a path")
	}
	if request.Format != domain.LocalizationFormatAndroidXML && request.Format != domain.LocalizationFormatJSON {
		return nil, invalidArgument("unsupported Geely resource format %q", request.Format)
	}
	content := []byte(request.Content)
	if len(content) == 0 || len(content) > research.MaxResourceContentBytes {
		return nil, invalidArgument("resource must be between 1 byte and %d bytes", research.MaxResourceContentBytes)
	}
	if !utf8.Valid(content) {
		return nil, invalidArgument("resource document is not valid UTF-8")
	}
	return content, nil
}

func compareGeelyResourceInspection(format domain.LocalizationFormat, filename string, content []byte, inspection domain.ResourceInspection) error {
	if _, err := validateGeelyResource(domain.GeelyResourceRequest{Filename: filename, Format: format, Content: string(content)}); err != nil {
		return err
	}
	document, err := localization.Parse(format, filename, content)
	if err != nil {
		return invalidArgument("parse localization resource: %v", err)
	}
	want := make(map[string]string)
	for _, resource := range document.Resources {
		if resource.Present && resource.Translatable {
			want[resource.Key] = resource.SourceText
		}
	}
	got := make(map[string]string, len(inspection.Resources))
	for _, resource := range inspection.Resources {
		if _, exists := got[resource.ID]; exists {
			return fmt.Errorf("%w: Geely adapter returned duplicate resource key %q", domain.ErrAdapterUnavailable, resource.ID)
		}
		got[resource.ID] = resource.Text
	}
	if len(want) != len(got) {
		return fmt.Errorf("%w: Geely adapter resource list does not match Localization Engine parse", domain.ErrAdapterUnavailable)
	}
	for key, text := range want {
		if gotText, exists := got[key]; !exists || gotText != text {
			return fmt.Errorf("%w: Geely adapter resource list does not match Localization Engine parse", domain.ErrAdapterUnavailable)
		}
	}
	return nil
}

func containsGeelyInput(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
