package deployment

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/localization"
)

const (
	MaxDeploymentPackageBytes      = 16 << 20
	maxDeploymentPackageEntries    = 4
	maxDeploymentUncompressedBytes = 16 << 20
	maxDeploymentManifestBytes     = 64 << 10
	maxDeploymentValidationBytes   = 2 << 20
	maxDeploymentChecksumsBytes    = 4 << 10
	maxDeploymentResourceBytes     = 10 << 20
)

type PackageManifest struct {
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

type VerifiedPackage struct {
	SHA256   string          `json:"sha256"`
	Manifest PackageManifest `json:"manifest"`
}

type packageManifestWire struct {
	Schema         string `json:"schema"`
	PackageType    string `json:"package_type"`
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name"`
	Locale         string `json:"locale"`
	ResourceFormat string `json:"resource_format"`
	ResourcePath   string `json:"resource_path"`
	SourceSHA256   string `json:"source_sha256"`
	OEMFirmware    *bool  `json:"oem_firmware"`
}

func VerifyLanguagePackage(artifact []byte, expectedSHA256 string) (VerifiedPackage, error) {
	if len(artifact) == 0 || len(artifact) > MaxDeploymentPackageBytes {
		return VerifiedPackage{}, fmt.Errorf("language package must be between 1 byte and %d bytes", MaxDeploymentPackageBytes)
	}
	if !isSHA256(expectedSHA256) {
		return VerifiedPackage{}, errors.New("expected language package SHA-256 is invalid")
	}
	actualPackageHash := packageSHA256(artifact)
	if actualPackageHash != expectedSHA256 {
		return VerifiedPackage{}, errors.New("language package SHA-256 does not match stored artifact hash")
	}

	archive, err := zip.NewReader(bytes.NewReader(artifact), int64(len(artifact)))
	if err != nil {
		return VerifiedPackage{}, fmt.Errorf("open language package ZIP: %w", err)
	}
	if len(archive.File) != maxDeploymentPackageEntries {
		return VerifiedPackage{}, fmt.Errorf("language package must contain exactly %d entries", maxDeploymentPackageEntries)
	}

	files := make(map[string][]byte, len(archive.File))
	var totalUncompressed uint64
	for _, file := range archive.File {
		if !file.Mode().IsRegular() || file.Mode()&os.ModeSymlink != 0 {
			return VerifiedPackage{}, fmt.Errorf("language package entry %q must be a regular file", file.Name)
		}
		if !safePackagePath(file.Name) {
			return VerifiedPackage{}, fmt.Errorf("language package entry %q has an unsafe path", file.Name)
		}
		if _, duplicate := files[file.Name]; duplicate {
			return VerifiedPackage{}, fmt.Errorf("language package contains duplicate entry %q", file.Name)
		}
		if _, allowed := allowedPackageEntry(file.Name); !allowed {
			return VerifiedPackage{}, fmt.Errorf("language package contains unexpected entry %q", file.Name)
		}
		if file.UncompressedSize64 > maxDeploymentUncompressedBytes || totalUncompressed > maxDeploymentUncompressedBytes-file.UncompressedSize64 {
			return VerifiedPackage{}, errors.New("language package exceeds the uncompressed size limit")
		}
		totalUncompressed += file.UncompressedSize64
		limit, _ := allowedPackageEntry(file.Name)
		content, err := readPackageEntry(file, limit)
		if err != nil {
			return VerifiedPackage{}, err
		}
		files[file.Name] = content
	}

	manifestBytes, ok := files["manifest.json"]
	if !ok {
		return VerifiedPackage{}, errors.New("language package is missing manifest.json")
	}
	var wire packageManifestWire
	if err := decodeStrictPackageJSON(manifestBytes, &wire); err != nil {
		return VerifiedPackage{}, fmt.Errorf("decode language package manifest: %w", err)
	}
	if wire.OEMFirmware == nil {
		return VerifiedPackage{}, errors.New("language package manifest must declare oem_firmware=false")
	}
	manifest := PackageManifest{
		Schema: wire.Schema, PackageType: wire.PackageType, ProjectID: wire.ProjectID, ProjectName: wire.ProjectName,
		Locale: wire.Locale, ResourceFormat: wire.ResourceFormat, ResourcePath: wire.ResourcePath,
		SourceSHA256: wire.SourceSHA256, OEMFirmware: *wire.OEMFirmware,
	}
	if err := validatePackageManifest(manifest); err != nil {
		return VerifiedPackage{}, err
	}
	if _, ok := files[manifest.ResourcePath]; !ok {
		return VerifiedPackage{}, errors.New("language package resource path does not match an archive entry")
	}
	resourceEntries := 0
	for name := range files {
		if strings.HasPrefix(name, "resources/") {
			resourceEntries++
			if name != manifest.ResourcePath {
				return VerifiedPackage{}, errors.New("language package contains an undeclared resource entry")
			}
		}
	}
	if resourceEntries != 1 {
		return VerifiedPackage{}, errors.New("language package must contain exactly one declared resource entry")
	}

	validationBytes, ok := files["validation.json"]
	if !ok {
		return VerifiedPackage{}, errors.New("language package is missing validation.json")
	}
	var validation domain.LocalizationValidationReport
	if err := decodeStrictPackageJSON(validationBytes, &validation); err != nil {
		return VerifiedPackage{}, fmt.Errorf("decode language package validation report: %w", err)
	}
	if !validation.Valid {
		return VerifiedPackage{}, errors.New("language package validation report is not valid")
	}

	resourceContent := files[manifest.ResourcePath]
	if _, err := localization.Parse(domain.LocalizationFormat(manifest.ResourceFormat), path.Base(manifest.ResourcePath), resourceContent); err != nil {
		return VerifiedPackage{}, fmt.Errorf("parse packaged localization resource: %w", err)
	}
	checksums, ok := files["checksums.sha256"]
	if !ok {
		return VerifiedPackage{}, errors.New("language package is missing checksums.sha256")
	}
	if err := verifyPackageChecksums(checksums, files); err != nil {
		return VerifiedPackage{}, err
	}
	return VerifiedPackage{SHA256: actualPackageHash, Manifest: manifest}, nil
}

func validatePackageManifest(manifest PackageManifest) error {
	if manifest.Schema != "carvion.language-package/v1" || manifest.PackageType != "carvion-localization" {
		return errors.New("language package manifest schema or package_type is not supported")
	}
	if manifest.OEMFirmware {
		return errors.New("OEM firmware packages are not accepted by CARVION Deployment")
	}
	if strings.TrimSpace(manifest.ProjectID) == "" || strings.TrimSpace(manifest.ProjectName) == "" || strings.TrimSpace(manifest.Locale) == "" {
		return errors.New("language package manifest is missing project or locale identity")
	}
	if manifest.ResourceFormat != string(domain.LocalizationFormatAndroidXML) && manifest.ResourceFormat != string(domain.LocalizationFormatJSON) {
		return fmt.Errorf("language package resource format %q is not supported", manifest.ResourceFormat)
	}
	if !isSHA256(manifest.SourceSHA256) {
		return errors.New("language package source_sha256 is invalid")
	}
	if !safePackagePath(manifest.ResourcePath) || path.Dir(manifest.ResourcePath) != "resources" || path.Base(manifest.ResourcePath) == "." {
		return errors.New("language package resource_path is unsafe")
	}
	return nil
}

func allowedPackageEntry(name string) (int64, bool) {
	switch name {
	case "manifest.json":
		return maxDeploymentManifestBytes, true
	case "validation.json":
		return maxDeploymentValidationBytes, true
	case "checksums.sha256":
		return maxDeploymentChecksumsBytes, true
	default:
		if strings.HasPrefix(name, "resources/") && path.Dir(name) == "resources" && path.Base(name) != "." && path.Base(name) != "" {
			return maxDeploymentResourceBytes, true
		}
		return 0, false
	}
}

func safePackagePath(name string) bool {
	if name == "" || strings.Contains(name, `\`) || strings.ContainsRune(name, 0) || path.IsAbs(name) || path.Clean(name) != name {
		return false
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func readPackageEntry(file *zip.File, limit int64) ([]byte, error) {
	if limit < 0 || file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("language package entry %q exceeds its size limit", file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open language package entry %q: %w", file.Name, err)
	}
	content, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read language package entry %q: %w", file.Name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close language package entry %q: %w", file.Name, closeErr)
	}
	if int64(len(content)) > limit || uint64(len(content)) != file.UncompressedSize64 {
		return nil, fmt.Errorf("language package entry %q has an invalid uncompressed size", file.Name)
	}
	return content, nil
}

func verifyPackageChecksums(checksumFile []byte, files map[string][]byte) error {
	if !utf8.Valid(checksumFile) || len(checksumFile) == 0 || checksumFile[len(checksumFile)-1] != '\n' {
		return errors.New("language package checksum manifest is not valid UTF-8 or lacks a final newline")
	}
	lines := strings.Split(strings.TrimSuffix(string(checksumFile), "\n"), "\n")
	if len(lines) != len(files)-1 {
		return errors.New("language package checksum manifest does not list every package entry")
	}
	seen := make(map[string]struct{}, len(lines))
	previousName := ""
	for _, line := range lines {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || !isSHA256(parts[0]) || parts[1] == "checksums.sha256" {
			return errors.New("language package checksum manifest contains an invalid line")
		}
		name := parts[1]
		if previousName != "" && name <= previousName {
			return errors.New("language package checksum entries must be unique and sorted")
		}
		previousName = name
		content, ok := files[name]
		if !ok {
			return fmt.Errorf("language package checksum references unknown entry %q", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("language package checksum repeats entry %q", name)
		}
		seen[name] = struct{}{}
		if packageSHA256(content) != parts[0] {
			return fmt.Errorf("language package entry %q failed its SHA-256 check", name)
		}
	}
	return nil
}

func decodeStrictPackageJSON(data []byte, target any) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON contains multiple values")
		}
		return fmt.Errorf("decode trailing JSON data: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var readValue func() error
	readValue = func() error {
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
				if _, duplicate := seen[key]; duplicate {
					return fmt.Errorf("JSON object contains duplicate key %q", key)
				}
				seen[key] = struct{}{}
				if err := readValue(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := readValue(); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
		}
	}
	if err := readValue(); err != nil {
		return fmt.Errorf("validate JSON object: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("JSON contains multiple values")
		}
		return fmt.Errorf("decode trailing JSON data: %w", err)
	}
	return nil
}

func isSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func packageSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
