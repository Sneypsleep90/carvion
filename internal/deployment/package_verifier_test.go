package deployment

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
	"github.com/Sneypsleep90/carvion/internal/localization"
)

func TestVerifyLanguagePackageAcceptsCARVIONPackage(t *testing.T) {
	artifact, hash := buildTestLanguagePackage(t)
	verified, err := VerifyLanguagePackage(artifact, hash)
	if err != nil {
		t.Fatalf("VerifyLanguagePackage() error = %v", err)
	}
	if verified.SHA256 != hash {
		t.Fatalf("verified hash = %q, want %q", verified.SHA256, hash)
	}
	if verified.Manifest.PackageType != "carvion-localization" || verified.Manifest.OEMFirmware {
		t.Fatalf("verified manifest = %+v, want non-OEM CARVION localization artifact", verified.Manifest)
	}
}

func TestVerifyLanguagePackageRejectsUnsafeOrInvalidArchives(t *testing.T) {
	artifact, hash := buildTestLanguagePackage(t)
	entries := readPackageEntries(t, artifact)
	manifest := findPackageEntry(t, entries, "manifest.json")
	var manifestFields map[string]any
	if err := json.Unmarshal(manifest.data, &manifestFields); err != nil {
		t.Fatal(err)
	}
	validation := findPackageEntry(t, entries, "validation.json")
	var report map[string]any
	if err := json.Unmarshal(validation.data, &report); err != nil {
		t.Fatal(err)
	}

	unsafeManifest := clonePackageEntries(entries)
	unsafeFields := cloneJSONMap(t, manifestFields)
	unsafeFields["resource_path"] = "resources/../outside.xml"
	replacePackageEntry(t, unsafeManifest, "manifest.json", marshalTestJSON(t, unsafeFields))

	unsafeArchivePath := clonePackageEntries(entries)
	for i := range unsafeArchivePath {
		if unsafeArchivePath[i].name == "resources/strings.xml" {
			unsafeArchivePath[i].name = "resources/../strings.xml"
		}
	}

	unsupportedSchema := clonePackageEntries(entries)
	unsupportedSchemaFields := cloneJSONMap(t, manifestFields)
	unsupportedSchemaFields["schema"] = "carvion.language-package/v999"
	replacePackageEntry(t, unsupportedSchema, "manifest.json", marshalTestJSON(t, unsupportedSchemaFields))

	missingOEMDeclaration := clonePackageEntries(entries)
	missingOEMFields := cloneJSONMap(t, manifestFields)
	delete(missingOEMFields, "oem_firmware")
	replacePackageEntry(t, missingOEMDeclaration, "manifest.json", marshalTestJSON(t, missingOEMFields))

	duplicateManifestKey := clonePackageEntries(entries)
	replacePackageEntry(t, duplicateManifestKey, "manifest.json", []byte(`{"schema":"carvion.language-package/v1","schema":"carvion.language-package/v1"}`))

	oemManifest := clonePackageEntries(entries)
	oemFields := cloneJSONMap(t, manifestFields)
	oemFields["oem_firmware"] = true
	replacePackageEntry(t, oemManifest, "manifest.json", marshalTestJSON(t, oemFields))

	invalidReport := clonePackageEntries(entries)
	invalidReportFields := cloneJSONMap(t, report)
	invalidReportFields["valid"] = false
	replacePackageEntry(t, invalidReport, "validation.json", marshalTestJSON(t, invalidReportFields))

	checksumMismatch := clonePackageEntries(entries)
	replacePackageEntry(t, checksumMismatch, "resources/strings.xml", []byte(`<resources><string name="title">tampered</string></resources>`))

	duplicateEntry := clonePackageEntries(entries)
	duplicateEntry = append(duplicateEntry, packageTestEntry{name: "manifest.json", data: append([]byte(nil), manifest.data...)})

	unknownEntry := clonePackageEntries(entries)
	unknownEntry = append(unknownEntry, packageTestEntry{name: "unexpected.bin", data: []byte("synthetic")})

	oversizedEntry := clonePackageEntries(entries)
	replacePackageEntry(t, oversizedEntry, "resources/strings.xml", bytes.Repeat([]byte{'x'}, maxDeploymentResourceBytes+1))

	tests := []struct {
		name    string
		content []byte
		hash    string
	}{
		{name: "external SHA-256 mismatch", content: append([]byte(nil), artifact...), hash: "00"},
		{name: "corrupt ZIP", content: []byte("not a zip archive"), hash: packageSHA256([]byte("not a zip archive"))},
		{name: "path traversal", content: writePackageEntries(t, unsafeManifest), hash: ""},
		{name: "archive path traversal", content: writePackageEntries(t, unsafeArchivePath), hash: ""},
		{name: "unsupported schema", content: writePackageEntries(t, unsupportedSchema), hash: ""},
		{name: "missing OEM declaration", content: writePackageEntries(t, missingOEMDeclaration), hash: ""},
		{name: "duplicate manifest key", content: writePackageEntries(t, duplicateManifestKey), hash: ""},
		{name: "OEM firmware marker", content: writePackageEntries(t, oemManifest), hash: ""},
		{name: "invalid validation report", content: writePackageEntries(t, invalidReport), hash: ""},
		{name: "entry checksum mismatch", content: writePackageEntries(t, checksumMismatch), hash: ""},
		{name: "duplicate entry", content: writePackageEntries(t, duplicateEntry), hash: ""},
		{name: "unknown entry", content: writePackageEntries(t, unknownEntry), hash: ""},
		{name: "oversized resource entry", content: writePackageEntries(t, oversizedEntry), hash: ""},
		{name: "oversized package", content: bytes.Repeat([]byte{'x'}, MaxDeploymentPackageBytes+1), hash: ""},
	}
	for i := range tests {
		if tests[i].hash == "" {
			tests[i].hash = packageSHA256(tests[i].content)
		}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := VerifyLanguagePackage(test.content, test.hash); err == nil {
				t.Fatal("VerifyLanguagePackage() accepted invalid deployment package")
			}
		})
	}
	if hash == "" {
		t.Fatal("test package builder returned an empty hash")
	}
}

type packageTestEntry struct {
	name string
	data []byte
}

func buildTestLanguagePackage(t *testing.T) ([]byte, string) {
	t.Helper()
	content, hash, err := localization.BuildPackage(localization.PackageInput{
		ProjectID: "synthetic-project", ProjectName: "Synthetic localization", Locale: "ru-RU",
		Format: domain.LocalizationFormatAndroidXML, Filename: "strings.xml", SourceSHA256: packageSHA256([]byte("source")),
		ResourceContent: []byte(`<resources><string name="title">Синтетическая строка</string></resources>`),
		Report:          domain.LocalizationValidationReport{Valid: true, Issues: []domain.LocalizationValidationIssue{}},
	})
	if err != nil {
		t.Fatalf("BuildPackage() error = %v", err)
	}
	return content, hash
}

func readPackageEntries(t *testing.T, artifact []byte) []packageTestEntry {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(artifact), int64(len(artifact)))
	if err != nil {
		t.Fatalf("open package fixture: %v", err)
	}
	entries := make([]packageTestEntry, 0, len(archive.File))
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open fixture entry %q: %v", file.Name, err)
		}
		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			t.Fatalf("read fixture entry %q: %v", file.Name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close fixture entry %q: %v", file.Name, closeErr)
		}
		entries = append(entries, packageTestEntry{name: file.Name, data: data})
	}
	return entries
}

func writePackageEntries(t *testing.T, entries []packageTestEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, item := range entries {
		entry, err := writer.Create(item.name)
		if err != nil {
			t.Fatalf("create package fixture entry %q: %v", item.name, err)
		}
		if _, err := entry.Write(item.data); err != nil {
			t.Fatalf("write package fixture entry %q: %v", item.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close package fixture ZIP: %v", err)
	}
	return output.Bytes()
}

func findPackageEntry(t *testing.T, entries []packageTestEntry, name string) packageTestEntry {
	t.Helper()
	for _, item := range entries {
		if item.name == name {
			return item
		}
	}
	t.Fatalf("package fixture has no %q entry", name)
	return packageTestEntry{}
}

func replacePackageEntry(t *testing.T, entries []packageTestEntry, name string, content []byte) {
	t.Helper()
	for i := range entries {
		if entries[i].name == name {
			entries[i].data = content
			return
		}
	}
	t.Fatalf("package fixture has no %q entry to replace", name)
}

func clonePackageEntries(entries []packageTestEntry) []packageTestEntry {
	cloned := make([]packageTestEntry, len(entries))
	for i, item := range entries {
		cloned[i] = packageTestEntry{name: item.name, data: append([]byte(nil), item.data...)}
	}
	return cloned
}

func cloneJSONMap(t *testing.T, values map[string]any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func marshalTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
