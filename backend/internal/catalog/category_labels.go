package catalog

import (
	"regexp"
	"strings"
)

var (
	reCatalogRoot = regexp.MustCompile(`(?i)^каталог$`)
	reSale        = regexp.MustCompile(`распродаж`)
	reCollection  = regexp.MustCompile(`коллекц`)
	reGift        = regexp.MustCompile(`подар`)
	reContent     = regexp.MustCompile(`content:`)
	reManualPick  = regexp.MustCompile(`ручные подборки`)
	reWowBrand    = regexp.MustCompile(`^wow\s+.+$`)
	reForWomen    = regexp.MustCompile(`для женщин`)
	reForMen      = regexp.MustCompile(`для мужчин`)
	reReadySteady = regexp.MustCompile(`ready!?\s*steady!?\s*go!`)
)

// IsCatalogRootLabel — служебный корень «Каталог».
func IsCatalogRootLabel(label string) bool {
	return reCatalogRoot.MatchString(strings.TrimSpace(label))
}

// IsMarketingCategoryLabel — коллекции, распродажи и прочие не-каталожные ветки.
func IsMarketingCategoryLabel(label string) bool {
	s := normalizeToken(label)
	if s == "" {
		return false
	}
	return reSale.MatchString(s) ||
		reCollection.MatchString(s) ||
		reGift.MatchString(s) ||
		reContent.MatchString(s) ||
		reManualPick.MatchString(s) ||
		reWowBrand.MatchString(s) ||
		reForWomen.MatchString(s) ||
		reForMen.MatchString(s) ||
		reReadySteady.MatchString(s)
}

// IsCatalogSectionRootLabel — верхний раздел ветки «Каталог» (Женщины, Дети, …).
func IsCatalogSectionRootLabel(label string) bool {
	s := normalizeToken(label)
	switch s {
	case "женщины", "женщинам", "мужчины", "мужчинам", "дети", "детям", "подростки", "подросткам":
		return true
	default:
		return false
	}
}

// IsCatalogCategoryLabel — подходит для outfit rule match_type=category.
func IsCatalogCategoryLabel(label string) bool {
	clean := strings.TrimSpace(label)
	return clean != "" && !IsCatalogRootLabel(clean) && !IsMarketingCategoryLabel(clean)
}

// CatalogCategoryLabels — каталожные сегменты path + leaf без дублей.
func CatalogCategoryLabels(categoryPath []string, leafCategory string) []string {
	labels := append([]string{strings.TrimSpace(leafCategory)}, categoryPath...)
	seen := make(map[string]struct{})
	out := make([]string, 0, len(labels))
	for _, label := range labels {
		clean := strings.TrimSpace(label)
		key := normalizeToken(clean)
		if !IsCatalogCategoryLabel(clean) || key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return out
}

// PickRuleCategoryLabel — ближайшая каталожная категория (не «Коллекции»).
func PickRuleCategoryLabel(categoryPath []string, leafCategory string) string {
	candidates := []string{strings.TrimSpace(leafCategory)}
	for i := len(categoryPath) - 1; i >= 0; i-- {
		candidates = append(candidates, strings.TrimSpace(categoryPath[i]))
	}
	for _, label := range candidates {
		if IsCatalogCategoryLabel(label) {
			return label
		}
	}
	return ""
}
