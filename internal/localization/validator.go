package localization

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func Validate(resources []domain.LocalizationResource, translations []domain.LocalizationTranslation) domain.LocalizationValidationReport {
	issues := make([]domain.LocalizationValidationIssue, 0)
	add := func(code, key, message string) {
		issues = append(issues, domain.LocalizationValidationIssue{Code: code, Severity: "error", ResourceKey: key, Message: message})
	}
	resourceCounts := make(map[string]int, len(resources))
	pluralGroups := make(map[string]map[string]struct{})
	pluralCategoryCounts := make(map[string]map[string]int)
	for _, resource := range resources {
		resourceCounts[resource.Key]++
		if !utf8.ValidString(resource.SourceText) {
			add("invalid_utf8", resource.Key, "Исходный текст не является корректным UTF-8.")
		}
		if resource.Kind == domain.ResourceKindPluralItem {
			if !validPluralCategory(resource.PluralCategory) {
				add("invalid_plural_category", resource.Key, fmt.Sprintf("Недопустимая plural category %q.", resource.PluralCategory))
			}
			if pluralGroups[resource.GroupKey] == nil {
				pluralGroups[resource.GroupKey] = make(map[string]struct{})
			}
			pluralGroups[resource.GroupKey][resource.PluralCategory] = struct{}{}
			if pluralCategoryCounts[resource.GroupKey] == nil {
				pluralCategoryCounts[resource.GroupKey] = make(map[string]int)
			}
			pluralCategoryCounts[resource.GroupKey][resource.PluralCategory]++
		}
	}
	for key, count := range resourceCounts {
		if count > 1 {
			add("duplicate_resource_key", key, "Ключ ресурса встречается несколько раз.")
		}
	}
	for group, categories := range pluralGroups {
		if _, ok := categories["other"]; !ok {
			add("plural_other_required", group, "Plural group должен содержать категорию other.")
		}
		for category, count := range pluralCategoryCounts[group] {
			if count > 1 {
				add("duplicate_plural_category", group, fmt.Sprintf("Plural category %q встречается несколько раз.", category))
			}
		}
	}

	locales := make(map[string]struct{})
	translationsByPair := make(map[string]domain.LocalizationTranslation, len(translations))
	for _, translation := range translations {
		if strings.TrimSpace(translation.Locale) == "" {
			add("invalid_locale", translation.ResourceKey, "Для перевода не указан locale.")
			continue
		}
		locales[translation.Locale] = struct{}{}
		pair := translation.Locale + "\x00" + translation.ResourceKey
		if _, exists := translationsByPair[pair]; exists {
			add("duplicate_translation", translation.ResourceKey, "Для ресурса и locale сохранено несколько переводов.")
			continue
		}
		translationsByPair[pair] = translation
		if _, exists := resourceCounts[translation.ResourceKey]; !exists {
			add("unknown_translation_key", translation.ResourceKey, "Перевод ссылается на отсутствующий ресурс.")
		}
	}
	if len(locales) == 0 {
		locales["ru-RU"] = struct{}{}
	}
	localeNames := make([]string, 0, len(locales))
	for locale := range locales {
		localeNames = append(localeNames, locale)
	}
	// Sorting makes the issue order stable for reports and reproducible package builds.
	sortStrings(localeNames)
	for _, locale := range localeNames {
		for _, resource := range resources {
			if !resource.Translatable || !resource.Present {
				continue
			}
			translation, exists := translationsByPair[locale+"\x00"+resource.Key]
			if !exists {
				add("missing_translation", resource.Key, "Для ресурса отсутствует перевод.")
				continue
			}
			if !utf8.ValidString(translation.TargetText) {
				add("invalid_utf8", resource.Key, "Перевод не является корректным UTF-8.")
			}
			if strings.TrimSpace(translation.TargetText) == "" {
				add("empty_translation", resource.Key, "Перевод не может быть пустым.")
			}
			if translation.Status == domain.TranslationStatusNeedsReview || (translation.Status == domain.TranslationStatusApproved && translation.ApprovedSourceSHA256 != resource.SourceSHA256) {
				add("source_changed_after_approval", resource.Key, "Исходная строка изменилась после утверждения перевода; проверьте его повторно.")
			}
			if resource.Formatted {
				sourceArgs, sourceErr := formatArguments(resource.SourceText)
				targetArgs, targetErr := formatArguments(translation.TargetText)
				if sourceErr != nil || targetErr != nil {
					add("invalid_format_arguments", resource.Key, "Строка содержит некорректный форматный аргумент.")
				} else if !sameArguments(sourceArgs, targetArgs) {
					add("format_arguments_mismatch", resource.Key, "Набор форматных аргументов перевода не совпадает с оригиналом.")
				}
			}
		}
	}
	valid := true
	for _, issue := range issues {
		if issue.Severity == "error" {
			valid = false
			break
		}
	}
	return domain.LocalizationValidationReport{Valid: valid, Locale: onlyLocale(localeNames), Issues: issues}
}

func validPluralCategory(category string) bool {
	switch category {
	case "zero", "one", "two", "few", "many", "other":
		return true
	default:
		return false
	}
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func onlyLocale(values []string) string {
	if len(values) == 1 {
		return values[0]
	}
	return ""
}

type formatArg struct {
	index    int
	typeCode byte
}

func formatArguments(value string) ([]formatArg, error) {
	args := make([]formatArg, 0)
	nextIndex := 1
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		start := i
		i++
		if i >= len(value) {
			return nil, fmt.Errorf("dangling format marker at byte %d", start)
		}
		if value[i] == '%' || value[i] == 'n' {
			continue
		}
		index := 0
		digitStart := i
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i < len(value) && i > digitStart && value[i] == '$' {
			fmt.Sscanf(value[digitStart:i], "%d", &index)
			i++
		} else {
			i = digitStart
		}
		for i < len(value) && strings.ContainsRune("-#+ 0,(<", rune(value[i])) {
			i++
		}
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i < len(value) && value[i] == '.' {
			i++
			precisionStart := i
			for i < len(value) && value[i] >= '0' && value[i] <= '9' {
				i++
			}
			if precisionStart == i {
				return nil, fmt.Errorf("invalid format precision at byte %d", start)
			}
		}
		if i < len(value) && (value[i] == 't' || value[i] == 'T') {
			i++
		}
		if i >= len(value) || !strings.ContainsRune("bBhHsScCdoxXeEfgGaAtT", rune(value[i])) {
			return nil, fmt.Errorf("invalid format conversion at byte %d", start)
		}
		if index == 0 {
			index = nextIndex
			nextIndex++
		}
		args = append(args, formatArg{index: index, typeCode: value[i]})
	}
	return args, nil
}

func sameArguments(left, right []formatArg) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[formatArg]int, len(left))
	for _, arg := range left {
		counts[arg]++
	}
	for _, arg := range right {
		if counts[arg] == 0 {
			return false
		}
		counts[arg]--
	}
	return true
}
