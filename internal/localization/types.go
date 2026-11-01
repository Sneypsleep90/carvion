package localization

import (
	"encoding/json"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

const (
	FormatAndroidXML = domain.LocalizationFormatAndroidXML
	FormatJSON       = domain.LocalizationFormatJSON
	JSONSchema       = "carvion.localization/v1"
)

type Document struct {
	Format    domain.LocalizationFormat
	Filename  string
	Source    []byte
	Metadata  json.RawMessage
	Resources []domain.LocalizationResource
	spans     map[string]sourceSpan
	jsonDoc   jsonDocument
}

type sourceSpan struct {
	start, end  int
	selfClosing bool
	elementName string
}

type jsonDocument struct {
	Schema    string          `json:"schema"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Resources []jsonResource  `json:"resources"`
}

type jsonResource struct {
	Key            string                          `json:"key"`
	Kind           domain.LocalizationResourceKind `json:"kind"`
	Text           string                          `json:"text"`
	GroupKey       string                          `json:"groupKey,omitempty"`
	ItemIndex      int                             `json:"itemIndex,omitempty"`
	PluralCategory string                          `json:"pluralCategory,omitempty"`
	Translatable   *bool                           `json:"translatable,omitempty"`
	Formatted      *bool                           `json:"formatted,omitempty"`
	Metadata       json.RawMessage                 `json:"metadata,omitempty"`
}

type CSVTranslation struct {
	ResourceKey string
	SourceText  string
	TargetText  string
	Locale      string
}
