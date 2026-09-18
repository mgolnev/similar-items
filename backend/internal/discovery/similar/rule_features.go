package similar

import "strings"

// ruleFeat — прекомпьют производных значений ОДНОЙ строки под ОДНО правило.
// Скоринг пары по правилам дальше работает только сравнениями/пересечениями
// готовых структур: без прекомпьюта каждая из ~73M пар прогрева заново сканила
// AttrMap (EqualFold по русским ключам), нормализовала цвета, токенизировала
// и парсила slot-set'ы — на активном пресете (22 правила) это давало ~150×
// замедление ранжирования.
type ruleFeat struct {
	kind           uint8
	baseApplicable bool
	val            string              // сырое значение правила у строки
	norm           string              // normalizeColorField(val)
	primary        string              // primaryColorToken(val)
	tokens         []string            // colorTokens(val)
	tokenSet       map[string]struct{} // set(tokens)
	fams           map[string]struct{} // colorFamilyInfo(val, families)
	multi          bool                // true для явно составного цвета
	set            map[string]struct{} // resolveAttributeSet (set_overlap)
	// gender exact: контекст строки для teen-совместимости без доступа к ProductRow в hot path
	section    string
	boyTeen    bool
	neutral    bool
	genderRule bool
}

const (
	ruleKindUnknown uint8 = iota
	ruleKindExact
	ruleKindTokenOverlap
	ruleKindSetOverlap
	ruleKindColorSimilarity
)

func ruleKind(matchType string) uint8 {
	switch matchType {
	case "exact":
		return ruleKindExact
	case "token_overlap":
		return ruleKindTokenOverlap
	case "set_overlap":
		return ruleKindSetOverlap
	case "color_similarity":
		return ruleKindColorSimilarity
	default:
		return ruleKindUnknown
	}
}

func tokensToSet(tokens []string) map[string]struct{} {
	if len(tokens) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		out[t] = struct{}{}
	}
	return out
}

func computeRuleFeat(row ProductRow, rule AttributeRuleConfig, families map[string][]string) ruleFeat {
	f := ruleFeat{
		kind:           ruleKind(rule.MatchType),
		baseApplicable: rule.Enabled && ruleMatchesScope(rule, normalizedRootSection(row)),
	}
	switch rule.MatchType {
	case "exact":
		f.norm = normalizeColorField(resolveRuleValue(row, rule.Source, rule.Key))
		f.neutral = isNeutralNormalizedAttributeValue(f.norm)
		f.genderRule = isGenderRule(rule)
		if f.genderRule {
			f.section = strings.TrimSpace(strings.ToLower(row.RootSection))
			f.boyTeen = row.BoyTeen
		}
	case "token_overlap":
		f.val = resolveRuleValue(row, rule.Source, rule.Key)
		f.tokens = colorTokens(f.val)
		f.tokenSet = tokensToSet(f.tokens)
	case "set_overlap":
		f.set = resolveAttributeSet(row, rule.Source, rule.Key)
	case "color_similarity":
		f.val = resolveRuleValue(row, rule.Source, rule.Key)
		if f.val != "" {
			f.norm = normalizeColorField(f.val)
			f.primary = primaryColorToken(f.val)
			f.tokens = colorTokens(f.val)
			f.tokenSet = tokensToSet(f.tokens)
			f.fams, f.multi = colorFamilyInfo(f.val, families)
		}
	}
	return f
}

// precomputeRuleFeatures считает фичи всех включённых правил для строки.
// Вызывается из preparedPool (до старта warm-воркеров — без гонок) и для
// базовой строки запроса.
func precomputeRuleFeatures(row *ProductRow, cfg StrategyConfig) {
	if len(cfg.Rules.Rules) == 0 {
		row.ruleFeats = nil
		return
	}
	feats := make([]ruleFeat, len(cfg.Rules.Rules))
	for i, rule := range cfg.Rules.Rules {
		if !rule.Enabled {
			continue
		}
		feats[i] = computeRuleFeat(*row, rule, cfg.Rules.ColorFamilies)
	}
	row.ruleFeats = feats
}

// scoreOneRuleFeats — горячий путь scoreOneRule по прекомпьюченным фичам.
// Семантика 1:1 с медленными вариантами (scoreExactRule и т.д.).
func scoreOneRuleFeats(bf, cf *ruleFeat, rule *AttributeRuleConfig) int {
	switch bf.kind {
	case ruleKindExact:
		if bf.norm == "" || cf.norm == "" || bf.neutral || cf.neutral {
			return 0
		}
		if bf.norm == cf.norm {
			return ruleExactWeight(*rule)
		}
		if bf.genderRule && genderCompatibleFeats(*bf, *cf) {
			return ruleExactWeight(*rule)
		}
		return 0
	case ruleKindTokenOverlap:
		overlap := shadeOverlapFeat(bf, cf)
		if overlap == 0 {
			return 0
		}
		w := rule.Weight
		if w <= 0 {
			w = rule.MaxScore
		}
		score := w * overlap / 100
		if score <= 0 && overlap > 0 {
			score = 1
		}
		return score
	case ruleKindSetOverlap:
		if len(bf.set) == 0 || len(cf.set) == 0 {
			return 0
		}
		w := rule.Weight
		if w <= 0 {
			w = rule.MaxScore
		}
		return familySetSimilarityScore(bf.set, cf.set, w)
	case ruleKindColorSimilarity:
		return scoreColorSimilarityFeats(bf, cf, rule)
	default:
		return 0
	}
}

// shadeOverlapFeat — shadeOverlapScore по готовым токенам.
func shadeOverlapFeat(a, b *ruleFeat) int {
	if len(a.tokens) == 0 || len(b.tokens) == 0 {
		return 0
	}
	common := 0
	for _, t := range a.tokens {
		if _, ok := b.tokenSet[t]; ok {
			common++
		}
	}
	union := len(a.tokens) + len(b.tokens) - common
	if union == 0 {
		return 0
	}
	ratio := float64(common) / float64(union)
	if ratio < 0.25 {
		return 0
	}
	return int(ratio*100 + 0.5)
}

// scoreColorSimilarityFeats — зеркало scoreColorSimilarityRule по фичам.
func scoreColorSimilarityFeats(bf, cf *ruleFeat, rule *AttributeRuleConfig) int {
	if bf.val == "" || cf.val == "" {
		return 0
	}
	cfgC := rule.Color
	if cfgC == nil {
		def := ColorSimilarityConfig{
			RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
			FamilyScore: 18, ShadeOverlapScore: 6, ConflictPenalty: -20,
		}
		cfgC = &def
	}

	score := 0
	knownFamiliesDisjoint := len(bf.fams) > 0 && len(cf.fams) > 0 && !colorFamilySetsOverlap(bf.fams, cf.fams)
	if bf.norm == cf.norm {
		score = cfgC.RawExactScore
	} else if knownFamiliesDisjoint {
		return cfgC.ConflictPenalty
	} else if bf.multi != cf.multi {
		return 0
	} else if bf.multi && cf.multi {
		score = familySetSimilarityScore(bf.fams, cf.fams, cfgC.FamilyScore)
	} else if bf.primary == cf.primary && cfgC.PrimaryExactScore > score {
		score = cfgC.PrimaryExactScore
	} else if !knownFamiliesDisjoint {
		for _, t := range bf.tokens {
			if _, ok := cf.tokenSet[t]; ok && cfgC.AliasExactScore > score {
				score = cfgC.AliasExactScore
				break
			}
		}
	}

	if !bf.multi && !cf.multi && len(bf.fams) > 0 && len(cf.fams) > 0 {
		for f := range bf.fams {
			if _, ok := cf.fams[f]; ok && cfgC.FamilyScore > score {
				score = cfgC.FamilyScore
			}
		}
	}

	allowShadeOverlap := !knownFamiliesDisjoint
	if shade := shadeOverlapFeat(bf, cf); allowShadeOverlap && shade > 0 {
		add := cfgC.ShadeOverlapScore * shade / 100
		if add > 0 {
			score += add
		}
	}

	if score <= 0 {
		return 0
	}
	max := rule.MaxScore
	if max <= 0 {
		max = rule.Weight
	}
	if max > 0 && score > max {
		score = max
	}
	return score
}
