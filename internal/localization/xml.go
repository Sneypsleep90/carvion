package localization

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/Sneypsleep90/carvion/internal/core/domain"
)

type xmlContainer struct {
	name         string
	key          string
	translatable bool
	formatted    bool
	metadata     json.RawMessage
	index        int
	categories   map[string]struct{}
}

type xmlValue struct {
	key          string
	groupKey     string
	kind         domain.LocalizationResourceKind
	category     string
	index        int
	translatable bool
	formatted    bool
	metadata     json.RawMessage
	innerStart   int
	innerEnd     int
	selfClosing  bool
	elementName  string
	text         strings.Builder
	nested       bool
}

func parseAndroidXML(doc Document) (Document, error) {
	decoder := xml.NewDecoder(bytes.NewReader(doc.Source))
	decoder.Strict = true
	doc.spans = make(map[string]sourceSpan)
	var container *xmlContainer
	var value *xmlValue
	skipDepth := 0
	seen := make(map[string]struct{})
	rootSeen, rootClosed := false, false
	for {
		before := int(decoder.InputOffset())
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Document{}, fmt.Errorf("parse Android XML: %w", err)
		}
		after := int(decoder.InputOffset())
		switch item := token.(type) {
		case xml.StartElement:
			if skipDepth > 0 {
				skipDepth++
				continue
			}
			rawStart := string(doc.Source[before:after])
			if !rootSeen {
				if item.Name.Local != "resources" {
					return Document{}, errors.New("Android XML root element must be <resources>")
				}
				if item.Name.Space != "" {
					return Document{}, errors.New("namespaced Android XML root element is not supported")
				}
				rootSeen = true
				continue
			}
			if rootClosed {
				return Document{}, errors.New("Android XML contains content after </resources>")
			}
			if item.Name.Space != "" && isAndroidResourceTag(item.Name.Local) {
				return Document{}, fmt.Errorf("namespaced Android resource element <%s> is not supported", item.Name.Local)
			}
			if value != nil {
				value.nested = true
				continue
			}
			switch item.Name.Local {
			case "string":
				if container != nil {
					return Document{}, errors.New("nested string resource in Android XML")
				}
				attrs, err := xmlAttributes(item.Attr)
				if err != nil {
					return Document{}, err
				}
				key := attrs["name"]
				if key == "" {
					return Document{}, errors.New("Android string resource requires a name")
				}
				translatable, err := booleanAttr(attrs, "translatable", true)
				if err != nil {
					return Document{}, err
				}
				formatted, err := booleanAttr(attrs, "formatted", true)
				if err != nil {
					return Document{}, err
				}
				metadata, _ := json.Marshal(map[string]any{"attributes": attrs, "rawStartTag": rawStart})
				value = &xmlValue{key: key, kind: domain.ResourceKindString, translatable: translatable, formatted: formatted, metadata: metadata, innerStart: after, selfClosing: isSelfClosingTag(doc.Source[before:after]), elementName: item.Name.Local}
			case "string-array", "plurals":
				attrs, err := xmlAttributes(item.Attr)
				if err != nil {
					return Document{}, err
				}
				key := attrs["name"]
				if key == "" {
					return Document{}, fmt.Errorf("Android %s resource requires a name", item.Name.Local)
				}
				translatable, err := booleanAttr(attrs, "translatable", true)
				if err != nil {
					return Document{}, err
				}
				formatted, err := booleanAttr(attrs, "formatted", true)
				if err != nil {
					return Document{}, err
				}
				metadata, _ := json.Marshal(map[string]any{"attributes": attrs, "rawStartTag": rawStart})
				container = &xmlContainer{name: item.Name.Local, key: key, translatable: translatable, formatted: formatted, metadata: metadata, categories: make(map[string]struct{})}
			case "item":
				if container == nil {
					continue
				}
				attrs, err := xmlAttributes(item.Attr)
				if err != nil {
					return Document{}, err
				}
				kind := domain.ResourceKindStringArrayItem
				key := fmt.Sprintf("%s[%d]", container.key, container.index)
				category := ""
				if container.name == "plurals" {
					kind = domain.ResourceKindPluralItem
					category = attrs["quantity"]
					if !validPluralCategory(category) {
						return Document{}, fmt.Errorf("invalid plural category %q for %q", category, container.key)
					}
					if _, exists := container.categories[category]; exists {
						return Document{}, fmt.Errorf("duplicate plural category %q for %q", category, container.key)
					}
					container.categories[category] = struct{}{}
					key = container.key + "#" + category
				}
				metadata, _ := json.Marshal(map[string]any{"container": json.RawMessage(container.metadata), "attributes": attrs, "rawStartTag": rawStart})
				value = &xmlValue{key: key, groupKey: container.key, kind: kind, category: category, index: container.index, translatable: container.translatable, formatted: container.formatted, metadata: metadata, innerStart: after, selfClosing: isSelfClosingTag(doc.Source[before:after]), elementName: item.Name.Local}
			default:
				// Keep non-string Android resources in the untouched source snapshot.
				// They are outside this parser's translation scope.
				skipDepth = 1
			}
		case xml.CharData:
			if value != nil {
				value.text.Write([]byte(item))
			}
		case xml.Comment:
			if value != nil && value.translatable {
				return Document{}, fmt.Errorf("XML comments inside translatable resource %q are not supported", value.key)
			}
		case xml.ProcInst:
			if value != nil && value.translatable {
				return Document{}, fmt.Errorf("XML processing instructions inside translatable resource %q are not supported", value.key)
			}
		case xml.Directive:
			if value != nil && value.translatable {
				return Document{}, fmt.Errorf("XML directives inside translatable resource %q are not supported", value.key)
			}
		case xml.EndElement:
			if skipDepth > 0 {
				skipDepth--
				continue
			}
			if value != nil {
				isValueEnd := (value.kind == domain.ResourceKindString && item.Name.Local == "string") || (value.kind != domain.ResourceKindString && item.Name.Local == "item")
				if isValueEnd {
					value.innerEnd = before
					if value.translatable && (value.nested || bytes.Contains(doc.Source[value.innerStart:value.innerEnd], []byte("<![CDATA["))) {
						return Document{}, fmt.Errorf("nested markup is not supported in resource %q", value.key)
					}
					resource := makeResource(value)
					resource.Order = len(doc.Resources)
					if _, exists := seen[resource.Key]; exists {
						return Document{}, fmt.Errorf("duplicate resource key %q", resource.Key)
					}
					seen[resource.Key] = struct{}{}
					span := sourceSpan{start: value.innerStart, end: value.innerEnd, elementName: value.elementName}
					if value.selfClosing {
						span.start = before - 2
						span.end = before
						span.selfClosing = true
						if span.start < 0 || string(doc.Source[span.start:span.end]) != "/>" {
							return Document{}, fmt.Errorf("invalid self-closing XML span for resource %q", value.key)
						}
					}
					doc.spans[resource.Key] = span
					doc.Resources = append(doc.Resources, resource)
					value = nil
					if container != nil {
						container.index++
					}
				}
				continue
			}
			if container != nil && item.Name.Local == container.name {
				container = nil
			}
			if item.Name.Local == "resources" {
				rootClosed = true
			}
		}
	}
	if !rootSeen || !rootClosed {
		return Document{}, errors.New("malformed Android XML: missing <resources> root")
	}
	return doc, nil
}

func isAndroidResourceTag(name string) bool {
	switch name {
	case "string", "string-array", "plurals", "item":
		return true
	default:
		return false
	}
}

func isSelfClosingTag(raw []byte) bool {
	return bytes.HasSuffix(bytes.TrimSpace(raw), []byte("/>"))
}

func makeResource(value *xmlValue) domain.LocalizationResource {
	text := value.text.String()
	return domain.LocalizationResource{
		Key: value.key, GroupKey: value.groupKey, Kind: value.kind, SourceText: text,
		SourceSHA256: sourceHash(text), Order: 0, ItemIndex: value.index,
		PluralCategory: value.category, Translatable: value.translatable,
		Formatted: value.formatted, Present: true, Metadata: value.metadata,
	}
}

func xmlAttributes(attrs []xml.Attr) (map[string]string, error) {
	result := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		key := attr.Name.Local
		if attr.Name.Space != "" {
			key = "{" + attr.Name.Space + "}" + key
		}
		result[key] = attr.Value
	}
	return result, nil
}

func booleanAttr(attrs map[string]string, name string, fallback bool) (bool, error) {
	value, exists := attrs[name]
	if !exists {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("Android attribute %s must be true or false", name)
	}
	return parsed, nil
}

func exportAndroidXML(doc Document, translations map[string]domain.LocalizationTranslation) ([]byte, error) {
	if len(doc.spans) == 0 && len(doc.Resources) > 0 {
		parsed, err := Parse(FormatAndroidXML, doc.Filename, doc.Source)
		if err != nil {
			return nil, err
		}
		doc.spans = parsed.spans
	}
	type replacement struct {
		start, end int
		text       []byte
	}
	replacements := make([]replacement, 0)
	for _, resource := range doc.Resources {
		translation, ok := translations[resource.Key]
		span, hasSpan := doc.spans[resource.Key]
		if !ok || !hasSpan || !resource.Translatable {
			continue
		}
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(translation.TargetText)); err != nil {
			return nil, fmt.Errorf("escape translation %q: %w", resource.Key, err)
		}
		text := escaped.Bytes()
		if span.selfClosing {
			text = append([]byte{'>'}, text...)
			text = append(text, []byte("</"+span.elementName+">")...)
		}
		replacements = append(replacements, replacement{span.start, span.end, text})
	}
	sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
	output := append([]byte(nil), doc.Source...)
	for _, replacement := range replacements {
		if replacement.start < 0 || replacement.end < replacement.start || replacement.end > len(output) {
			return nil, errors.New("invalid Android XML source span")
		}
		updated := make([]byte, 0, len(output)-(replacement.end-replacement.start)+len(replacement.text))
		updated = append(updated, output[:replacement.start]...)
		updated = append(updated, replacement.text...)
		updated = append(updated, output[replacement.end:]...)
		output = updated
	}
	return output, nil
}
