package localization

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func parseJSON(doc Document) (Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(doc.Source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc.jsonDoc); err != nil {
		return Document{}, fmt.Errorf("parse localization JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Document{}, errors.New("localization JSON contains trailing data")
		}
		return Document{}, fmt.Errorf("parse localization JSON trailing data: %w", err)
	}
	if doc.jsonDoc.Schema != JSONSchema {
		return Document{}, fmt.Errorf("unsupported localization JSON schema %q", doc.jsonDoc.Schema)
	}
	if doc.jsonDoc.Resources == nil {
		return Document{}, errors.New("localization JSON resources array is required")
	}
	if len(doc.jsonDoc.Metadata) > 0 && !isJSONObject(doc.jsonDoc.Metadata) {
		return Document{}, errors.New("localization JSON metadata must be an object")
	}
	doc.Metadata = append([]byte(nil), doc.jsonDoc.Metadata...)
	seen := make(map[string]struct{}, len(doc.jsonDoc.Resources))
	pluralCategories := make(map[string]map[string]struct{})
	for i, item := range doc.jsonDoc.Resources {
		if strings.TrimSpace(item.Key) == "" {
			return Document{}, fmt.Errorf("localization JSON resource %d has an empty key", i)
		}
		if _, exists := seen[item.Key]; exists {
			return Document{}, fmt.Errorf("duplicate resource key %q", item.Key)
		}
		seen[item.Key] = struct{}{}
		if item.Kind == "" {
			item.Kind = domain.ResourceKindString
			doc.jsonDoc.Resources[i].Kind = item.Kind
		}
		if item.Kind != domain.ResourceKindString && item.Kind != domain.ResourceKindStringArrayItem && item.Kind != domain.ResourceKindPluralItem {
			return Document{}, fmt.Errorf("unknown localization resource kind %q", item.Kind)
		}
		if len(item.Metadata) > 0 && !isJSONObject(item.Metadata) {
			return Document{}, fmt.Errorf("metadata for resource %q must be an object", item.Key)
		}
		translatable, formatted := true, true
		if item.Translatable != nil {
			translatable = *item.Translatable
		}
		if item.Formatted != nil {
			formatted = *item.Formatted
		}
		groupKey := item.GroupKey
		category := item.PluralCategory
		if item.Kind == domain.ResourceKindPluralItem {
			if groupKey == "" || !validPluralCategory(category) {
				return Document{}, fmt.Errorf("plural resource %q requires groupKey and a valid pluralCategory", item.Key)
			}
			if pluralCategories[groupKey] == nil {
				pluralCategories[groupKey] = make(map[string]struct{})
			}
			if _, exists := pluralCategories[groupKey][category]; exists {
				return Document{}, fmt.Errorf("duplicate plural category %q for %q", category, groupKey)
			}
			pluralCategories[groupKey][category] = struct{}{}
		}
		doc.Resources = append(doc.Resources, domain.LocalizationResource{
			Key: item.Key, GroupKey: groupKey, Kind: item.Kind, SourceText: item.Text,
			SourceSHA256: sourceHash(item.Text), Order: i, ItemIndex: item.ItemIndex,
			PluralCategory: category, Translatable: translatable, Formatted: formatted,
			Present: true, Metadata: append([]byte(nil), item.Metadata...),
		})
	}
	return doc, nil
}

func exportJSON(doc Document, translations map[string]domain.LocalizationTranslation) ([]byte, error) {
	parsed := doc.jsonDoc
	if parsed.Schema == "" {
		if err := json.Unmarshal(doc.Source, &parsed); err != nil {
			return nil, fmt.Errorf("decode localization JSON for export: %w", err)
		}
	}
	for i := range parsed.Resources {
		resource := parsed.Resources[i]
		if translation, ok := translations[resource.Key]; ok && (resource.Translatable == nil || *resource.Translatable) {
			parsed.Resources[i].Text = translation.TargetText
		}
	}
	output, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode localization JSON: %w", err)
	}
	return append(output, '\n'), nil
}

func isJSONObject(value []byte) bool {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(trimmed, &object) == nil
}
