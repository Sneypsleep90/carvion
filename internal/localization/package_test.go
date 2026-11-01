package localization

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestBuildPackageIsDeterministicAndDeclaresCARVIONProvenance(t *testing.T) {
	input := PackageInput{
		ProjectID: "project-1", ProjectName: "Synthetic UI", Locale: "ru-RU",
		Format: FormatAndroidXML, Filename: "strings.xml", SourceSHA256: "source-hash",
		ResourceContent: []byte(`<resources><string name="title">Главная</string></resources>`),
		Report:          domain.LocalizationValidationReport{Valid: true, CheckedAt: testTimestamp(), Issues: []domain.LocalizationValidationIssue{}},
	}
	first, firstHash, err := BuildPackage(input)
	if err != nil {
		t.Fatalf("BuildPackage() error = %v", err)
	}
	second, secondHash, err := BuildPackage(input)
	if err != nil {
		t.Fatalf("BuildPackage() second error = %v", err)
	}
	if !bytes.Equal(first, second) || firstHash != secondHash {
		t.Fatal("identical package inputs produced different ZIP output")
	}
	archive, err := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if err != nil {
		t.Fatalf("open package ZIP: %v", err)
	}
	files := make(map[string][]byte)
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = content
	}
	for _, name := range []string{"manifest.json", "validation.json", "checksums.sha256", "resources/strings.xml"} {
		if _, ok := files[name]; !ok {
			t.Errorf("package is missing %q", name)
		}
	}
	var manifest map[string]any
	if err := json.Unmarshal(files["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["package_type"] != "carvion-localization" || manifest["oem_firmware"] != false {
		t.Errorf("package manifest provenance fields = %#v", manifest)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(files["checksums.sha256"])), "\n") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 || hex.EncodeToString(sumBytes(files[parts[1]])) != parts[0] {
			t.Errorf("checksum line %q does not match file contents", line)
		}
	}
	if hex.EncodeToString(sumBytes(first)) != firstHash {
		t.Fatal("returned package SHA-256 does not match ZIP")
	}
}

func TestBuildPackageRejectsInvalidReport(t *testing.T) {
	_, _, err := BuildPackage(PackageInput{ProjectID: "p", ProjectName: "p", Locale: "ru-RU", Filename: "strings.xml", ResourceContent: []byte("x")})
	if err == nil {
		t.Fatal("BuildPackage() accepted an invalid validation report")
	}
}

func testTimestamp() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) }

func sumBytes(content []byte) []byte {
	sum := sha256.Sum256(content)
	return sum[:]
}
