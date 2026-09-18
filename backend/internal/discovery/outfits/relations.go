package outfits

import (
	"encoding/json"
	"strings"
)

func relatedReferences(attributes string, cfg StrategyConfig) []string {
	attributes = strings.TrimSpace(attributes)
	if attributes == "" || attributes == "{}" {
		return nil
	}
	var attrs map[string]string
	if err := json.Unmarshal([]byte(attributes), &attrs); err != nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(cfg.AttributeKeys))
	for _, key := range cfg.AttributeKeys {
		allowed[normalizeAttributeKey(key)] = struct{}{}
	}
	seen := make(map[string]struct{})
	refs := make([]string, 0)
	for key, value := range attrs {
		if _, ok := allowed[normalizeAttributeKey(key)]; !ok {
			continue
		}
		for _, ref := range splitReferences(value) {
			normalized := normalizeArticle(ref)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			refs = append(refs, ref)
		}
	}
	return refs
}

func splitReferences(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var jsonValues []string
	if strings.HasPrefix(raw, "[") && json.Unmarshal([]byte(raw), &jsonValues) == nil {
		return cleanReferences(jsonValues)
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '|', '\n', '\r', '\t':
			return true
		default:
			return false
		}
	})
	return cleanReferences(parts)
}

func cleanReferences(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), `"'[]`)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeArticle(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
