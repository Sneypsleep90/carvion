package geelyresearch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"
)

const (
	DiagnosticReportSchema = "carvion.geely-diagnostic-report/v1"
	maxReportSourceBytes   = 10 << 20
	maxReportResourceKeys  = 4096
)

type DiagnosticReport struct {
	Schema         string              `json:"schema"`
	GeneratedAt    time.Time           `json:"generated_at"`
	Profile        DiagnosticProfile   `json:"profile"`
	ProfileSHA256  string              `json:"profile_sha256"`
	Compatibility  CompatibilityResult `json:"compatibility"`
	PublicEvidence []Evidence          `json:"public_evidence"`
	Resource       *DiagnosticResource `json:"resource,omitempty"`
	Limitations    []string            `json:"limitations"`
}

// DiagnosticProfile adds trusted import provenance only in the report output.
// The input JSON cannot set this value because Profile.Origin is not decoded.
type DiagnosticProfile struct {
	Profile
	Origin string `json:"origin"`
}

type DiagnosticResource struct {
	Filename string   `json:"filename"`
	Format   string   `json:"format"`
	SHA256   string   `json:"sha256"`
	Keys     []string `json:"keys"`
}

// BuildDiagnosticReport creates a report containing bounded metadata and
// hashes only. It never embeds the source file contents or filesystem path.
func BuildDiagnosticReport(
	profileJSON []byte,
	compatibility CompatibilityResult,
	resourceFilename string,
	resourceFormat string,
	resourceContent []byte,
	resourceKeys []string,
	generatedAt time.Time,
) (DiagnosticReport, []byte, error) {
	profile, err := ParseProfile(profileJSON)
	if err != nil {
		return DiagnosticReport{}, nil, fmt.Errorf("build diagnostic report from invalid profile: %w", err)
	}
	if compatibility.Status != "supported" && compatibility.Status != "unsupported" && compatibility.Status != "unknown" {
		return DiagnosticReport{}, nil, errors.New("diagnostic report has an invalid compatibility status")
	}
	if strings.TrimSpace(compatibility.Reason) == "" || len(compatibility.Reason) > 512 || containsControl(compatibility.Reason) || containsSensitiveMaterial(compatibility.Reason) {
		return DiagnosticReport{}, nil, errors.New("diagnostic report requires a bounded compatibility reason")
	}
	if len(resourceFilename) == 0 && len(resourceContent) == 0 && resourceFormat == "" && len(resourceKeys) == 0 {
		resourceFilename = ""
	} else {
		basename, err := safeBasename(resourceFilename)
		if err != nil {
			return DiagnosticReport{}, nil, err
		}
		if resourceFormat != "android-strings-xml" && resourceFormat != "carvion-json-v1" {
			return DiagnosticReport{}, nil, fmt.Errorf("unsupported diagnostic resource format %q", resourceFormat)
		}
		if len(resourceContent) == 0 || len(resourceContent) > maxReportSourceBytes {
			return DiagnosticReport{}, nil, errors.New("diagnostic resource is empty or exceeds the size limit")
		}
		if len(resourceKeys) > maxReportResourceKeys {
			return DiagnosticReport{}, nil, errors.New("diagnostic resource contains too many keys")
		}
		keys := append([]string(nil), resourceKeys...)
		seen := make(map[string]struct{}, len(keys))
		for _, key := range keys {
			if strings.TrimSpace(key) == "" || len(key) > 512 || containsControl(key) || containsSensitiveMaterial(key) {
				return DiagnosticReport{}, nil, errors.New("diagnostic resource contains an invalid key")
			}
			if _, exists := seen[key]; exists {
				return DiagnosticReport{}, nil, fmt.Errorf("diagnostic resource contains duplicate key %q", key)
			}
			seen[key] = struct{}{}
		}
		resourceFilename = basename
	}

	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}
	report := DiagnosticReport{
		Schema:         DiagnosticReportSchema,
		GeneratedAt:    generatedAt.UTC(),
		Profile:        DiagnosticProfile{Profile: profile, Origin: string(ProfileOriginImportedFile)},
		ProfileSHA256:  digest(profileJSON),
		Compatibility:  compatibility,
		PublicEvidence: PublicEvidence(),
		Limitations: []string{
			"Profile values came from a user-selected file and have not been independently verified.",
			"This report does not represent detection of or a connection to a real vehicle.",
			"Compatibility requires an exact reviewed catalog record; absence of a record is unknown.",
			"This report does not authorize or describe firmware installation or modification.",
		},
	}
	if resourceFilename != "" {
		report.Resource = &DiagnosticResource{
			Filename: resourceFilename,
			Format:   resourceFormat,
			SHA256:   digest(resourceContent),
			Keys:     append([]string(nil), resourceKeys...),
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return DiagnosticReport{}, nil, fmt.Errorf("encode diagnostic report: %w", err)
	}
	data = append(data, '\n')
	return report, data, nil
}

func safeBasename(filename string) (string, error) {
	if strings.TrimSpace(filename) == "" || len(filename) > 4096 || containsControl(filename) || containsSensitiveMaterial(filename) {
		return "", errors.New("diagnostic resource filename is invalid")
	}
	basename := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	if basename == "." || basename == ".." || strings.TrimSpace(basename) == "" || len(basename) > 255 {
		return "", errors.New("diagnostic resource filename is invalid")
	}
	for _, char := range basename {
		if unicode.IsControl(char) {
			return "", errors.New("diagnostic resource filename is invalid")
		}
	}
	return basename, nil
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
