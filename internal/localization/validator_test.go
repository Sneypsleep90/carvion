package localization

import (
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestValidateReportsMissingEmptyAndChangedApprovedTranslations(t *testing.T) {
	resources := []domain.LocalizationResource{
		{Key: "title", Kind: domain.ResourceKindString, SourceText: "Hello", SourceSHA256: "source-v2", Translatable: true, Formatted: true, Present: true},
		{Key: "subtitle", Kind: domain.ResourceKindString, SourceText: "Subtitle", SourceSHA256: "subtitle-hash", Translatable: true, Formatted: true, Present: true},
		{Key: "private", Kind: domain.ResourceKindString, SourceText: "hidden", Translatable: false, Formatted: true, Present: true},
	}
	translations := []domain.LocalizationTranslation{
		{ResourceKey: "title", Locale: "ru-RU", TargetText: "   ", Status: domain.TranslationStatusDraft},
		{ResourceKey: "subtitle", Locale: "ru-RU", TargetText: "Подзаголовок", Status: domain.TranslationStatusApproved, ApprovedSourceSHA256: "subtitle-hash-old"},
	}

	report := Validate(resources, translations)
	if report.Valid {
		t.Fatal("Validate() valid = true, want invalid report")
	}
	wantCodes := []string{"empty_translation", "source_changed_after_approval"}
	for _, code := range wantCodes {
		if !hasIssue(report, code) {
			t.Errorf("validation report %+v is missing issue code %q", report.Issues, code)
		}
	}
	if hasIssue(report, "missing_translation") {
		t.Errorf("validation report %+v unexpectedly includes non-translatable resource", report.Issues)
	}
}

func TestValidateChecksMissingKeysDuplicateKeysPluralCategoriesAndUnicode(t *testing.T) {
	invalidUTF8 := string([]byte{0xff})
	resources := []domain.LocalizationResource{
		{Key: "duplicate", GroupKey: "", Kind: domain.ResourceKindString, SourceText: "same", SourceSHA256: "a", Translatable: true, Present: true},
		{Key: "duplicate", Kind: domain.ResourceKindString, SourceText: "same", SourceSHA256: "b", Translatable: true, Present: true},
		{Key: "count#other", GroupKey: "count", Kind: domain.ResourceKindPluralItem, PluralCategory: "other", SourceText: invalidUTF8, SourceSHA256: "c", Translatable: true, Present: true},
		{Key: "count#invalid", GroupKey: "count", Kind: domain.ResourceKindPluralItem, PluralCategory: "invalid", SourceText: "item", SourceSHA256: "d", Translatable: true, Present: true},
		{Key: "missing", Kind: domain.ResourceKindString, SourceText: "Missing", SourceSHA256: "e", Translatable: true, Present: true},
	}
	translations := []domain.LocalizationTranslation{
		{ResourceKey: "duplicate", Locale: "ru-RU", TargetText: "Перевод", Status: domain.TranslationStatusDraft},
		{ResourceKey: "count#other", Locale: "ru-RU", TargetText: "элементы", Status: domain.TranslationStatusDraft},
		{ResourceKey: "count#invalid", Locale: "ru-RU", TargetText: "элемент", Status: domain.TranslationStatusDraft},
	}
	report := Validate(resources, translations)
	for _, code := range []string{"duplicate_resource_key", "invalid_plural_category", "invalid_utf8", "missing_translation"} {
		if !hasIssue(report, code) {
			t.Errorf("validation report %+v is missing %q", report.Issues, code)
		}
	}
}

func TestValidateComparesFormatArgumentsAndAllowsPositionalReordering(t *testing.T) {
	resources := []domain.LocalizationResource{
		{Key: "valid", Kind: domain.ResourceKindString, SourceText: "User %1$s has %2$d items", SourceSHA256: "v", Translatable: true, Formatted: true, Present: true},
		{Key: "invalid", Kind: domain.ResourceKindString, SourceText: "User %s has %d items", SourceSHA256: "i", Translatable: true, Formatted: true, Present: true},
	}
	translations := []domain.LocalizationTranslation{
		{ResourceKey: "valid", Locale: "ru-RU", TargetText: "Элементов: %2$d у пользователя %1$s", Status: domain.TranslationStatusDraft},
		{ResourceKey: "invalid", Locale: "ru-RU", TargetText: "Пользователь получил %d элементов %s", Status: domain.TranslationStatusDraft},
	}
	report := Validate(resources, translations)
	if hasIssueForKey(report, "valid", "format_arguments_mismatch") {
		t.Errorf("report %+v should allow reordering explicitly positional arguments", report.Issues)
	}
	if !hasIssueForKey(report, "invalid", "format_arguments_mismatch") {
		t.Errorf("report %+v does not detect reordered non-positional arguments", report.Issues)
	}
}

func TestValidateRequiresOtherPluralAndRejectsMalformedFormatToken(t *testing.T) {
	resources := []domain.LocalizationResource{
		{Key: "count#one", GroupKey: "count", Kind: domain.ResourceKindPluralItem, PluralCategory: "one", SourceText: "%s", SourceSHA256: "one", Translatable: true, Formatted: true, Present: true},
		{Key: "label", Kind: domain.ResourceKindString, SourceText: "Value %", SourceSHA256: "label", Translatable: true, Formatted: true, Present: true},
	}
	translations := []domain.LocalizationTranslation{
		{ResourceKey: "count#one", Locale: "ru-RU", TargetText: "%s значение", Status: domain.TranslationStatusDraft},
		{ResourceKey: "label", Locale: "ru-RU", TargetText: "Значение", Status: domain.TranslationStatusDraft},
	}
	report := Validate(resources, translations)
	if !hasIssue(report, "plural_other_required") || !hasIssue(report, "invalid_format_arguments") {
		t.Errorf("report = %+v, want missing plural other and malformed format issues", report.Issues)
	}
}

func TestValidateReportsDuplicatePluralCategories(t *testing.T) {
	resources := []domain.LocalizationResource{
		{Key: "count-one-a", GroupKey: "count", Kind: domain.ResourceKindPluralItem, PluralCategory: "one", SourceText: "one", Translatable: true, Present: true},
		{Key: "count-one-b", GroupKey: "count", Kind: domain.ResourceKindPluralItem, PluralCategory: "one", SourceText: "another one", Translatable: true, Present: true},
		{Key: "count-other", GroupKey: "count", Kind: domain.ResourceKindPluralItem, PluralCategory: "other", SourceText: "other", Translatable: true, Present: true},
	}
	report := Validate(resources, nil)
	if !hasIssue(report, "duplicate_plural_category") {
		t.Errorf("report %+v does not flag duplicate plural category", report.Issues)
	}
}

func hasIssue(report domain.LocalizationValidationReport, code string) bool {
	return hasIssueForKey(report, "", code)
}

func hasIssueForKey(report domain.LocalizationValidationReport, key, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code && (key == "" || issue.ResourceKey == key) {
			return true
		}
	}
	return false
}

func TestCSVExportUsesStableHeaderAndEscapesCommasAndNewlines(t *testing.T) {
	resources := []domain.LocalizationResource{
		{Key: "hello", Kind: domain.ResourceKindString, SourceText: "Hello,\nworld", Order: 0, Translatable: true, Present: true},
	}
	translations := []domain.LocalizationTranslation{
		{ResourceKey: "hello", Locale: "ru-RU", TargetText: "Привет,\nмир", Status: domain.TranslationStatusDraft},
	}
	data, err := ExportTranslationCSV(resources, translations, "ru-RU")
	if err != nil {
		t.Fatalf("ExportTranslationCSV() error = %v", err)
	}
	if !strings.HasPrefix(string(data), "resource_key,source_text,target_text,locale\n") {
		t.Fatalf("CSV = %q, want stable UTF-8 header", data)
	}
	rows, err := ParseTranslationCSV(data)
	if err != nil || len(rows) != 1 || rows[0].SourceText != "Hello,\nworld" || rows[0].TargetText != "Привет,\nмир" {
		t.Fatalf("CSV round trip = %+v, %v", rows, err)
	}
}
