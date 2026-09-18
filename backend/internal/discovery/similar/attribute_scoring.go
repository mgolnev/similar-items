package similar

import (
	"encoding/json"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

func scoreAttributeRules(base, cand ProductRow, cfg StrategyConfig) (total int, byRule map[string]int) {
	if len(cfg.Rules.Rules) == 0 {
		return 0, nil
	}
	section := normalizedRootSection(base)
	for i, rule := range cfg.Rules.Rules {
		if !rule.Enabled {
			continue
		}
		if !ruleMatchesScope(rule, section) {
			continue
		}
		score := scoreOneRuleAt(&base, &cand, &rule, i, cfg.Rules.ColorFamilies)
		if score == 0 {
			continue
		}
		if rule.MaxScore > 0 && score > rule.MaxScore {
			score = rule.MaxScore
		}
		// Лениво: карту аллоцируем только когда есть что класть. Иначе на каждой из
		// ~миллионов пар при прогреве создавалась пустая map — главный источник
		// GC-churn → OOM. Потребители читают nil-map безопасно.
		if byRule == nil {
			byRule = make(map[string]int, len(cfg.Rules.Rules))
		}
		byRule[rule.ID] = score
		total += score
	}
	return total, byRule
}

// scoreAttributeRulesTotal — production/builder hot path: та же сумма, что у
// scoreAttributeRules, но без диагностической map rule_id→score. На полном
// прогреве эта map создавалась для каждой пары с хотя бы одним совпавшим
// правилом, хотя публичной выдаче нужны только итоговые score и ID.
func scoreAttributeRulesTotal(base, cand *ProductRow, cfg *StrategyConfig) (total int) {
	if len(cfg.Rules.Rules) == 0 {
		return 0
	}
	if len(base.ruleFeats) == len(cfg.Rules.Rules) &&
		len(cand.ruleFeats) == len(cfg.Rules.Rules) {
		for i := range cfg.Rules.Rules {
			bf := &base.ruleFeats[i]
			if !bf.baseApplicable {
				continue
			}
			rule := &cfg.Rules.Rules[i]
			score := scoreOneRuleFeats(bf, &cand.ruleFeats[i], rule)
			if score == 0 {
				continue
			}
			if rule.MaxScore > 0 && score > rule.MaxScore {
				score = rule.MaxScore
			}
			total += score
		}
		return total
	}
	section := normalizedRootSection(*base)
	for i := range cfg.Rules.Rules {
		rule := &cfg.Rules.Rules[i]
		if !rule.Enabled || !ruleMatchesScope(*rule, section) {
			continue
		}
		score := scoreOneRuleAt(base, cand, rule, i, cfg.Rules.ColorFamilies)
		if score == 0 {
			continue
		}
		if rule.MaxScore > 0 && score > rule.MaxScore {
			score = rule.MaxScore
		}
		total += score
	}
	return total
}

func normalizedRootSection(row ProductRow) string {
	if row.RootSectionNorm != "" {
		return row.RootSectionNorm
	}
	return strings.TrimSpace(strings.ToLower(row.RootSection))
}

func ruleMatchesScope(rule AttributeRuleConfig, section string) bool {
	if len(rule.ScopeSections) == 0 {
		return true
	}
	if section == "" {
		return false
	}
	for _, s := range rule.ScopeSections {
		if strings.TrimSpace(strings.ToLower(s)) == section {
			return true
		}
	}
	return false
}

func scoreOneRule(base, cand ProductRow, rule AttributeRuleConfig, families map[string][]string) int {
	switch rule.MatchType {
	case "color_similarity":
		return scoreColorSimilarityRule(base, cand, rule, families)
	case "exact":
		return scoreExactRule(base, cand, rule)
	case "token_overlap":
		return scoreTokenOverlapRule(base, cand, rule)
	case "set_overlap":
		return scoreSetOverlapRule(base, cand, rule)
	default:
		return 0
	}
}

func scoreOneRuleAt(
	base, cand *ProductRow,
	rule *AttributeRuleConfig,
	ruleIndex int,
	families map[string][]string,
) int {
	if ruleIndex >= 0 &&
		ruleIndex < len(base.ruleFeats) &&
		ruleIndex < len(cand.ruleFeats) {
		return scoreOneRuleFeats(
			&base.ruleFeats[ruleIndex],
			&cand.ruleFeats[ruleIndex],
			rule,
		)
	}
	return scoreOneRule(*base, *cand, *rule, families)
}

func resolveRuleValue(row ProductRow, source, key string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	key = strings.TrimSpace(key)
	switch source {
	case "column":
		return resolveColumnValue(row, key)
	case "attributes":
		return resolveAttributeValue(row, key)
	case "computed":
		if key == "article_prefix" {
			return articlePrefix(row)
		}
	}
	return ""
}

func resolveColumnValue(row ProductRow, key string) string {
	k := strings.TrimSpace(strings.ToLower(key))
	switch k {
	case "color", "normalized_color":
		return candColor(row)
	case "product_type":
		return strings.TrimSpace(row.ProductType)
	case "gender", "gender_or_age_group":
		return strings.TrimSpace(row.Gender)
	case "fit", "fit_or_silhouette":
		return firstNonEmpty(row.Fit, row.Silhouette)
	case "season", "season_or_collection":
		return firstNonEmpty(row.Season, row.Collection)
	case "leaf_category":
		return strings.TrimSpace(row.LeafCategory)
	default:
		return ""
	}
}

func resolveAttributeValue(row ProductRow, key string) string {
	if row.Attributes == "" || key == "" {
		return ""
	}
	attrs := row.AttrMap // прекомпьют из enrichFromAttributes
	if attrs == nil {
		if err := json.Unmarshal([]byte(row.Attributes), &attrs); err != nil {
			return ""
		}
	}
	key = strings.TrimSpace(key)
	keyLower := strings.ToLower(key)
	for k, v := range attrs {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	// Частичное совпадение только для длинных ключей — иначе «Тип» ловит «Тип рукава».
	if len([]rune(keyLower)) < 4 {
		return ""
	}
	for k, v := range attrs {
		kl := strings.ToLower(strings.TrimSpace(k))
		if strings.Contains(kl, keyLower) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// isNeutralAttributeValue — дефолтные значения фида (C): не дают балл за «совпадение».
func isNeutralAttributeValue(v string) bool {
	return isNeutralNormalizedAttributeValue(normalizeColorField(v))
}

func isNeutralNormalizedAttributeValue(n string) bool {
	if n == "" {
		return true
	}
	switch n {
	case "без декора", "без элементов", "без рисунка", "без принта", "без узора",
		"без аппликации", "нет", "не указано", "не указан", "отсутствует":
		return true
	}
	return strings.HasPrefix(n, "без ")
}

func isGenderRule(rule AttributeRuleConfig) bool {
	key := strings.TrimSpace(strings.ToLower(rule.Key))
	return key == "gender" || key == "gender_or_age_group"
}

func ruleExactWeight(rule AttributeRuleConfig) int {
	w := rule.Weight
	if w <= 0 {
		w = rule.MaxScore
	}
	return w
}

var teenGenderCompatSet = map[string]struct{}{
	"девочки": {},
	"унисекс": {},
}

func genderCompatibleFeats(bf, cf ruleFeat) bool {
	if bf.section != "teens" {
		return false
	}
	if bf.boyTeen || cf.boyTeen {
		return false
	}
	if bf.norm == "" || cf.norm == "" || bf.norm == cf.norm {
		return false
	}
	_, oka := teenGenderCompatSet[bf.norm]
	_, okb := teenGenderCompatSet[cf.norm]
	return oka && okb
}

func boyTeenFromRow(row ProductRow) bool {
	if g := normalizeYo(strings.TrimSpace(row.Gender)); g == "мальчики" {
		return true
	}
	leaf := normalizeYo(strings.TrimSpace(row.LeafCategory))
	if strings.Contains(leaf, "мальчик") {
		return true
	}
	parts := row.CategoryParts
	if len(parts) == 0 {
		parts = parseCategoryPath(row.CategoryPath)
	}
	for _, seg := range parts {
		s := normalizeYo(strings.TrimSpace(seg))
		if strings.Contains(s, "мальчик") {
			return true
		}
	}
	return false
}

func normalizeYo(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "ё", "е"))
}

func scoreExactRule(base, cand ProductRow, rule AttributeRuleConfig) int {
	bv := normalizeColorField(resolveRuleValue(base, rule.Source, rule.Key))
	cv := normalizeColorField(resolveRuleValue(cand, rule.Source, rule.Key))
	if bv == "" || cv == "" || isNeutralAttributeValue(bv) || isNeutralAttributeValue(cv) {
		return 0
	}
	if bv == cv {
		return ruleExactWeight(rule)
	}
	if genderCompatibleForSimilar(base, cand, rule, bv, cv) {
		return ruleExactWeight(rule)
	}
	return 0
}

func genderCompatibleForSimilar(base, cand ProductRow, rule AttributeRuleConfig, bv, cv string) bool {
	if !isGenderRule(rule) {
		return false
	}
	bf := ruleFeat{
		norm:    normalizeColorField(bv),
		section: strings.TrimSpace(strings.ToLower(base.RootSection)),
		boyTeen: boyTeenFromRow(base),
	}
	cf := ruleFeat{
		norm:    normalizeColorField(cv),
		boyTeen: boyTeenFromRow(cand),
	}
	return genderCompatibleFeats(bf, cf)
}

func isBoyTeenProduct(row ProductRow) bool {
	return boyTeenFromRow(row)
}

func scoreTokenOverlapRule(base, cand ProductRow, rule AttributeRuleConfig) int {
	bv := resolveRuleValue(base, rule.Source, rule.Key)
	cv := resolveRuleValue(cand, rule.Source, rule.Key)
	if bv == "" || cv == "" || isNeutralAttributeValue(bv) || isNeutralAttributeValue(cv) {
		return 0
	}
	overlap := shadeOverlapScore(bv, cv)
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
}

func scoreSetOverlapRule(base, cand ProductRow, rule AttributeRuleConfig) int {
	bSet := resolveAttributeSet(base, rule.Source, rule.Key)
	cSet := resolveAttributeSet(cand, rule.Source, rule.Key)
	if len(bSet) == 0 || len(cSet) == 0 {
		return 0
	}
	w := rule.Weight
	if w <= 0 {
		w = rule.MaxScore
	}
	return familySetSimilarityScore(bSet, cSet, w)
}

func resolveAttributeSet(row ProductRow, source, key string) map[string]struct{} {
	key = strings.TrimSpace(key)
	if formatted := attributeSetString(row, source, key); formatted != "" {
		return catalog.ParseAttributeSet(formatted)
	}
	return nil
}

func attributeSetString(row ProductRow, source, key string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	key = strings.TrimSpace(key)
	if source != "attributes" {
		return resolveRuleValue(row, source, key)
	}
	if row.Attributes == "" {
		return ""
	}
	attrs := row.AttrMap // прекомпьют из enrichFromAttributes
	if attrs == nil {
		if err := json.Unmarshal([]byte(row.Attributes), &attrs); err != nil {
			return ""
		}
	}
	if g := catalog.SlotSetGroupForKey(key); g != nil {
		if v := strings.TrimSpace(attrs[g.CanonKey]); v != "" {
			return v
		}
		return catalog.FormatAttributeSet(catalog.CollectSlotSetValues(attrs, g.BaseKey))
	}
	keyLower := strings.ToLower(key)
	for k, v := range attrs {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
		if strings.EqualFold(strings.TrimSpace(k), keyLower) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func formatAttributeSetDisplay(row ProductRow, source, key string) string {
	if strings.TrimSpace(strings.ToLower(source)) == "attributes" {
		return attributeSetString(row, source, key)
	}
	return resolveRuleValue(row, source, key)
}

func scoreColorSimilarityRule(base, cand ProductRow, rule AttributeRuleConfig, families map[string][]string) int {
	if len(families) == 0 {
		return 0
	}
	bc := resolveRuleValue(base, rule.Source, rule.Key)
	cc := resolveRuleValue(cand, rule.Source, rule.Key)
	if bc == "" || cc == "" {
		return 0
	}
	cfg := rule.Color
	if cfg == nil {
		def := ColorSimilarityConfig{
			RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
			FamilyScore: 18, ShadeOverlapScore: 6, ConflictPenalty: -20,
		}
		cfg = &def
	}

	bNorm := normalizeColorField(bc)
	cNorm := normalizeColorField(cc)
	score := 0
	bFam, bMulti := colorFamilyInfo(bc, families)
	cFam, cMulti := colorFamilyInfo(cc, families)
	knownFamiliesDisjoint := len(bFam) > 0 && len(cFam) > 0 && !colorFamilySetsOverlap(bFam, cFam)

	if bNorm == cNorm {
		score = cfg.RawExactScore
	} else if knownFamiliesDisjoint {
		return cfg.ConflictPenalty
	} else if bMulti != cMulti {
		return 0
	} else if bMulti && cMulti {
		score = familySetSimilarityScore(bFam, cFam, cfg.FamilyScore)
	} else if primaryColorToken(bc) == primaryColorToken(cc) && cfg.PrimaryExactScore > score {
		score = cfg.PrimaryExactScore
	} else if !knownFamiliesDisjoint {
		bt := colorTokens(bc)
		ct := colorTokens(cc)
		for _, t := range bt {
			for _, u := range ct {
				if t == u && cfg.AliasExactScore > score {
					score = cfg.AliasExactScore
				}
			}
		}
	}

	if !bMulti && !cMulti && len(bFam) > 0 && len(cFam) > 0 {
		for f := range bFam {
			if _, ok := cFam[f]; ok && cfg.FamilyScore > score {
				score = cfg.FamilyScore
			}
		}
	}

	// Общий модификатор («меланж») не делает похожими цвета из разных
	// известных семейств. Shade overlap полезен только внутри одного семейства
	// либо когда хотя бы один цвет словарём ещё не распознан.
	allowShadeOverlap := !knownFamiliesDisjoint
	if shade := shadeOverlapScore(bc, cc); allowShadeOverlap && shade > 0 {
		add := cfg.ShadeOverlapScore * shade / 100
		if add > 0 && score+add > score {
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

func explainAttributeRules(base, cand ProductRow, cfg StrategyConfig) map[string]RuleScoreDetail {
	details := make(map[string]RuleScoreDetail)
	section := strings.TrimSpace(strings.ToLower(base.RootSection))
	for i, rule := range cfg.Rules.Rules {
		if !rule.Enabled {
			continue
		}
		if !ruleMatchesScope(rule, section) {
			details[rule.ID] = RuleScoreDetail{Status: "scope"}
			continue
		}
		bv := resolveRuleValue(base, rule.Source, rule.Key)
		cv := resolveRuleValue(cand, rule.Source, rule.Key)
		if rule.MatchType == "set_overlap" {
			bv = formatAttributeSetDisplay(base, rule.Source, rule.Key)
			cv = formatAttributeSetDisplay(cand, rule.Source, rule.Key)
		}
		score := scoreOneRuleAt(&base, &cand, &rule, i, cfg.Rules.ColorFamilies)
		if rule.MaxScore > 0 && score > rule.MaxScore {
			score = rule.MaxScore
		}
		status := "mismatch"
		switch {
		case bv == "" && cv == "":
			status = "missing_both"
		case bv == "":
			status = "missing_base"
		case cv == "":
			status = "missing_cand"
		case score < 0:
			status = "conflict"
		case score > 0:
			status = "match"
		}
		details[rule.ID] = RuleScoreDetail{
			Score:     score,
			BaseValue: bv,
			CandValue: cv,
			Status:    status,
		}
	}
	return details
}

func explainBaseRuleValues(base ProductRow, cfg StrategyConfig) map[string]RuleScoreDetail {
	details := make(map[string]RuleScoreDetail)
	section := strings.TrimSpace(strings.ToLower(base.RootSection))
	for _, rule := range cfg.Rules.Rules {
		if !rule.Enabled {
			continue
		}
		if !ruleMatchesScope(rule, section) {
			details[rule.ID] = RuleScoreDetail{Status: "scope"}
			continue
		}
		bv := resolveRuleValue(base, rule.Source, rule.Key)
		if rule.MatchType == "set_overlap" {
			bv = formatAttributeSetDisplay(base, rule.Source, rule.Key)
		}
		status := "ok"
		if bv == "" {
			status = "missing_base"
		}
		details[rule.ID] = RuleScoreDetail{
			BaseValue: bv,
			Status:    status,
		}
	}
	return details
}
