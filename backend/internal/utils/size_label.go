package utils

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	standardSizeSuffix = regexp.MustCompile(`(?i)(?:^|[-_/])(XXS|XS|S|M|L|XL|XXL|XXXL|2XL|3XL|4XL|5XL|\d{2,3})(?:$|[-_/])`)
	skuLikePattern     = regexp.MustCompile(`^[A-Z]{3,}\d{4,}[A-Z]?\d{3,}$`)
)

// ResolveSizeLabel — человекочитаемый размер для чипа (не SKU).
func ResolveSizeLabel(sizeName, sku, attributesJSON string) string {
	sizeName = normalizeSizeToken(sizeName)
	if isReadableSizeLabel(sizeName, sku) {
		return sizeName
	}

	for _, v := range sizeFromAttributes(attributesJSON) {
		if isReadableSizeLabel(v, sku) {
			return v
		}
	}

	if tail := extractStandardSizeFromSKU(sku); tail != "" {
		return tail
	}

	return ""
}

func normalizeSizeToken(raw string) string {
	raw = strings.TrimSpace(raw)
	switch raw {
	case "", "-", "—", "–", "нет", "Нет", "N/A", "n/a":
		return ""
	}
	return raw
}

func isReadableSizeLabel(label, sku string) bool {
	label = strings.TrimSpace(label)
	if label == "" {
		return false
	}
	if strings.EqualFold(label, strings.TrimSpace(sku)) {
		return false
	}
	// Полный артикул GJ (буквы + цифры, без пробелов) — не размер.
	if len(label) >= 10 && !strings.Contains(label, " ") {
		upper := strings.ToUpper(label)
		if strings.Contains(upper, "F") && skuLikePattern.MatchString(upper) {
			return false
		}
	}
	return true
}

func sizeFromAttributes(attributesJSON string) []string {
	attributesJSON = strings.TrimSpace(attributesJSON)
	if attributesJSON == "" || attributesJSON == "{}" {
		return nil
	}
	var attrs map[string]string
	if err := json.Unmarshal([]byte(attributesJSON), &attrs); err != nil {
		return nil
	}
	keys := []string{"размер", "Размер", "РАЗМЕР", "size", "Size", "SIZE"}
	var out []string
	for _, want := range keys {
		for k, v := range attrs {
			if strings.EqualFold(strings.TrimSpace(k), want) {
				v = strings.TrimSpace(v)
				if v != "" {
					out = append(out, v)
				}
			}
		}
	}
	return out
}

func extractStandardSizeFromSKU(sku string) string {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return ""
	}
	if m := standardSizeSuffix.FindStringSubmatch(sku); len(m) > 1 {
		return strings.ToUpper(m[1])
	}
	return ""
}
