package geelyresearch

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const CompatibilityCatalogSchema = "carvion.geely-compatibility/v1"

//go:embed compatibility_records.json
var compatibilityCatalogJSON []byte

// exactCompatibilityEvidence is deliberately empty until an official source
// establishes an exact model/year/market/trim/platform/OS/hardware/build tuple.
// General product and naming evidence cannot authorize a support assertion.
var exactCompatibilityEvidence = map[string]struct{}{}

type CompatibilityKey struct {
	Manufacturer       string `json:"manufacturer"`
	Model              string `json:"model"`
	ModelYear          int    `json:"model_year"`
	Market             string `json:"market"`
	Trim               string `json:"trim"`
	Platform           string `json:"platform"`
	OSName             string `json:"os_name"`
	OSVersion          string `json:"os_version"`
	HardwareIdentifier string `json:"hardware_identifier"`
	HardwareRevision   string `json:"hardware_revision"`
	FirmwareVersion    string `json:"firmware_version"`
	FirmwareBuildID    string `json:"firmware_build_id"`
}

type CompatibilityRecord struct {
	ID         string                  `json:"id"`
	Key        CompatibilityKey        `json:"key"`
	Status     sdk.CompatibilityStatus `json:"status"`
	Reason     string                  `json:"reason"`
	Evidence   []string                `json:"evidence"`
	ReviewedAt string                  `json:"reviewed_at"`
}

type compatibilityCatalog struct {
	Schema  string                `json:"schema"`
	Records []CompatibilityRecord `json:"records"`
}

type CompatibilityResult struct {
	Status   sdk.CompatibilityStatus `json:"status"`
	Reason   string                  `json:"reason"`
	RecordID string                  `json:"record_id,omitempty"`
}

// CompatibilityKey returns the exact tuple used for a verified catalog match.
func (p Profile) CompatibilityKey() CompatibilityKey {
	return CompatibilityKey{
		Manufacturer:       p.Vehicle.Manufacturer,
		Model:              p.Vehicle.Model,
		ModelYear:          p.Vehicle.ModelYear,
		Market:             p.Vehicle.Market,
		Trim:               p.Vehicle.Trim,
		Platform:           p.Infotainment.Platform,
		OSName:             p.Infotainment.OSName,
		OSVersion:          p.Infotainment.OSVersion,
		HardwareIdentifier: p.Infotainment.HardwareIdentifier,
		HardwareRevision:   p.Infotainment.HardwareRevision,
		FirmwareVersion:    p.Infotainment.FirmwareVersion,
		FirmwareBuildID:    p.Infotainment.FirmwareBuildID,
	}
}

func (key CompatibilityKey) complete() bool {
	return strings.TrimSpace(key.Manufacturer) != "" &&
		strings.TrimSpace(key.Model) != "" && key.ModelYear != 0 &&
		strings.TrimSpace(key.Market) != "" && strings.TrimSpace(key.Trim) != "" &&
		strings.TrimSpace(key.Platform) != "" && strings.TrimSpace(key.OSName) != "" &&
		strings.TrimSpace(key.OSVersion) != "" && strings.TrimSpace(key.HardwareIdentifier) != "" &&
		strings.TrimSpace(key.HardwareRevision) != "" && strings.TrimSpace(key.FirmwareVersion) != "" &&
		strings.TrimSpace(key.FirmwareBuildID) != ""
}

func (key CompatibilityKey) exactMatch(other CompatibilityKey) bool {
	return key.complete() && other.complete() && key == other
}

// LoadCompatibilityRecords loads the locally curated, reviewed allow/deny
// records. User profile files cannot add or change this catalog.
func LoadCompatibilityRecords() ([]CompatibilityRecord, error) {
	decoder := json.NewDecoder(bytes.NewReader(compatibilityCatalogJSON))
	decoder.DisallowUnknownFields()
	var catalog compatibilityCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("decode Geely compatibility catalog: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("compatibility catalog contains multiple JSON values")
		}
		return nil, fmt.Errorf("decode trailing compatibility catalog data: %w", err)
	}
	if catalog.Schema != CompatibilityCatalogSchema || catalog.Records == nil {
		return nil, errors.New("Geely compatibility catalog has an invalid schema or records array")
	}
	knownEvidence := evidenceIDs()
	seen := make(map[string]struct{}, len(catalog.Records))
	for _, record := range catalog.Records {
		if err := validateRecord(record, knownEvidence); err != nil {
			return nil, fmt.Errorf("invalid compatibility record %q: %w", record.ID, err)
		}
		if _, exists := seen[record.ID]; exists {
			return nil, fmt.Errorf("duplicate compatibility record ID %q", record.ID)
		}
		seen[record.ID] = struct{}{}
	}
	return append([]CompatibilityRecord(nil), catalog.Records...), nil
}

// CheckCompatibility returns supported or unsupported only for a complete,
// exact key match to a reviewed record. Missing, conflicting, or unknown data
// is never interpreted as an incompatibility.
func CheckCompatibility(profile Profile, records []CompatibilityRecord) CompatibilityResult {
	key := profile.CompatibilityKey()
	if !key.complete() {
		return CompatibilityResult{Status: sdk.StatusUnknown, Reason: "profile is missing fields required for an exact compatibility match"}
	}
	for _, record := range records {
		if !isValidRecordStatus(record.Status) || strings.TrimSpace(record.ID) == "" ||
			strings.TrimSpace(record.Reason) == "" || strings.TrimSpace(record.ReviewedAt) == "" || len(record.Evidence) == 0 {
			continue
		}
		if key.exactMatch(record.Key) {
			return CompatibilityResult{Status: record.Status, Reason: record.Reason, RecordID: record.ID}
		}
	}
	return CompatibilityResult{Status: sdk.StatusUnknown, Reason: "no exact reviewed compatibility record matches this profile"}
}

func validateRecord(record CompatibilityRecord, knownEvidence map[string]struct{}) error {
	if strings.TrimSpace(record.ID) == "" || len(record.ID) > 128 {
		return errors.New("record ID is required and must be at most 128 bytes")
	}
	if !isValidRecordStatus(record.Status) {
		return fmt.Errorf("status %q is not a reviewed support or exclusion status", record.Status)
	}
	if strings.TrimSpace(record.Reason) == "" || len(record.Reason) > 512 {
		return errors.New("record reason is required and must be at most 512 bytes")
	}
	if strings.TrimSpace(record.ReviewedAt) == "" {
		return errors.New("reviewed_at is required")
	}
	if !record.Key.complete() {
		return errors.New("record key must specify every exact-match field")
	}
	if len(record.Evidence) == 0 {
		return errors.New("at least one reviewed evidence ID is required")
	}
	seen := make(map[string]struct{}, len(record.Evidence))
	for _, id := range record.Evidence {
		if _, ok := knownEvidence[id]; !ok {
			return fmt.Errorf("evidence ID %q is not in the local public evidence catalog", id)
		}
		if _, ok := exactCompatibilityEvidence[id]; !ok {
			return fmt.Errorf("evidence ID %q does not establish an exact compatibility tuple", id)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate evidence ID %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func isValidRecordStatus(status sdk.CompatibilityStatus) bool {
	return status == sdk.StatusSupported || status == sdk.StatusUnsupported
}
