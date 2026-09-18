package similar

import (
	"regexp"
	"strings"
)

var colorTokenSplit = regexp.MustCompile(`[/(),;|]+`)

// colorFamilySetsForString возвращает возможные семейства отдельно для каждого
// распознанного компонента цвета. Несколько семейств у одного компонента — это
// неоднозначный alias (например, «молочный» может быть beige и white), а не
// признак составного цвета.
func colorFamilySetsForString(color string, families map[string][]string) []map[string]struct{} {
	var found []map[string]struct{}
	color = normalizeColorField(color)
	for _, rawSegment := range colorTokenSplit.Split(color, -1) {
		segment := strings.Join(strings.Fields(rawSegment), " ")
		if segment == "" {
			continue
		}

		// Полная фраза точнее отдельных слов. Например, «бежевый меланж»
		// явно задан как beige и не должен дополнительно стать gray только
		// потому, что отдельный alias «меланж» находится в gray.
		exact := make(map[string]struct{})
		for famID, aliases := range families {
			for _, alias := range aliases {
				if normalizeColorField(alias) == segment {
					exact[famID] = struct{}{}
					break
				}
			}
		}
		if len(exact) > 0 {
			found = append(found, exact)
			continue
		}

		// Внутри одного сегмента берём первое распознанное цветовое слово.
		// Остальные слова обычно являются модификаторами («меланж»). Составные
		// цвета в фиде разделяются / , ( ) и другими разделителями выше.
		for _, tok := range strings.Fields(segment) {
			matched := make(map[string]struct{})
			for famID, aliases := range families {
				for _, alias := range aliases {
					alias = normalizeColorField(alias)
					// Многословный alias сравнивается только с полной фразой выше:
					// токен «меланж» не должен совпадать с «бежевый меланж».
					if strings.ContainsAny(alias, " \t\r\n") {
						continue
					}
					if tokenMatchesAlias(tok, alias) != "" {
						matched[famID] = struct{}{}
						break
					}
				}
			}
			if len(matched) == 0 {
				continue
			}
			found = append(found, matched)
			break
		}
	}
	return found
}

// colorFamilyInfo возвращает объединение возможных семейств и признак
// настоящего multi-color. Цвет считается составным, только если его явно
// разделённые компоненты невозможно отнести к одному общему семейству.
func colorFamilyInfo(color string, families map[string][]string) (map[string]struct{}, bool) {
	segmentSets := colorFamilySetsForString(color, families)
	all := make(map[string]struct{})
	if len(segmentSets) == 0 {
		return all, false
	}

	common := make(map[string]struct{}, len(segmentSets[0]))
	for famID := range segmentSets[0] {
		common[famID] = struct{}{}
	}
	for _, segmentSet := range segmentSets {
		for famID := range segmentSet {
			all[famID] = struct{}{}
		}
		for famID := range common {
			if _, ok := segmentSet[famID]; !ok {
				delete(common, famID)
			}
		}
	}

	return all, len(segmentSets) > 1 && len(common) == 0
}

// colorFamiliesForString возвращает id семейств, найденных в строке цвета.
func colorFamiliesForString(color string, families map[string][]string) map[string]struct{} {
	found, _ := colorFamilyInfo(color, families)
	return found
}

func colorFamilySetsOverlap(a, b map[string]struct{}) bool {
	for fam := range a {
		if _, ok := b[fam]; ok {
			return true
		}
	}
	return false
}

func familySetSimilarityScore(a, b map[string]struct{}, maxScore int) int {
	if maxScore <= 0 || len(a) == 0 || len(b) == 0 {
		return 0
	}
	common := 0
	for fam := range a {
		if _, ok := b[fam]; ok {
			common++
		}
	}
	if common == 0 {
		return 0
	}
	union := len(a) + len(b) - common
	if union <= 0 {
		return 0
	}
	score := maxScore * common / union
	if score == 0 {
		score = 1
	}
	return score
}

func colorTokens(color string) []string {
	color = strings.TrimSpace(strings.ToLower(color))
	color = strings.ReplaceAll(color, "ё", "е")
	if color == "" {
		return nil
	}
	parts := colorTokenSplit.Split(color, -1)
	var out []string
	seen := make(map[string]struct{})
	for _, p := range parts {
		for _, w := range strings.Fields(p) {
			w = strings.TrimSpace(w)
			if len([]rune(w)) < 2 {
				continue
			}
			if _, ok := seen[w]; ok {
				continue
			}
			seen[w] = struct{}{}
			out = append(out, w)
		}
	}
	if len(out) == 0 && color != "" {
		return []string{color}
	}
	return out
}

func tokenMatchesAlias(tok, alias string) string {
	alias = strings.TrimSpace(strings.ToLower(alias))
	alias = strings.ReplaceAll(alias, "ё", "е")
	if tok == alias {
		return tok
	}
	if strings.Contains(tok, alias) || strings.Contains(alias, tok) {
		if len([]rune(alias)) >= 3 || len([]rune(tok)) >= 3 {
			return alias
		}
	}
	return ""
}

func primaryColorToken(color string) string {
	tokens := colorTokens(color)
	if len(tokens) == 0 {
		return strings.TrimSpace(strings.ToLower(color))
	}
	return tokens[0]
}

func shadeOverlapScore(a, b string) int {
	ta := colorTokens(a)
	tb := colorTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	setB := make(map[string]struct{}, len(tb))
	for _, t := range tb {
		setB[t] = struct{}{}
	}
	common := 0
	for _, t := range ta {
		if _, ok := setB[t]; ok {
			common++
		}
	}
	union := len(ta) + len(tb) - common
	if union == 0 {
		return 0
	}
	ratio := float64(common) / float64(union)
	if ratio < 0.25 {
		return 0
	}
	return int(ratio*100 + 0.5)
}

func normalizeColorField(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "ё", "е")
	return s
}
