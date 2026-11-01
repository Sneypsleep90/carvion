package localization

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

func TestParseAndroidXMLReadsStringsArraysPluralsAndMetadata(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="utf-8"?>
<resources xmlns:tools="urn:tools">
    <string name="welcome" formatted="false">Hello, %1$s</string>
    <string name="internal" translatable="false">route</string>
    <string-array name="modes" tools:keep="true"><item>Eco</item><item>Snow &amp; ice</item></string-array>
    <plurals name="items"><item quantity="one">%d item</item><item quantity="other">%d items</item></plurals>
</resources>`)

	doc, err := Parse(FormatAndroidXML, "strings.xml", input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if doc.Format != FormatAndroidXML || doc.Filename != "strings.xml" || !bytes.Equal(doc.Source, input) {
		t.Fatalf("document metadata/source = (%q, %q, %q), want source preserved", doc.Format, doc.Filename, doc.Source)
	}
	want := map[string]struct {
		text         string
		kind         domain.LocalizationResourceKind
		translatable bool
		formatted    bool
	}{
		"welcome":     {text: "Hello, %1$s", kind: domain.ResourceKindString, translatable: true, formatted: false},
		"internal":    {text: "route", kind: domain.ResourceKindString, translatable: false, formatted: true},
		"modes[0]":    {text: "Eco", kind: domain.ResourceKindStringArrayItem, translatable: true, formatted: true},
		"modes[1]":    {text: "Snow & ice", kind: domain.ResourceKindStringArrayItem, translatable: true, formatted: true},
		"items#one":   {text: "%d item", kind: domain.ResourceKindPluralItem, translatable: true, formatted: true},
		"items#other": {text: "%d items", kind: domain.ResourceKindPluralItem, translatable: true, formatted: true},
	}
	if len(doc.Resources) != len(want) {
		t.Fatalf("resources count = %d, want %d: %+v", len(doc.Resources), len(want), doc.Resources)
	}
	for _, resource := range doc.Resources {
		expect, ok := want[resource.Key]
		if !ok {
			t.Errorf("unexpected resource key %q", resource.Key)
			continue
		}
		if resource.SourceText != expect.text || resource.Kind != expect.kind || resource.Translatable != expect.translatable || resource.Formatted != expect.formatted {
			t.Errorf("resource %q = %+v, want text=%q kind=%q translatable=%t formatted=%t", resource.Key, resource, expect.text, expect.kind, expect.translatable, expect.formatted)
		}
		if resource.Key == "modes[1]" && !strings.Contains(string(resource.Metadata), "tools:keep") {
			t.Errorf("array item metadata = %s, want container metadata preserved", resource.Metadata)
		}
	}
}

func TestParseAndroidXMLRejectsInvalidUTF8MalformedXMLAndDuplicateKeys(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		wantErr string
	}{
		{name: "invalid UTF-8", content: []byte{'<', 'r', '>', 0xff, '<', '/', 'r', '>'}, wantErr: "UTF-8"},
		{name: "malformed XML", content: []byte(`<resources><string name="x">broken</resources>`), wantErr: "XML"},
		{name: "duplicate resource names", content: []byte(`<resources><string name="x">one</string><string name="x">two</string></resources>`), wantErr: "duplicate"},
		{name: "invalid translatable attribute", content: []byte(`<resources><string name="x" translatable="sometimes">one</string></resources>`), wantErr: "translatable"},
		{name: "namespaced resources root", content: []byte(`<p:resources xmlns:p="urn:test"><string name="x">one</string></p:resources>`), wantErr: "namespaced"},
		{name: "namespaced resource element", content: []byte(`<resources xmlns:p="urn:test"><p:string name="x"/></resources>`), wantErr: "namespaced"},
		{name: "inline comment", content: []byte(`<resources><string name="x">before<!-- note -->after</string></resources>`), wantErr: "comment"},
		{name: "inline processing instruction", content: []byte(`<resources><string name="x">before<?keep value?>after</string></resources>`), wantErr: "processing instruction"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(FormatAndroidXML, "strings.xml", test.content)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Parse() error = %v, want error containing %q", err, test.wantErr)
			}
		})
	}
}

func TestExportAndroidXMLPreservesStructureAttributesAndNonTranslatableText(t *testing.T) {
	input := []byte("<resources><!-- retain -->\n  <string name=\"welcome\" tools:example=\"yes\">Hello &amp; welcome</string>\n  <string name=\"skip\" translatable=\"false\"><b>KEEP</b><!-- private comment --></string>\n</resources>")
	doc, err := Parse(FormatAndroidXML, "strings.xml", input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	output, err := Export(doc, []domain.LocalizationTranslation{{ResourceKey: "welcome", Locale: "ru-RU", TargetText: "Привет и добро пожаловать", Status: domain.TranslationStatusDraft}})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	text := string(output)
	for _, expected := range []string{"<!-- retain -->", `tools:example="yes"`, `name="welcome"`, "Привет и добро пожаловать", `<string name="skip" translatable="false"><b>KEEP</b><!-- private comment --></string>`} {
		if !strings.Contains(text, expected) {
			t.Errorf("exported XML %q does not contain %q", text, expected)
		}
	}
}

func TestExportAndroidXMLUpdatesArrayAndPluralItemsWithoutChangingStructure(t *testing.T) {
	input := []byte(`<resources><string-array name="modes" translate="true"><item>Eco</item><item>Snow &amp; ice</item></string-array><plurals name="items"><item quantity="one">%d item</item><item quantity="other">%d items</item></plurals></resources>`)
	doc, err := Parse(FormatAndroidXML, "strings.xml", input)
	if err != nil {
		t.Fatal(err)
	}
	output, err := Export(doc, []domain.LocalizationTranslation{
		{ResourceKey: "modes[1]", Locale: "ru-RU", TargetText: "Снег и лёд"},
		{ResourceKey: "items#one", Locale: "ru-RU", TargetText: "%d элемент"},
		{ResourceKey: "items#other", Locale: "ru-RU", TargetText: "%d элемента"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, expected := range []string{`<string-array name="modes" translate="true">`, `<item>Eco</item>`, `<item>Снег и лёд</item>`, `quantity="one">%d элемент`, `quantity="other">%d элемента`} {
		if !strings.Contains(text, expected) {
			t.Errorf("exported XML %q does not contain %q", text, expected)
		}
	}
}

func TestExportAndroidXMLExpandsSelfClosingTranslatableResources(t *testing.T) {
	input := []byte(`<resources><string name="empty"/><string-array name="options"><item/></string-array><plurals name="count"><item quantity="other"/></plurals></resources>`)
	doc, err := Parse(FormatAndroidXML, "strings.xml", input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	translations := []domain.LocalizationTranslation{
		{ResourceKey: "empty", Locale: "ru-RU", TargetText: "Пустая строка"},
		{ResourceKey: "options[0]", Locale: "ru-RU", TargetText: "Вариант"},
		{ResourceKey: "count#other", Locale: "ru-RU", TargetText: "%d элементов"},
	}
	output, err := Export(doc, translations)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	exported, err := Parse(FormatAndroidXML, "strings.xml", output)
	if err != nil {
		t.Fatalf("Export() produced malformed XML %q: %v", output, err)
	}
	got := make(map[string]string, len(exported.Resources))
	for _, resource := range exported.Resources {
		got[resource.Key] = resource.SourceText
	}
	for _, translation := range translations {
		if got[translation.ResourceKey] != translation.TargetText {
			t.Errorf("exported resource %q = %q, want %q; XML: %s", translation.ResourceKey, got[translation.ResourceKey], translation.TargetText, output)
		}
	}
}

func TestAndroidXMLPreservesUnknownResourceElementsWithoutTranslatingThem(t *testing.T) {
	input := []byte(`<resources><color name="accent">#123456</color><string name="title">Home</string></resources>`)
	doc, err := Parse(FormatAndroidXML, "strings.xml", input)
	if err != nil {
		t.Fatalf("Parse() rejected a non-localized resource element: %v", err)
	}
	if len(doc.Resources) != 1 || doc.Resources[0].Key != "title" {
		t.Fatalf("parsed resources = %+v, want only the translatable string entry", doc.Resources)
	}
	output, err := Export(doc, []domain.LocalizationTranslation{{ResourceKey: "title", Locale: "ru-RU", TargetText: "Главная"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `<color name="accent">#123456</color>`) || !strings.Contains(string(output), "Главная") {
		t.Errorf("exported XML = %s; want untouched color and translated string", output)
	}
}

func TestParseJSONRequiresVersionedSchemaAndPreservesMetadata(t *testing.T) {
	input := []byte(`{"schema":"carvion.localization/v1","metadata":{"source":"test"},"resources":[{"key":"home.title","kind":"string","text":"Home","metadata":{"comment":"screen title"}}]}`)
	doc, err := Parse(FormatJSON, "strings.json", input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(doc.Resources) != 1 || doc.Resources[0].Key != "home.title" || doc.Resources[0].SourceText != "Home" || !strings.Contains(string(doc.Resources[0].Metadata), "screen title") {
		t.Fatalf("parsed resources = %+v", doc.Resources)
	}
	if !strings.Contains(string(doc.Metadata), `"source":"test"`) {
		t.Fatalf("document metadata = %s", doc.Metadata)
	}

	exported, err := Export(doc, []domain.LocalizationTranslation{{ResourceKey: "home.title", Locale: "ru-RU", TargetText: "Главная", Status: domain.TranslationStatusDraft}})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if !bytes.Contains(exported, []byte(`"schema": "carvion.localization/v1"`)) || !bytes.Contains(exported, []byte(`"text": "Главная"`)) || !bytes.Contains(exported, []byte(`"comment": "screen title"`)) {
		t.Errorf("exported JSON = %s, want schema, translated text and metadata", exported)
	}
}

func TestParseJSONRejectsUnknownSchemaDuplicateKeysAndInvalidUTF8(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		wantErr string
	}{
		{name: "unknown schema", content: []byte(`{"schema":"other/v1","resources":[]}`), wantErr: "schema"},
		{name: "duplicate keys", content: []byte(`{"schema":"carvion.localization/v1","resources":[{"key":"x","kind":"string","text":"a"},{"key":"x","kind":"string","text":"b"}]}`), wantErr: "duplicate"},
		{name: "unknown field", content: []byte(`{"schema":"carvion.localization/v1","resources":[],"ignored":true}`), wantErr: "unknown field"},
		{name: "invalid UTF-8", content: []byte{'{', 0xff, '}'}, wantErr: "UTF-8"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse(FormatJSON, "strings.json", test.content); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Parse() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestTranslationCSVRequiresHeaderAndRejectsDuplicates(t *testing.T) {
	good := []byte("resource_key,source_text,target_text,locale\nhello,Hello,Привет,ru-RU\n")
	rows, err := ParseTranslationCSV(good)
	if err != nil {
		t.Fatalf("ParseTranslationCSV() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ResourceKey != "hello" || rows[0].SourceText != "Hello" || rows[0].TargetText != "Привет" || rows[0].Locale != "ru-RU" {
		t.Fatalf("CSV rows = %+v", rows)
	}

	for _, test := range []struct {
		name    string
		content []byte
		wantErr string
	}{
		{name: "wrong header", content: []byte("key,target\nx,y\n"), wantErr: "header"},
		{name: "duplicate key", content: []byte("resource_key,source_text,target_text,locale\nx,a,A,ru-RU\nx,b,B,ru-RU\n"), wantErr: "duplicate"},
		{name: "invalid UTF-8", content: []byte{'r', 'e', 's', 'o', 'u', 'r', 'c', 'e', '_', 'k', 'e', 'y', ',', 0xff}, wantErr: "UTF-8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseTranslationCSV(test.content); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("ParseTranslationCSV() error = %v, want %q", err, test.wantErr)
			}
		})
	}
}
