package localization

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

type PackageInput struct {
	ProjectID       string
	ProjectName     string
	Locale          string
	Format          domain.LocalizationFormat
	Filename        string
	SourceSHA256    string
	ResourceContent []byte
	Report          domain.LocalizationValidationReport
}

type packageManifest struct {
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

func BuildPackage(input PackageInput) ([]byte, string, error) {
	if strings.TrimSpace(input.ProjectID) == "" || strings.TrimSpace(input.ProjectName) == "" || strings.TrimSpace(input.Locale) == "" || len(input.ResourceContent) == 0 {
		return nil, "", fmt.Errorf("package requires project identity, locale and resource content")
	}
	if !input.Report.Valid {
		return nil, "", fmt.Errorf("package cannot be built from an invalid validation report")
	}
	filename := path.Base(strings.ReplaceAll(input.Filename, "\\", "/"))
	if filename == "." || filename == "/" || filename == "" {
		return nil, "", fmt.Errorf("resource filename is invalid")
	}
	resourcePath := "resources/" + filename
	manifest := packageManifest{
		Schema: "carvion.language-package/v1", PackageType: "carvion-localization",
		ProjectID: input.ProjectID, ProjectName: input.ProjectName, Locale: input.Locale,
		ResourceFormat: string(input.Format), ResourcePath: resourcePath,
		SourceSHA256: input.SourceSHA256, OEMFirmware: false,
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("encode language package manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	// Volatile run timestamps are deliberately omitted from the package report so
	// identical source, translations and validation results produce identical ZIPs.
	report := input.Report
	report.CheckedAt = time.Time{}
	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, "", fmt.Errorf("encode language package validation: %w", err)
	}
	reportBytes = append(reportBytes, '\n')
	files := map[string][]byte{
		"manifest.json":   manifestBytes,
		"validation.json": reportBytes,
		resourcePath:      append([]byte(nil), input.ResourceContent...),
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var checksums strings.Builder
	for _, name := range names {
		sum := sha256.Sum256(files[name])
		fmt.Fprintf(&checksums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
	}
	files["checksums.sha256"] = []byte(checksums.String())
	names = append(names, "checksums.sha256")
	sort.Strings(names)

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetModTime(time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, "", fmt.Errorf("create package entry %q: %w", name, err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			return nil, "", fmt.Errorf("write package entry %q: %w", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize language package: %w", err)
	}
	sum := sha256.Sum256(output.Bytes())
	return output.Bytes(), hex.EncodeToString(sum[:]), nil
}
