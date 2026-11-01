package application

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/simulator"
)

const (
	simulatorAdapterResourceFormat = "android-strings-xml"
	simulatorTargetLocale          = "ru-RU"
	simulatorRunTimeout            = 2 * time.Minute
)

var simulatorTranslations = map[string]string{
	"app_title":           "Демо мультимедийная система CARVION",
	"welcome_title":       "Здравствуйте, добро пожаловать в демонстрационную систему",
	"audio_mode_standard": "Стандартный режим",
}

type simulatorObservation struct {
	ProfileID string `json:"profile_id"`
	Identity  struct {
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		PlatformID   string `json:"platform_id"`
	} `json:"identity"`
}

type simulatorPackageManifest struct {
	Schema         string `json:"schema"`
	PackageType    string `json:"package_type"`
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name"`
	Locale         string `json:"locale"`
	ResourceFormat string `json:"resource_format"`
	ResourcePath   string `json:"resource_path"`
	SourceSHA256   string `json:"source_sha256"`
	OEMFirmware    bool   `json:"oem_firmware"`
}

func (s *Service) ListSimulatorProfiles(ctx context.Context) ([]domain.SimulatorProfile, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	profiles, err := simulator.Profiles()
	if err != nil {
		return nil, fmt.Errorf("list synthetic simulator profiles: %w", err)
	}
	return profiles, nil
}

// RunSimulatorScenario executes a bounded, local-only simulator scenario. Runtime
// failures are returned as structured events so Desktop can show the last step;
// invalid input and startup context cancellation remain Go errors.
func (s *Service) RunSimulatorScenario(ctx context.Context, profileID string) (domain.SimulatorRunResult, error) {
	if ctx == nil {
		return domain.SimulatorRunResult{}, invalidArgument("context is required")
	}
	ctx, cancel := context.WithTimeout(ctx, simulatorRunTimeout)
	defer cancel()
	if err := contextError(ctx); err != nil {
		return domain.SimulatorRunResult{}, err
	}
	if err := validateField("profile_id", profileID); err != nil {
		return domain.SimulatorRunResult{}, err
	}
	profile, err := simulator.LoadProfile(profileID)
	if err != nil {
		if errors.Is(err, simulator.ErrProfileNotFound) {
			return domain.SimulatorRunResult{}, invalidArgument("unknown simulator profile %q", profileID)
		}
		return domain.SimulatorRunResult{}, fmt.Errorf("load simulator profile: %w", err)
	}
	runID, err := newSimulatorRunID()
	if err != nil {
		return domain.SimulatorRunResult{}, fmt.Errorf("create simulator run ID: %w", err)
	}
	result := domain.SimulatorRunResult{
		RunID:   runID,
		Profile: profile.Public,
		Status:  domain.SimulatorRunFailed,
		Events:  []domain.SimulatorEvent{},
	}
	addEvent := func(step, status, message string) {
		result.Events = append(result.Events, domain.SimulatorEvent{At: time.Now().UTC(), Step: step, Status: status, Message: message})
	}
	fail := func(step string, cause error) (domain.SimulatorRunResult, error) {
		if ctx.Err() != nil {
			step = "scenario.cancelled"
			cause = ctx.Err()
		}
		result.Status = domain.SimulatorRunFailed
		result.ErrorMessage = cause.Error()
		addEvent(step, "failed", cause.Error())
		return result, nil
	}
	addEvent("scenario.started", "completed", "Запущен локальный сценарий на синтетическом профиле.")

	if err := ctx.Err(); err != nil {
		return fail("scenario.cancelled", err)
	}
	adapters, err := s.adapters.List(ctx)
	if err != nil {
		return fail("adapters.discovered", fmt.Errorf("discover adapters: %w", err))
	}
	if err := ctx.Err(); err != nil {
		return fail("scenario.cancelled", err)
	}
	selected, detection, probeErr := selectSimulatorAdapter(ctx, s.adapters, adapters, profile)
	if probeErr != nil {
		return fail("adapter.detect", probeErr)
	}
	if selected.ID == "" {
		result.Status = domain.SimulatorRunUnknown
		result.Compatibility = domain.CompatibilityResult{Status: domain.StatusUnknown, Reason: "no installed adapter detected this synthetic profile"}
		addEvent("device.detected", "unknown", result.Compatibility.Reason)
		addEvent("scenario.stopped", "unknown", "Сценарий остановлен до проверки ресурсов.")
		return result, nil
	}
	if detection.Status != domain.StatusSupported {
		result.Status = domain.SimulatorRunUnknown
		result.Compatibility = domain.CompatibilityResult{Status: domain.StatusUnknown, Reason: detection.Reason}
		addEvent("device.detected", "unknown", detection.Reason)
		addEvent("scenario.stopped", "unknown", "Адаптер не подтвердил синтетическое устройство.")
		return result, nil
	}
	result.AdapterID = selected.ID
	addEvent("device.detected", "completed", "Обнаружено только синтетическое устройство из локального каталога.")
	addEvent("adapter.selected", "completed", fmt.Sprintf("Выбран локальный адаптер %s.", selected.ID))

	if err := ctx.Err(); err != nil {
		return fail("scenario.cancelled", err)
	}
	compatibilityRequest := domain.CompatibilityRequest{
		Identity: profile.Public.Identity,
		Hardware: profile.Public.Hardware,
		Firmware: profile.Public.Firmware,
	}
	compatibility, err := s.adapters.CheckCompatibility(ctx, selected.ID, compatibilityRequest)
	if err != nil {
		return fail("compatibility.checked", fmt.Errorf("check synthetic profile compatibility: %w", err))
	}
	result.Compatibility = compatibility
	addEvent("compatibility.checked", string(compatibility.Status), compatibility.Reason)
	switch compatibility.Status {
	case domain.StatusUnsupported:
		result.Status = domain.SimulatorRunUnsupported
		addEvent("scenario.stopped", "unsupported", "Ресурсы и пакет не создавались для несовместимого профиля.")
		return result, nil
	case domain.StatusUnknown:
		result.Status = domain.SimulatorRunUnknown
		addEvent("scenario.stopped", "unknown", "Неизвестная сборка не прошла дальше проверки совместимости.")
		return result, nil
	case domain.StatusSupported:
	default:
		return fail("compatibility.checked", fmt.Errorf("adapter returned invalid compatibility status %q", compatibility.Status))
	}

	if err := ctx.Err(); err != nil {
		return fail("scenario.cancelled", err)
	}
	resourceXML := profile.ResourceXML()
	inspection, err := s.adapters.InspectResources(ctx, selected.ID, domain.InspectResourcesRequest{
		Format: simulatorAdapterResourceFormat, Filename: "strings.xml", Content: string(resourceXML),
	})
	if err != nil {
		return fail("resources.inspected", fmt.Errorf("inspect synthetic Android resources: %w", err))
	}
	if len(inspection.Resources) == 0 {
		return fail("resources.inspected", errors.New("adapter returned no translatable resources from the synthetic fixture"))
	}
	addEvent("resources.inspected", "completed", fmt.Sprintf("Адаптер распознал %d тестовых Android-строк.", len(inspection.Resources)))

	project, err := s.ImportLocalizationProject(ctx, ImportLocalizationRequest{
		Name: "CARVION Simulator Demo", Locale: simulatorTargetLocale,
		Format: domain.LocalizationFormatAndroidXML, Filename: "strings.xml", Content: resourceXML,
	})
	if err != nil {
		return fail("localization.imported", fmt.Errorf("import synthetic localization resources: %w", err))
	}
	result.LocalizationProjectID = project.ID
	if err := matchInspectedResources(project.Resources, inspection.Resources); err != nil {
		return fail("resources.verified", err)
	}
	addEvent("localization.imported", "completed", fmt.Sprintf("Импортирован тестовый проект %s.", project.ID))

	for _, resource := range project.Resources {
		if !resource.Present || !resource.Translatable {
			continue
		}
		translation, ok := simulatorTranslations[resource.Key]
		if !ok {
			return fail("translations.prepared", fmt.Errorf("no fixed simulator translation exists for resource %q", resource.Key))
		}
		if _, err := s.UpdateLocalizationTranslation(ctx, project.ID, resource.Key, translation); err != nil {
			return fail("translations.prepared", fmt.Errorf("prepare synthetic translation %q: %w", resource.Key, err))
		}
		if _, err := s.ApproveLocalizationTranslation(ctx, project.ID, resource.Key); err != nil {
			return fail("translations.prepared", fmt.Errorf("approve synthetic translation %q: %w", resource.Key, err))
		}
	}
	addEvent("translations.prepared", "completed", "Подготовлены и утверждены фиксированные демонстрационные переводы.")

	report, err := s.ValidateLocalizationProject(ctx, project.ID)
	if err != nil {
		return fail("translations.validated", fmt.Errorf("validate synthetic translations: %w", err))
	}
	if !report.Valid {
		return fail("translations.validated", errors.New("synthetic translation validation failed"))
	}
	addEvent("translations.validated", "completed", "Проверка переводов завершилась без ошибок.")

	item, err := s.BuildLocalizationPackage(ctx, project.ID)
	if err != nil {
		return fail("package.built", fmt.Errorf("build CARVION localization package: %w", err))
	}
	result.Package = item
	addEvent("package.built", "completed", fmt.Sprintf("Собран CARVION-пакет %s.", item.Filename))
	if err := verifySimulatorPackage(item); err != nil {
		return fail("package.verified", err)
	}
	result.PackageVerified = true
	addEvent("package.verified", "completed", "ZIP, SHA-256, ресурсы и non-OEM manifest проверены.")
	result.Status = domain.SimulatorRunCompleted
	addEvent("scenario.completed", "completed", "Локальный сценарий завершён; установка в автомобиль не выполнялась.")
	return result, nil
}

func selectSimulatorAdapter(ctx context.Context, gateway AdapterGateway, adapters []domain.AdapterInfo, profile simulator.Profile) (domain.AdapterInfo, domain.DetectionResult, error) {
	candidates := make([]domain.AdapterInfo, 0, len(adapters))
	for _, adapter := range adapters {
		if adapter.Status != domain.AdapterStatusAvailable || !adapter.Capabilities.Detect ||
			!adapter.Capabilities.CheckCompatibility || !adapter.Capabilities.InspectResources ||
			!containsString(adapter.ResourceFormats, simulatorAdapterResourceFormat) {
			continue
		}
		candidates = append(candidates, adapter)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	var probeErrors []error
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return domain.AdapterInfo{}, domain.DetectionResult{}, err
		}
		observation := simulatorObservation{ProfileID: profile.Public.ID}
		observation.Identity.Manufacturer = profile.Public.Identity.Manufacturer
		observation.Identity.Model = profile.Public.Identity.Model
		observation.Identity.PlatformID = profile.Public.Identity.PlatformID
		content, err := json.Marshal(observation)
		if err != nil {
			return domain.AdapterInfo{}, domain.DetectionResult{}, fmt.Errorf("encode synthetic observation: %w", err)
		}
		detection, err := gateway.Detect(ctx, candidate.ID, domain.DetectRequest{Observation: content})
		if err != nil {
			probeErrors = append(probeErrors, fmt.Errorf("adapter %s detection: %w", candidate.ID, err))
			continue
		}
		if detection.Status != domain.StatusSupported || !sameSimulatorIdentity(detection.Identity, profile.Public.Identity) {
			continue
		}
		return candidate, detection, nil
	}
	if len(probeErrors) > 0 {
		return domain.AdapterInfo{}, domain.DetectionResult{}, errors.Join(probeErrors...)
	}
	return domain.AdapterInfo{}, domain.DetectionResult{}, nil
}

func sameSimulatorIdentity(left domain.VehicleIdentity, right domain.VehicleIdentity) bool {
	return left.Manufacturer == right.Manufacturer && left.Model == right.Model && left.PlatformID == right.PlatformID
}

func matchInspectedResources(resources []domain.LocalizationResource, inspected []domain.TranslatableResource) error {
	if len(resources) != len(inspected) {
		return fmt.Errorf("adapter reported %d resources but Core parsed %d from the same fixture", len(inspected), len(resources))
	}
	for i, resource := range resources {
		if !resource.Present || !resource.Translatable || inspected[i].ID != resource.Key || inspected[i].Text != resource.SourceText {
			return fmt.Errorf("adapter resource inspection differs from Core parsing at resource %d", i)
		}
	}
	return nil
}

func verifySimulatorPackage(item domain.LocalizationPackage) error {
	content, err := base64.StdEncoding.DecodeString(item.ContentBase64)
	if err != nil {
		return fmt.Errorf("decode generated language package: %w", err)
	}
	if len(content) == 0 || len(content) != item.Size {
		return errors.New("generated language package size does not match its content")
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != item.SHA256 {
		return errors.New("generated language package SHA-256 does not match its content")
	}
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return fmt.Errorf("open generated language package ZIP: %w", err)
	}
	files := make(map[string][]byte, len(archive.File))
	for _, entry := range archive.File {
		if _, exists := files[entry.Name]; exists {
			return fmt.Errorf("generated language package contains duplicate entry %q", entry.Name)
		}
		if path.IsAbs(entry.Name) || strings.Contains(entry.Name, "..") {
			return fmt.Errorf("generated language package contains unsafe entry %q", entry.Name)
		}
		reader, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open generated language package entry %q: %w", entry.Name, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, 10<<20))
		closeErr := reader.Close()
		if readErr != nil {
			return fmt.Errorf("read generated language package entry %q: %w", entry.Name, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close generated language package entry %q: %w", entry.Name, closeErr)
		}
		files[entry.Name] = data
	}
	manifestData, ok := files["manifest.json"]
	if !ok {
		return errors.New("generated language package is missing manifest.json")
	}
	var manifest simulatorPackageManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("decode generated language package manifest: %w", err)
	}
	if manifest.Schema != "carvion.language-package/v1" || manifest.PackageType != "carvion-localization" ||
		manifest.Locale != simulatorTargetLocale || manifest.ResourceFormat != string(domain.LocalizationFormatAndroidXML) || manifest.OEMFirmware {
		return errors.New("generated language package manifest is invalid or claims to be OEM firmware")
	}
	resourcePath := path.Clean(manifest.ResourcePath)
	if resourcePath != "resources/strings.xml" || files[resourcePath] == nil {
		return errors.New("generated language package is missing the expected Android strings resource")
	}
	if !strings.Contains(string(files[resourcePath]), "Стандартный режим") || strings.Contains(string(files[resourcePath]), "标准模式") {
		return errors.New("generated language package does not contain the prepared synthetic Russian resources")
	}
	validationData, ok := files["validation.json"]
	if !ok {
		return errors.New("generated language package is missing validation.json")
	}
	var report domain.LocalizationValidationReport
	if err := json.Unmarshal(validationData, &report); err != nil || !report.Valid {
		return errors.New("generated language package does not contain a valid validation report")
	}
	if err := verifyPackageChecksums(files); err != nil {
		return err
	}
	return nil
}

func verifyPackageChecksums(files map[string][]byte) error {
	checksums, ok := files["checksums.sha256"]
	if !ok {
		return errors.New("generated language package is missing checksums.sha256")
	}
	verified := 0
	for _, line := range strings.Split(strings.TrimSpace(string(checksums)), "\n") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || parts[1] == "checksums.sha256" {
			return errors.New("generated language package contains a malformed checksum entry")
		}
		content, exists := files[parts[1]]
		if !exists {
			return fmt.Errorf("generated language package checksum references missing file %q", parts[1])
		}
		actual := sha256.Sum256(content)
		if hex.EncodeToString(actual[:]) != parts[0] {
			return fmt.Errorf("generated language package checksum mismatch for %q", parts[1])
		}
		verified++
	}
	if verified != len(files)-1 {
		return errors.New("generated language package checksums do not cover every package file")
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func newSimulatorRunID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}
