package catalog

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

// Канонические ключи наборов (слоты N, N2, … → sorted set через |).
const (
	DecorSetAttributeKey      = "_decor_set"
	PatternSetAttributeKey      = "_pattern_set"
	MaterialSetAttributeKey     = "_material_set"
	FabricTypeSetAttributeKey   = "_fabric_type_set"
)

// SlotSetGroup — одна семья слотов в фиде (Декор*, Узор*, …).
type SlotSetGroup struct {
	CanonKey      string
	BaseKey       string // «декор», «узор», «материал», «тип ткани» — без цифрового суффикса
	RuleID        string
	LabelRu       string
	DefaultWeight int
}

// AttributeSlotSetGroups — все поддерживаемые семьи слотов.
var AttributeSlotSetGroups = []SlotSetGroup{
	{CanonKey: DecorSetAttributeKey, BaseKey: "декор", RuleID: "decor", LabelRu: "Декор (набор)", DefaultWeight: 12},
	{CanonKey: PatternSetAttributeKey, BaseKey: "узор", RuleID: "pattern", LabelRu: "Узор (набор)", DefaultWeight: 10},
	{CanonKey: MaterialSetAttributeKey, BaseKey: "материал", RuleID: "material", LabelRu: "Материал (набор)", DefaultWeight: 10},
	{CanonKey: FabricTypeSetAttributeKey, BaseKey: "тип ткани", RuleID: "fabric_type", LabelRu: "Тип ткани (набор)", DefaultWeight: 8},
}

var slotSetPatterns = func() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, len(AttributeSlotSetGroups))
	for _, g := range AttributeSlotSetGroups {
		out[g.BaseKey] = regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(g.BaseKey) + `\d*$`)
	}
	return out
}()

// NormalizeAttributeSetToken — нормализация значения для set_overlap.
func NormalizeAttributeSetToken(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.ReplaceAll(s, "ё", "е")
}

// NormalizeDecorToken — alias для совместимости.
func NormalizeDecorToken(s string) string {
	return NormalizeAttributeSetToken(s)
}

// IsSlotSetKey — param «Узор», «Узор2», «Материал3», «Тип ткани2» и т.п.
func IsSlotSetKey(key string, baseKey string) bool {
	p, ok := slotSetPatterns[strings.ToLower(strings.TrimSpace(baseKey))]
	if !ok {
		return false
	}
	return p.MatchString(strings.TrimSpace(key))
}

// IsDecorSlotKey — param «Декор», «Декор2», …
func IsDecorSlotKey(key string) bool {
	return IsSlotSetKey(key, "декор")
}

// IsAnyLegacySlotSetKey — любой из известных слотов (не канонический _*_set).
func IsAnyLegacySlotSetKey(key string) bool {
	for _, g := range AttributeSlotSetGroups {
		if IsSlotSetKey(key, g.BaseKey) {
			return true
		}
	}
	return false
}

// SlotSetGroupForKey — семья по ключу param или canon/rule key.
func SlotSetGroupForKey(key string) *SlotSetGroup {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	keyLower := strings.ToLower(key)
	for i := range AttributeSlotSetGroups {
		g := &AttributeSlotSetGroups[i]
		if key == g.CanonKey || keyLower == strings.ToLower(g.CanonKey) {
			return g
		}
		if keyLower == g.BaseKey || keyLower == g.RuleID {
			return g
		}
		if IsSlotSetKey(key, g.BaseKey) {
			return g
		}
	}
	return nil
}

// CollectSlotSetValues собирает значения из слотов одной семьи.
func CollectSlotSetValues(attrs map[string]string, baseKey string) []string {
	if len(attrs) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for key, val := range attrs {
		if !IsSlotSetKey(key, baseKey) {
			continue
		}
		tok := NormalizeAttributeSetToken(val)
		if tok == "" {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	sort.Strings(out)
	return out
}

// CollectDecorSlotValues — alias.
func CollectDecorSlotValues(attrs map[string]string) []string {
	return CollectSlotSetValues(attrs, "декор")
}

// FormatAttributeSet — каноническая строка набора (sorted, через |).
func FormatAttributeSet(values []string) string {
	if len(values) == 0 {
		return ""
	}
	seen := make(map[string]struct{})
	unique := make([]string, 0, len(values))
	for _, v := range values {
		tok := NormalizeAttributeSetToken(v)
		if tok == "" {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		unique = append(unique, tok)
	}
	sort.Strings(unique)
	return strings.Join(unique, "|")
}

// FormatDecorSet — alias.
func FormatDecorSet(values []string) string {
	return FormatAttributeSet(values)
}

// ParseAttributeSet разбирает canon-строку в множество токенов.
func ParseAttributeSet(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, "|")
	out := make(map[string]struct{})
	for _, p := range parts {
		tok := NormalizeAttributeSetToken(p)
		if tok == "" {
			continue
		}
		out[tok] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseDecorSet — alias.
func ParseDecorSet(raw string) map[string]struct{} {
	return ParseAttributeSet(raw)
}

func mergeOneSlotSetIntoParams(params map[string]string, g SlotSetGroup) {
	if params == nil {
		return
	}
	formatted := FormatAttributeSet(CollectSlotSetValues(params, g.BaseKey))
	if formatted == "" {
		delete(params, g.CanonKey)
		return
	}
	params[g.CanonKey] = formatted
}

// MergeAllSlotSetsIntoParams дописывает все _*_set из слотов фида.
func MergeAllSlotSetsIntoParams(params map[string]string) {
	for _, g := range AttributeSlotSetGroups {
		mergeOneSlotSetIntoParams(params, g)
	}
}

// MergeDecorSetIntoParams — только декор (совместимость).
func MergeDecorSetIntoParams(params map[string]string) {
	mergeOneSlotSetIntoParams(params, AttributeSlotSetGroups[0])
}

// RebuildAttributeSlotSetsInAttributesJSON пересчитывает все _*_set в JSON attributes.
func RebuildAttributeSlotSetsInAttributesJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return raw
	}
	var attrs map[string]string
	if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
		return raw
	}
	MergeAllSlotSetsIntoParams(attrs)
	out, err := json.Marshal(attrs)
	if err != nil {
		return raw
	}
	return string(out)
}

// RebuildDecorSetInAttributesJSON — alias.
func RebuildDecorSetInAttributesJSON(raw string) string {
	return RebuildAttributeSlotSetsInAttributesJSON(raw)
}
