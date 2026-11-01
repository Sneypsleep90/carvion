package domain

import (
	"encoding/json"
	"time"
)

type LocalizationFormat string

const (
	LocalizationFormatAndroidXML LocalizationFormat = "android-strings-xml"
	LocalizationFormatJSON       LocalizationFormat = "carvion-json-v1"
)

type LocalizationResourceKind string

const (
	ResourceKindString          LocalizationResourceKind = "string"
	ResourceKindStringArrayItem LocalizationResourceKind = "string-array-item"
	ResourceKindPluralItem      LocalizationResourceKind = "plural-item"
)

type TranslationStatus string

const (
	TranslationStatusDraft       TranslationStatus = "draft"
	TranslationStatusApproved    TranslationStatus = "approved"
	TranslationStatusNeedsReview TranslationStatus = "needs_review"
)

type LocalizationResource struct {
	Key            string                   `json:"key"`
	GroupKey       string                   `json:"groupKey,omitempty"`
	Kind           LocalizationResourceKind `json:"kind"`
	SourceText     string                   `json:"sourceText"`
	SourceSHA256   string                   `json:"sourceSha256"`
	Order          int                      `json:"order"`
	ItemIndex      int                      `json:"itemIndex,omitempty"`
	PluralCategory string                   `json:"pluralCategory,omitempty"`
	Translatable   bool                     `json:"translatable"`
	Formatted      bool                     `json:"formatted"`
	Present        bool                     `json:"present"`
	Metadata       json.RawMessage          `json:"metadata,omitempty"`
}

type LocalizationTranslation struct {
	ResourceKey          string            `json:"resourceKey"`
	Locale               string            `json:"locale"`
	TargetText           string            `json:"targetText"`
	Status               TranslationStatus `json:"status"`
	ApprovedSourceSHA256 string            `json:"approvedSourceSha256,omitempty"`
	UpdatedAt            time.Time         `json:"updatedAt,omitempty"`
}

type LocalizationValidationIssue struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	ResourceKey string `json:"resourceKey,omitempty"`
	Message     string `json:"message"`
}

type LocalizationValidationReport struct {
	Valid     bool                          `json:"valid"`
	Locale    string                        `json:"locale,omitempty"`
	CheckedAt time.Time                     `json:"checkedAt,omitempty"`
	Issues    []LocalizationValidationIssue `json:"issues"`
}

type LocalizationProject struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Locale       string                    `json:"locale"`
	Format       LocalizationFormat        `json:"format"`
	Filename     string                    `json:"filename"`
	SourceSHA256 string                    `json:"sourceSha256"`
	Source       []byte                    `json:"-"`
	Resources    []LocalizationResource    `json:"resources"`
	Translations []LocalizationTranslation `json:"translations"`
	CreatedAt    time.Time                 `json:"createdAt,omitempty"`
	UpdatedAt    time.Time                 `json:"updatedAt,omitempty"`
}

type LocalizationProjectSummary struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Locale       string             `json:"locale"`
	Format       LocalizationFormat `json:"format"`
	Filename     string             `json:"filename"`
	SourceSHA256 string             `json:"sourceSha256"`
	UpdatedAt    time.Time          `json:"updatedAt,omitempty"`
}

type LocalizationPackage struct {
	ID            string `json:"id,omitempty"`
	Filename      string `json:"filename"`
	SHA256        string `json:"sha256"`
	Size          int    `json:"size"`
	ContentBase64 string `json:"contentBase64"`
}

type LocalizationArtifact struct {
	Filename      string `json:"filename"`
	ContentBase64 string `json:"contentBase64"`
}
