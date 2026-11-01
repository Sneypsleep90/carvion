package localization

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func Parse(format domain.LocalizationFormat, filename string, content []byte) (Document, error) {
	if !utf8.Valid(content) {
		return Document{}, errors.New("resource document is not valid UTF-8")
	}
	doc := Document{Format: format, Filename: filename, Source: append([]byte(nil), content...)}
	switch format {
	case FormatAndroidXML:
		return parseAndroidXML(doc)
	case FormatJSON:
		return parseJSON(doc)
	default:
		return Document{}, fmt.Errorf("unsupported localization format %q", format)
	}
}

func Export(doc Document, translations []domain.LocalizationTranslation) ([]byte, error) {
	byKey := make(map[string]domain.LocalizationTranslation, len(translations))
	for _, translation := range translations {
		if !utf8.ValidString(translation.TargetText) {
			return nil, fmt.Errorf("translation %q is not valid UTF-8", translation.ResourceKey)
		}
		byKey[translation.ResourceKey] = translation
	}
	switch doc.Format {
	case FormatAndroidXML:
		return exportAndroidXML(doc, byKey)
	case FormatJSON:
		return exportJSON(doc, byKey)
	default:
		return nil, fmt.Errorf("unsupported localization format %q", doc.Format)
	}
}

func ParseTranslationCSV(content []byte) ([]CSVTranslation, error) {
	if !utf8.Valid(content) {
		return nil, errors.New("translation CSV is not valid UTF-8")
	}
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = 4
	reader.TrimLeadingSpace = false
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	want := []string{"resource_key", "source_text", "target_text", "locale"}
	if len(header) != len(want) {
		return nil, errors.New("translation CSV header must be resource_key,source_text,target_text,locale")
	}
	for i := range want {
		if header[i] != want[i] {
			return nil, errors.New("translation CSV header must be resource_key,source_text,target_text,locale")
		}
	}
	rows := make([]CSVTranslation, 0)
	seen := make(map[string]struct{})
	for line := 2; ; line++ {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", line, readErr)
		}
		row := CSVTranslation{ResourceKey: record[0], SourceText: record[1], TargetText: record[2], Locale: record[3]}
		if strings.TrimSpace(row.ResourceKey) == "" || strings.TrimSpace(row.Locale) == "" {
			return nil, fmt.Errorf("CSV row %d requires resource_key and locale", line)
		}
		key := row.Locale + "\x00" + row.ResourceKey
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("duplicate translation key %q for locale %q", row.ResourceKey, row.Locale)
		}
		seen[key] = struct{}{}
		rows = append(rows, row)
	}
	return rows, nil
}

func ExportTranslationCSV(resources []domain.LocalizationResource, translations []domain.LocalizationTranslation, locale string) ([]byte, error) {
	byKey := make(map[string]domain.LocalizationTranslation, len(translations))
	for _, translation := range translations {
		if translation.Locale == locale {
			byKey[translation.ResourceKey] = translation
		}
	}
	ordered := append([]domain.LocalizationResource(nil), resources...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Order < ordered[j].Order })
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"resource_key", "source_text", "target_text", "locale"}); err != nil {
		return nil, err
	}
	for _, resource := range ordered {
		if !resource.Translatable || !resource.Present {
			continue
		}
		translation := byKey[resource.Key]
		if err := writer.Write([]string{resource.Key, resource.SourceText, translation.TargetText, locale}); err != nil {
			return nil, fmt.Errorf("write CSV row %q: %w", resource.Key, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush translation CSV: %w", err)
	}
	return buffer.Bytes(), nil
}

func sourceHash(value string) string {
	return sha256Hex([]byte(value))
}

func SHA256(content []byte) string { return sha256Hex(content) }

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum[:])
}
