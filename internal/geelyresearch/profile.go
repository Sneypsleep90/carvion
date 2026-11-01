package geelyresearch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	ProfileSchema           = "carvion.geely-research/v1"
	maxProfileBytes         = 128 << 10
	maxProfileSources       = 32
	MaxResourceContentBytes = 128 << 10
)

var profileIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
var vehicleIdentifierPattern = regexp.MustCompile(`(?i)(^|[^A-Z0-9])[A-HJ-NPR-Z0-9]{17}([^A-Z0-9]|$)`)
var credentialAssignmentPattern = regexp.MustCompile(`(?i)\b(password|passwd|secret|token|access[_ -]?token|refresh[_ -]?token|api[_ -]?key|authorization|bearer|credential)\s*[:=]`)

type ProfileOrigin string

const ProfileOriginImportedFile ProfileOrigin = "imported-file-unverified"

// Profile contains values reported by an imported local file. Those values do
// not become verified facts merely because a profile contains a citation.
type Profile struct {
	Schema       string              `json:"schema"`
	ProfileID    string              `json:"profile_id"`
	Vehicle      VehicleProfile      `json:"vehicle"`
	Infotainment InfotainmentProfile `json:"infotainment"`
	Sources      []ProfileSource     `json:"sources,omitempty"`
	Origin       ProfileOrigin       `json:"-"`
}

type VehicleProfile struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	ModelYear    int    `json:"model_year,omitempty"`
	Market       string `json:"market,omitempty"`
	Trim         string `json:"trim,omitempty"`
}

type InfotainmentProfile struct {
	Platform           string `json:"platform,omitempty"`
	OSName             string `json:"os_name,omitempty"`
	OSVersion          string `json:"os_version,omitempty"`
	HardwareIdentifier string `json:"hardware_identifier,omitempty"`
	HardwareRevision   string `json:"hardware_revision,omitempty"`
	FirmwareVersion    string `json:"firmware_version,omitempty"`
	FirmwareBuildID    string `json:"firmware_build_id,omitempty"`
}

// ProfileSource is a citation supplied by the profile author. It is displayed
// as an unverified reference and is never fetched by CARVION.
type ProfileSource struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Publisher   string `json:"publisher"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at"`
	RetrievedAt string `json:"retrieved_at"`
	Scope       string `json:"scope"`
}

// ParseProfile validates a versioned, UTF-8 JSON observation. Unknown fields
// are rejected so identifiers such as VIN or credentials cannot be smuggled
// into the Geely research flow.
func ParseProfile(data []byte) (Profile, error) {
	if len(data) == 0 || len(data) > maxProfileBytes {
		return Profile{}, fmt.Errorf("profile is empty or exceeds %d bytes", maxProfileBytes)
	}
	if !utf8.Valid(data) {
		return Profile{}, errors.New("profile is not valid UTF-8")
	}
	if err := rejectDuplicateJSONFields(data); err != nil {
		return Profile{}, fmt.Errorf("validate profile JSON: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var profile Profile
	if err := decoder.Decode(&profile); err != nil {
		return Profile{}, fmt.Errorf("decode Geely research profile: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return Profile{}, errors.New("profile contains multiple JSON values")
		}
		return Profile{}, fmt.Errorf("decode trailing profile data: %w", err)
	}
	if err := validateProfile(profile); err != nil {
		return Profile{}, err
	}
	profile.Origin = ProfileOriginImportedFile
	return profile, nil
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON contains trailing values")
		}
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("JSON object contains duplicate field %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return errors.New("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return errors.New("JSON array is not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func validateProfile(profile Profile) error {
	if profile.Schema != ProfileSchema {
		return fmt.Errorf("unsupported Geely research profile schema %q", profile.Schema)
	}
	if !profileIDPattern.MatchString(profile.ProfileID) {
		return errors.New("profile_id must contain lowercase letters, digits, and hyphens")
	}
	if containsSensitiveMaterial(profile.ProfileID) {
		return errors.New("profile_id appears to contain a vehicle identifier or credential material")
	}
	fields := []struct {
		name  string
		value string
		limit int
	}{
		{"vehicle.manufacturer", profile.Vehicle.Manufacturer, 128},
		{"vehicle.model", profile.Vehicle.Model, 128},
		{"vehicle.market", profile.Vehicle.Market, 64},
		{"vehicle.trim", profile.Vehicle.Trim, 128},
		{"infotainment.platform", profile.Infotainment.Platform, 128},
		{"infotainment.os_name", profile.Infotainment.OSName, 128},
		{"infotainment.os_version", profile.Infotainment.OSVersion, 128},
		{"infotainment.hardware_identifier", profile.Infotainment.HardwareIdentifier, 128},
		{"infotainment.hardware_revision", profile.Infotainment.HardwareRevision, 128},
		{"infotainment.firmware_version", profile.Infotainment.FirmwareVersion, 128},
		{"infotainment.firmware_build_id", profile.Infotainment.FirmwareBuildID, 128},
	}
	for _, field := range fields {
		if len(field.value) > field.limit {
			return fmt.Errorf("%s exceeds %d bytes", field.name, field.limit)
		}
		if containsControl(field.value) {
			return fmt.Errorf("%s contains unsupported control characters", field.name)
		}
		if containsSensitiveMaterial(field.value) {
			return fmt.Errorf("%s appears to contain a vehicle identifier or credential material", field.name)
		}
	}
	if profile.Vehicle.ModelYear != 0 && (profile.Vehicle.ModelYear < 1900 || profile.Vehicle.ModelYear > 2200) {
		return errors.New("vehicle.model_year is outside the accepted range")
	}
	if len(profile.Sources) > maxProfileSources {
		return fmt.Errorf("profile contains more than %d source references", maxProfileSources)
	}
	seenSources := make(map[string]struct{}, len(profile.Sources))
	for i, source := range profile.Sources {
		if !profileIDPattern.MatchString(source.ID) {
			return fmt.Errorf("sources[%d].id must contain lowercase letters, digits, and hyphens", i)
		}
		if containsSensitiveMaterial(source.ID) {
			return fmt.Errorf("sources[%d].id appears to contain a vehicle identifier or credential material", i)
		}
		if _, exists := seenSources[source.ID]; exists {
			return fmt.Errorf("duplicate profile source ID %q", source.ID)
		}
		seenSources[source.ID] = struct{}{}
		for _, field := range []struct {
			name  string
			value string
			limit int
		}{
			{fmt.Sprintf("sources[%d].title", i), source.Title, 256},
			{fmt.Sprintf("sources[%d].publisher", i), source.Publisher, 128},
			{fmt.Sprintf("sources[%d].url", i), source.URL, 2048},
			{fmt.Sprintf("sources[%d].published_at", i), source.PublishedAt, 32},
			{fmt.Sprintf("sources[%d].retrieved_at", i), source.RetrievedAt, 32},
			{fmt.Sprintf("sources[%d].scope", i), source.Scope, 1024},
		} {
			if strings.TrimSpace(field.value) == "" {
				return fmt.Errorf("%s is required", field.name)
			}
			if len(field.value) > field.limit || containsControl(field.value) {
				return fmt.Errorf("%s is invalid or too long", field.name)
			}
			if containsSensitiveMaterial(field.value) {
				return fmt.Errorf("%s appears to contain credential material", field.name)
			}
		}
		parsedURL, err := url.Parse(source.URL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
			return fmt.Errorf("sources[%d].url must be a public HTTPS URL without credentials, query, or fragment", i)
		}
	}
	return nil
}

func containsSensitiveMaterial(value string) bool {
	return vehicleIdentifierPattern.MatchString(value) || credentialAssignmentPattern.MatchString(value)
}

func containsControl(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}
