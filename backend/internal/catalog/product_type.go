package catalog

import "strings"

// productTypeParamKeys — допустимые ключи param фида для типа товара (не «Тип ткани», «Тип кармана» и т.п.).
var productTypeParamKeys = []string{
	"Тип",
	"Тип товара",
	"ProductType",
}

// ExtractProductTypeFromParams возвращает тип товара из param фида.
// Используется при импорте, фасетах и CSV-выгрузке.
func ExtractProductTypeFromParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	for _, key := range productTypeParamKeys {
		if val, ok := params[key]; ok {
			if v := normalizeProductTypeValue(val); v != "" {
				return v
			}
		}
	}
	for key, val := range params {
		if normalizeToken(key) == "тип" {
			if v := normalizeProductTypeValue(val); v != "" {
				return v
			}
		}
	}
	return ""
}

// ResolveProductType — тип для UI/фасетов: param → leaf_category → fallback.
func ResolveProductType(params map[string]string, leafCategory, stored string) string {
	if v := ExtractProductTypeFromParams(params); v != "" {
		return v
	}
	if v := strings.TrimSpace(leafCategory); v != "" {
		return v
	}
	return strings.TrimSpace(stored)
}

func normalizeProductTypeValue(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" || v == "<nil>" {
		return ""
	}
	low := strings.ToLower(v)
	if strings.Contains(low, "весна") || strings.Contains(low, "лето") ||
		strings.Contains(low, "осень") || strings.Contains(low, "зима") {
		return ""
	}
	return v
}
