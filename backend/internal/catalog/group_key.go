package catalog

import "strings"

// CatalogGroupKey — короткий ключ карточки из group_id/sales_sku.
// «GAC025757-1 темно-зеленый» → «GAC025757-1».
func CatalogGroupKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, " "); idx > 0 {
		return strings.TrimSpace(raw[:idx])
	}
	return raw
}

// RelatedGroupKeys — ключи для поиска связанных карточек (полный + короткий).
func RelatedGroupKeys(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	short := CatalogGroupKey(raw)
	if short == raw {
		return []string{raw}
	}
	return []string{raw, short}
}
