package catalog

import "strings"

// rsgCategoryIDs — дерево Ready! Steady! Go! в YML (2709+), без правок PIM.
var rsgCategoryIDs = map[int]struct{}{
	2709: {}, 2710: {}, 2711: {},
	2712: {}, 2713: {}, 2714: {}, 2715: {}, 2716: {}, 2717: {}, 2718: {}, 2719: {},
	2721: {}, 2722: {}, 2723: {}, 2724: {}, 2725: {}, 2726: {}, 2727: {}, 2728: {},
	2729: {}, 2730: {}, 2731: {}, 2732: {}, 2733: {}, 2734: {}, 2735: {}, 2736: {},
	2737: {}, 2739: {}, 2740: {},
}

// IsRSGCategoryID — категория из коллекции RSG в фиде.
func IsRSGCategoryID(id int) bool {
	if _, ok := rsgCategoryIDs[id]; ok {
		return true
	}
	return false
}

// HasRSGCategoryInFeed — в поле categories есть ID RSG.
func HasRSGCategoryInFeed(categoriesFeedField string) bool {
	for _, part := range strings.Split(categoriesFeedField, " | ") {
		if IsRSGCategoryID(parseCategoryIDFromSegment(part)) {
			return true
		}
	}
	return false
}

// IsTeenBoyCatalogLabel — подростковая ветка для парней (RSG и «… для парней»).
func IsTeenBoyCatalogLabel(label string) bool {
	s := normalizeToken(label)
	if s == "" {
		return false
	}
	if strings.Contains(s, "для парней") {
		return true
	}
	return strings.Contains(s, "мальчик") && strings.Contains(s, "подрост")
}

// TeenProductClass — укрупнённый тип внутри teens для cross-leaf affinity.
func TeenProductClass(leaf string, path []string) string {
	s := normalizeToken(leaf)
	if s == "" && len(path) > 0 {
		s = normalizeToken(path[len(path)-1])
	}
	if s == "" {
		return ""
	}
	switch {
	case strings.Contains(s, "джинс"):
		return "jeans"
	case strings.Contains(s, "брюк"):
		return "pants"
	case strings.Contains(s, "шорт"):
		return "shorts"
	case strings.Contains(s, "юбк"):
		return "skirts"
	case strings.Contains(s, "плать"), strings.Contains(s, "сараф"):
		return "dresses"
	case strings.Contains(s, "худи"):
		return "hoodie"
	case strings.Contains(s, "свитшот"), strings.Contains(s, "толстов"):
		return "sweatshirt"
	case strings.Contains(s, "футбол"), strings.Contains(s, "лонгслив"), strings.Contains(s, "топ"):
		return "tops"
	case strings.Contains(s, "курт"), strings.Contains(s, "ветров"), strings.Contains(s, "бомбер"),
		strings.Contains(s, "пухов"), strings.Contains(s, "жилет"):
		return "outerwear"
	case strings.Contains(s, "рубаш"):
		return "shirts"
	case strings.Contains(s, "кардиг"), strings.Contains(s, "джемпер"), strings.Contains(s, "свитер"):
		return "knit"
	case strings.Contains(s, "поло"):
		return "polo"
	case strings.Contains(s, "спортив"):
		return "sportswear"
	default:
		return ""
	}
}

func isTeenGirlByLabels(path []string, leaf, feed string) bool {
	for _, seg := range path {
		if IsTeenGirlCatalogLabel(seg) {
			return true
		}
	}
	if IsTeenGirlCatalogLabel(leaf) {
		return true
	}
	return hasRSGGirlBranchInFeed(feed)
}

func isTeenBoyByLabels(path []string, leaf, feed string) bool {
	for _, seg := range path {
		if IsTeenBoyCatalogLabel(seg) {
			return true
		}
	}
	if IsTeenBoyCatalogLabel(leaf) {
		return true
	}
	return hasRSGBoyBranchInFeed(feed)
}

func hasRSGGirlBranchInFeed(feed string) bool {
	for _, part := range strings.Split(feed, " | ") {
		id := parseCategoryIDFromSegment(part)
		switch id {
		case 2710, 2712, 2713, 2714, 2715, 2716, 2717, 2718, 2719, 2721, 2722, 2723, 2724, 2725, 2726, 2727, 2728:
			return true
		}
	}
	return false
}

func hasRSGBoyBranchInFeed(feed string) bool {
	for _, part := range strings.Split(feed, " | ") {
		id := parseCategoryIDFromSegment(part)
		switch id {
		case 2711, 2729, 2730, 2731, 2732, 2733, 2734, 2735, 2736, 2737, 2739, 2740:
			return true
		}
	}
	return false
}

// ResolveTeenSubsection — teen_boys / teen_girls для скоринга similar.
func ResolveTeenSubsection(path []string, leaf, gender, categoriesFeed string) Subsection {
	path = DropCatalogPrefix(path)
	root := ResolveRootSection(categoriesFeed, path, gender)
	if root != SectionTeens {
		return ""
	}
	if sub := ClassifySubsection(root, path, gender); sub == SubTeenBoys || sub == SubTeenGirls {
		return sub
	}
	if isTeenBoyByLabels(path, leaf, categoriesFeed) {
		return SubTeenBoys
	}
	if isTeenGirlByLabels(path, leaf, categoriesFeed) {
		return SubTeenGirls
	}
	return ""
}

func pathStartsWithPodrostki(path []string) bool {
	return len(path) > 0 && normalizeToken(path[0]) == "подростки"
}

func subsectionAnchorLabel(sub Subsection) string {
	switch sub {
	case SubTeenGirls:
		return "Девочки-подростки"
	case SubTeenBoys:
		return "Мальчики-подростки"
	default:
		return ""
	}
}

// NormalizeTeenDiscoveryPath дополняет shallow RSG-path префиксом «Подростки → …».
func NormalizeTeenDiscoveryPath(path []string, categoriesFeed, gender string) []string {
	path = DropCatalogPrefix(path)
	if len(path) == 0 {
		return path
	}
	if pathStartsWithPodrostki(path) {
		return path
	}
	root := ResolveRootSection(categoriesFeed, path, gender)
	if root != SectionTeens {
		return path
	}
	sub := ResolveTeenSubsection(path, path[len(path)-1], gender, categoriesFeed)
	anchor := subsectionAnchorLabel(sub)
	if anchor == "" {
		return path
	}
	// Уже есть якорь подсекции — не дублируем.
	for _, seg := range path {
		if normalizeToken(seg) == normalizeToken(anchor) {
			return append([]string{"Подростки"}, path...)
		}
	}
	return append([]string{"Подростки", anchor}, path...)
}
