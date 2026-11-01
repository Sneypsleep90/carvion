package simulator

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

const syntheticManufacturer = "CARVION TEST LAB"

const (
	FaultNone     = ""
	FaultRPCError = "rpc-error"
)

var (
	ErrProfileNotFound = errors.New("simulator profile not found")
	profileIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
)

//go:embed testdata/profiles.json testdata/resources-zh-CN.xml
var fixtures embed.FS

type Profile struct {
	Public        domain.SimulatorProfile
	KnownFirmware bool
	FaultMode     string
	resourceXML   []byte
}

type profileCatalog struct {
	Profiles []catalogEntry `json:"profiles"`
}

type catalogEntry struct {
	ID                    string                     `json:"id"`
	Name                  string                     `json:"name"`
	Description           string                     `json:"description"`
	Synthetic             bool                       `json:"synthetic"`
	Identity              catalogVehicleIdentity     `json:"identity"`
	Hardware              catalogHardware            `json:"hardware"`
	Firmware              catalogFirmware            `json:"firmware"`
	ExpectedCompatibility domain.CompatibilityStatus `json:"expectedCompatibility"`
	KnownFirmware         bool                       `json:"known_firmware"`
	FaultMode             string                     `json:"fault_mode,omitempty"`
}

type catalogVehicleIdentity struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	PlatformID   string `json:"platform_id"`
}

type catalogHardware struct {
	PlatformID string `json:"platform_id"`
	Revision   string `json:"revision"`
}

type catalogFirmware struct {
	BuildID string `json:"build_id"`
	Version string `json:"version"`
}

func Profiles() ([]domain.SimulatorProfile, error) {
	loaded, err := loadProfiles()
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.SimulatorProfile, len(loaded))
	for i, profile := range loaded {
		profiles[i] = profile.Public
	}
	return profiles, nil
}

func LoadProfile(id string) (Profile, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return Profile{}, err
	}
	for _, profile := range profiles {
		if profile.Public.ID == id {
			profile.resourceXML = append([]byte(nil), profile.resourceXML...)
			return profile, nil
		}
	}
	return Profile{}, fmt.Errorf("%w: %q", ErrProfileNotFound, id)
}

func (p Profile) ResourceXML() []byte {
	return append([]byte(nil), p.resourceXML...)
}

func loadProfiles() ([]Profile, error) {
	data, err := fixtures.ReadFile("testdata/profiles.json")
	if err != nil {
		return nil, fmt.Errorf("read embedded simulator profiles: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var catalog profileCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("decode embedded simulator profiles: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("embedded simulator profiles contain trailing JSON values")
		}
		return nil, fmt.Errorf("decode trailing simulator profile data: %w", err)
	}
	if len(catalog.Profiles) == 0 {
		return nil, errors.New("embedded simulator profile catalog is empty")
	}
	resourceXML, err := fixtures.ReadFile("testdata/resources-zh-CN.xml")
	if err != nil {
		return nil, fmt.Errorf("read embedded simulator resource fixture: %w", err)
	}
	if len(resourceXML) == 0 || !utf8.Valid(resourceXML) {
		return nil, errors.New("embedded simulator resource fixture must be non-empty UTF-8")
	}

	profiles := make([]Profile, 0, len(catalog.Profiles))
	seen := make(map[string]struct{}, len(catalog.Profiles))
	for _, entry := range catalog.Profiles {
		profile := domain.SimulatorProfile{
			ID:          entry.ID,
			Name:        entry.Name,
			Description: entry.Description,
			Synthetic:   entry.Synthetic,
			Identity: domain.VehicleIdentity{
				Manufacturer: entry.Identity.Manufacturer,
				Model:        entry.Identity.Model,
				PlatformID:   entry.Identity.PlatformID,
			},
			Hardware: domain.HardwareInfo{
				PlatformID: entry.Hardware.PlatformID,
				Revision:   entry.Hardware.Revision,
			},
			Firmware: domain.FirmwareInfo{
				BuildID: entry.Firmware.BuildID,
				Version: entry.Firmware.Version,
			},
			ExpectedCompatibility: entry.ExpectedCompatibility,
		}
		if err := validateProfile(profile, entry.KnownFirmware, entry.FaultMode); err != nil {
			return nil, fmt.Errorf("validate simulator profile %q: %w", profile.ID, err)
		}
		if _, exists := seen[profile.ID]; exists {
			return nil, fmt.Errorf("duplicate simulator profile ID %q", profile.ID)
		}
		seen[profile.ID] = struct{}{}
		profiles = append(profiles, Profile{
			Public:        profile,
			KnownFirmware: entry.KnownFirmware,
			FaultMode:     entry.FaultMode,
			resourceXML:   append([]byte(nil), resourceXML...),
		})
	}
	return profiles, nil
}

func validateProfile(profile domain.SimulatorProfile, knownFirmware bool, faultMode string) error {
	if !profileIDPattern.MatchString(profile.ID) {
		return errors.New("ID must contain lowercase letters, digits, and hyphens")
	}
	if strings.TrimSpace(profile.Name) == "" || strings.TrimSpace(profile.Description) == "" {
		return errors.New("name and description are required")
	}
	if !profile.Synthetic {
		return errors.New("all simulator profiles must be explicitly marked synthetic")
	}
	if profile.Identity.Manufacturer != syntheticManufacturer {
		return errors.New("simulator manufacturer must use the synthetic CARVION TEST LAB marker")
	}
	if strings.TrimSpace(profile.Identity.Model) == "" || strings.TrimSpace(profile.Identity.PlatformID) == "" ||
		profile.Identity.PlatformID != profile.Hardware.PlatformID || strings.TrimSpace(profile.Hardware.Revision) == "" ||
		strings.TrimSpace(profile.Firmware.BuildID) == "" || strings.TrimSpace(profile.Firmware.Version) == "" {
		return errors.New("identity, matching hardware platform, revision, and firmware build/version are required")
	}
	switch profile.ExpectedCompatibility {
	case domain.StatusSupported, domain.StatusUnsupported, domain.StatusUnknown:
	default:
		return fmt.Errorf("invalid expected compatibility status %q", profile.ExpectedCompatibility)
	}
	if knownFirmware == (profile.ExpectedCompatibility == domain.StatusUnknown) && faultMode == FaultNone {
		return errors.New("known_firmware must agree with the declared compatibility outcome")
	}
	if faultMode != FaultNone && faultMode != FaultRPCError {
		return fmt.Errorf("unsupported simulator fault mode %q", faultMode)
	}
	if faultMode == FaultRPCError && profile.ExpectedCompatibility != domain.StatusUnknown {
		return errors.New("communication failure profile must not claim a supported compatibility result")
	}
	return nil
}
