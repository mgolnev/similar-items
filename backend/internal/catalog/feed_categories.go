package catalog

import "strings"

// ParseCategorySegmentName извлекает название из пары "id:Название".
func ParseCategorySegmentName(pair string) string {
	pair = strings.TrimSpace(pair)
	if pair == "" {
		return ""
	}
	if idx := strings.Index(pair, ":"); idx >= 0 {
		return strings.TrimSpace(pair[idx+1:])
	}
	return pair
}

// ParseCategoryChainsFromFeedField разбивает поле products.categories на цепочки YML.
func ParseCategoryChainsFromFeedField(raw string) [][]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, " | ")
	chains := make([][]string, 0)
	var current []string

	for _, part := range parts {
		name := ParseCategorySegmentName(part)
		if name == "" {
			continue
		}
		rootToken := normalizeToken(name)
		if rootToken == "каталог" || rootToken == "коллекции" {
			if len(current) > 0 {
				chains = append(chains, current)
			}
			current = []string{name}
			continue
		}
		if len(current) == 0 {
			current = []string{name}
			continue
		}
		current = append(current, name)
	}
	if len(current) > 0 {
		chains = append(chains, current)
	}
	return chains
}

func isCatalogRootChain(chain []string) bool {
	return len(chain) > 0 && IsCatalogRootLabel(chain[0])
}

// CatalogLeafPathsFromFeedField — полные пути конечных категорий из всех
// независимых веток «Каталог» в поле categories.
//
// Фид перечисляет одну и ту же иерархию несколькими цепочками: отдельно
// родитель и отдельно каждый дочерний узел. Поэтому путь является листом, если
// он не служит префиксом более глубокого пути. Сравнивать только глобальную
// максимальную глубину нельзя: у товара могут одновременно быть валидные
// параллельные ветки разной длины (например, «Женские спортивные брюки» и
// «Женские свободные брюки»).
//
// Полный путь важен: в разных ветках могут существовать разные узлы с
// одинаковым названием leaf (например, «Чёрная форма» для девочек и мальчиков).
func CatalogLeafPathsFromFeedField(raw string) [][]string {
	chains := ParseCategoryChainsFromFeedField(raw)
	paths := make([][]string, 0, len(chains))
	normalizedPaths := make([][]string, 0, len(chains))
	seen := make(map[string]struct{})
	for _, chain := range chains {
		if !isCatalogRootChain(chain) {
			continue
		}
		path := collapseAdjacentCategorySegments(catalogPathFromChain(chain))
		if len(path) == 0 {
			continue
		}
		normalized := normalizeCategoryPath(path)
		key := strings.Join(normalized, "\x1f")
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, path)
		normalizedPaths = append(normalizedPaths, normalized)
	}

	out := make([][]string, 0, len(paths))
	for i, path := range paths {
		isParent := false
		for j := range paths {
			if i == j || len(normalizedPaths[i]) >= len(normalizedPaths[j]) {
				continue
			}
			if isCategoryPathPrefix(normalizedPaths[i], normalizedPaths[j]) {
				isParent = true
				break
			}
		}
		if isParent {
			continue
		}
		out = append(out, append([]string(nil), path...))
	}
	return out
}

func collapseAdjacentCategorySegments(path []string) []string {
	out := make([]string, 0, len(path))
	last := ""
	for _, segment := range path {
		segment = strings.TrimSpace(segment)
		token := normalizeToken(segment)
		if segment == "" || token == "" || token == last {
			continue
		}
		out = append(out, segment)
		last = token
	}
	return out
}

func normalizeCategoryPath(path []string) []string {
	out := make([]string, 0, len(path))
	for _, segment := range path {
		if token := normalizeToken(segment); token != "" {
			out = append(out, token)
		}
	}
	return out
}

func isCategoryPathPrefix(prefix, path []string) bool {
	if len(prefix) >= len(path) {
		return false
	}
	for i := range prefix {
		if prefix[i] != path[i] {
			return false
		}
	}
	return true
}

// CatalogLeafLabelsFromFeedField — листовые категории из каждой ветки «Каталог» в поле categories.
// Один товар может иметь несколько leaf (оверсайз и короткий рукав и т.д.).
func CatalogLeafLabelsFromFeedField(raw string) []string {
	paths := CatalogLeafPathsFromFeedField(raw)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		out = append(out, strings.TrimSpace(path[len(path)-1]))
	}
	return out
}

// CatalogCategoryLabelsFromFeedField — все каталожные сегменты из всех веток «Каталог».
func CatalogCategoryLabelsFromFeedField(raw string) []string {
	chains := ParseCategoryChainsFromFeedField(raw)
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, chain := range chains {
		if !isCatalogRootChain(chain) {
			continue
		}
		for i, label := range chain {
			if i == 0 && IsCatalogRootLabel(label) {
				continue
			}
			if !IsCatalogCategoryLabel(label) {
				continue
			}
			key := normalizeToken(label)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, label)
		}
	}
	return out
}

// PickRuleCategoryLabelFromFeedField — первая каталожная категория из веток «Каталог».
func PickRuleCategoryLabelFromFeedField(raw string) string {
	for _, chain := range ParseCategoryChainsFromFeedField(raw) {
		if !isCatalogRootChain(chain) {
			continue
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if IsCatalogCategoryLabel(chain[i]) {
				return chain[i]
			}
		}
	}
	return ""
}

// CatalogBreadcrumbPathFromFeedField — самая длинная цепочка «Каталог → …» для хлебных крошек PDP.
func CatalogBreadcrumbPathFromFeedField(raw string) []string {
	return bestCatalogPathFromFeedField(raw, false)
}

// CatalogNavigationPathFromFeedField — цепочка для навигации по разделам витрины.
// Берёт самую длинную ветку «Каталог», у которой первый сегмент — Женщины/Мужчины/Дети/Подростки.
// Отсекает маркетинговые подветки вроде Ready! Steady! Go!, даже если они длиннее.
func CatalogNavigationPathFromFeedField(raw string) []string {
	return bestCatalogPathFromFeedField(raw, true)
}

func bestCatalogPathFromFeedField(raw string, requireSectionRoot bool) []string {
	chains := ParseCategoryChainsFromFeedField(raw)
	var best []string
	for _, chain := range chains {
		if !isCatalogRootChain(chain) {
			continue
		}
		path := catalogPathFromChain(chain)
		if requireSectionRoot && (len(path) == 0 || !IsCatalogSectionRootLabel(path[0])) {
			continue
		}
		if len(path) > len(best) {
			best = path
		} else if !requireSectionRoot && len(path) == len(best) && len(path) > 0 {
			best = path
		}
	}
	return best
}

func catalogPathFromChain(chain []string) []string {
	path := make([]string, 0, len(chain))
	for i, label := range chain {
		if i == 0 && IsCatalogRootLabel(label) {
			continue
		}
		if !IsCatalogCategoryLabel(label) {
			continue
		}
		path = append(path, label)
	}
	return path
}

// CatalogBreadcrumbPath — path для крошек: feed chains → fallback category_path без маркетинговых веток.
func CatalogBreadcrumbPath(categoriesFeedField string, categoryPath []string, leafCategory string) []string {
	if path := CatalogBreadcrumbPathFromFeedField(categoriesFeedField); len(path) > 0 {
		return path
	}
	labels := CatalogCategoryLabels(categoryPath, leafCategory)
	if len(labels) == 0 {
		return nil
	}
	// CatalogCategoryLabels не сохраняет порядок path — восстанавливаем из categoryPath.
	path := make([]string, 0, len(categoryPath))
	for _, label := range categoryPath {
		clean := strings.TrimSpace(label)
		if !IsCatalogCategoryLabel(clean) {
			continue
		}
		path = append(path, clean)
	}
	if len(path) > 0 {
		return path
	}
	return labels
}
