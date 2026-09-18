package catalog

import (
	"regexp"
	"strconv"
	"strings"
)

// RootSection — верхний раздел каталога (как в client discovery).
type RootSection string

const (
	SectionWomen   RootSection = "women"
	SectionMen     RootSection = "men"
	SectionKids    RootSection = "kids"
	SectionTeens   RootSection = "teens"
	SectionUnknown RootSection = "unknown"
)

// Subsection — уточнение для kids/teens.
type Subsection string

const (
	SubBoys      Subsection = "boys"
	SubGirls     Subsection = "girls"
	SubTeenBoys  Subsection = "teen_boys"
	SubTeenGirls Subsection = "teen_girls"
)

var (
	reWomen = regexp.MustCompile(`(?i)(жен|woman|female|ladies)`)
	reMen   = regexp.MustCompile(`(?i)(муж|man|male|men\b)`)
	reKids  = regexp.MustCompile(`(?i)(дет|kid|child|малыш|мальчик|девоч)`)
	reTeens = regexp.MustCompile(`(?i)(подрост|teen)`)

)

func normalizeToken(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.ReplaceAll(s, "ё", "е")
}

// DropCatalogPrefix убирает служебный корень «Каталог» / «Коллекции».
func DropCatalogPrefix(path []string) []string {
	return NormalizeCategoryPathForSimilarity(path)
}

// NormalizeCategoryPathForSimilarity — путь категории без служебных корней YML.
func NormalizeCategoryPathForSimilarity(path []string) []string {
	p := append([]string(nil), path...)
	for len(p) > 0 {
		head := normalizeToken(p[0])
		if head == "каталог" || head == "коллекции" {
			p = p[1:]
			continue
		}
		break
	}
	return p
}

// InferRootSectionFromGender — подсказка из param «Пол».
func InferRootSectionFromGender(gender string) RootSection {
	if gender == "" {
		return ""
	}
	x := normalizeToken(gender)
	switch {
	case reWomen.MatchString(x):
		return SectionWomen
	case reMen.MatchString(x):
		return SectionMen
	case reTeens.MatchString(x):
		return SectionTeens
	case reKids.MatchString(x):
		return SectionKids
	default:
		return ""
	}
}

// rootSectionFromSegment определяет раздел по одному сегменту path (не по склеенной строке:
// иначе «зауженные» ложно матчит women-regex на подстроку «жен»).
func rootSectionFromSegment(segment string) RootSection {
	s := normalizeToken(segment)
	if s == "" {
		return ""
	}
	switch s {
	case "женщины", "женщинам":
		return SectionWomen
	case "мужчины", "мужчинам":
		return SectionMen
	case "дети", "детям":
		return SectionKids
	case "подростки", "подросткам":
		return SectionTeens
	}
	if strings.HasPrefix(s, "женщин") || strings.HasPrefix(s, "женск") {
		return SectionWomen
	}
	if strings.HasPrefix(s, "мужчин") || strings.HasPrefix(s, "мужск") {
		if strings.Contains(s, "дет") {
			return ""
		}
		return SectionMen
	}
	if strings.HasPrefix(s, "подрост") {
		return SectionTeens
	}
	if strings.Contains(s, "подрост") && (strings.Contains(s, "девоч") || strings.Contains(s, "девуш") || strings.Contains(s, "мальчик")) {
		return SectionTeens
	}
	if strings.HasPrefix(s, "дет") || strings.HasPrefix(s, "мальчик") || strings.HasPrefix(s, "девоч") {
		return SectionKids
	}
	return ""
}

// IsTeenGirlCatalogLabel — каталожный сегмент подростковой коллекции для девушек (RSG, «… для девушек», «… девочек-подростков»).
func IsTeenGirlCatalogLabel(label string) bool {
	s := normalizeToken(label)
	if s == "" {
		return false
	}
	if strings.Contains(s, "мальчик") {
		return false
	}
	if strings.Contains(s, "для девушек") {
		return true
	}
	if strings.Contains(s, "девоч") && strings.Contains(s, "подрост") {
		return true
	}
	return false
}

// HasRSGTeenCategoryInFeed — в YML categories есть ID подростковой коллекции RSG.
func HasRSGTeenCategoryInFeed(categoriesFeedField string) bool {
	return HasRSGCategoryInFeed(categoriesFeedField)
}

func parseCategoryIDFromSegment(pair string) int {
	pair = strings.TrimSpace(pair)
	if pair == "" {
		return 0
	}
	idx := strings.Index(pair, ":")
	if idx <= 0 {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(pair[:idx]))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func rootSectionFromPath(path []string) RootSection {
	for _, seg := range path {
		if sec := rootSectionFromSegment(seg); sec != "" {
			return sec
		}
	}
	for _, seg := range path {
		if IsTeenGirlCatalogLabel(seg) {
			return SectionTeens
		}
	}
	return ""
}

// ResolveRootSection — root_section для discovery/similar: path → teen-girl labels → RSG IDs → пол.
func ResolveRootSection(categoriesFeedField string, categoryPath []string, gender string) RootSection {
	path := DropCatalogPrefix(categoryPath)
	if sec := rootSectionFromPath(path); sec != "" {
		return sec
	}
	if HasRSGTeenCategoryInFeed(categoriesFeedField) {
		return SectionTeens
	}
	if g := InferRootSectionFromGender(gender); g != "" {
		return g
	}
	return SectionUnknown
}

// ClassifyRootSection определяет раздел по пути категории и опционально полу.
func ClassifyRootSection(categoryPath []string, gender string) RootSection {
	return ResolveRootSection("", categoryPath, gender)
}

// ClassifySubsection — boys/girls для kids и teens.
func ClassifySubsection(root RootSection, categoryPath []string, gender string) Subsection {
	joined := normalizeToken(strings.Join(append(categoryPath, gender), " | "))
	switch root {
	case SectionKids:
		if regexp.MustCompile(`(?i)(мальчик|мальч|boys|boy)`).MatchString(joined) {
			return SubBoys
		}
		if regexp.MustCompile(`(?i)(девоч|девушк|girls|girl)`).MatchString(joined) {
			return SubGirls
		}
	case SectionTeens:
		if regexp.MustCompile(`(?i)(мальчик|мальч|boys|boy)`).MatchString(joined) {
			return SubTeenBoys
		}
		if regexp.MustCompile(`(?i)(девоч|девушк|girls|girl)`).MatchString(joined) {
			return SubTeenGirls
		}
	}
	return ""
}

// NormalizeColor приводит цвет к каноническому виду для поиска/similar.
func NormalizeColor(color string) string {
	return normalizeToken(strings.TrimSpace(color))
}

// ExtractColorFromGroupID — суффикс цвета в group_id фида («GSH012871-1 светло-голубой …»).
func ExtractColorFromGroupID(groupID string) string {
	groupID = strings.TrimSpace(groupID)
	idx := strings.Index(groupID, " ")
	if idx <= 0 {
		return ""
	}
	prefix := strings.TrimSpace(groupID[:idx])
	if !strings.Contains(prefix, "-") {
		return ""
	}
	return strings.TrimSpace(groupID[idx+1:])
}
