package similar

import (
	"regexp"
	"strings"
)

var (
	multiSpace   = regexp.MustCompile(`\s+`)
	nonLetterNum = regexp.MustCompile(`[^\p{L}\p{N}]+`)
)

func normalizeTitleText(input string) string {
	s := strings.TrimSpace(strings.ToLower(input))
	s = strings.ReplaceAll(s, "ё", "е")
	s = multiSpace.ReplaceAllString(nonLetterNum.ReplaceAllString(s, " "), " ")
	return strings.TrimSpace(s)
}

// DefaultTitleSynonymGroups — стартовый набор для встроенного дефолта.
func DefaultTitleSynonymGroups() []SynonymGroup {
	return []SynonymGroup{
		{ID: "outerwear_light", Words: []string{"ветровка", "куртка"}},
		{ID: "fit_loose", Words: []string{"свободные", "широкие", "широкий"}},
		{ID: "jacket_blazer", Words: []string{"жакет", "пиджак"}},
	}
}

// BuildSynonymIndex строит word→groupID. Повтор слова в другой группе игнорируется (первое побеждает).
func BuildSynonymIndex(groups []SynonymGroup) map[string]string {
	if len(groups) == 0 {
		return nil
	}
	out := make(map[string]string, len(groups)*4)
	for _, g := range groups {
		id := strings.TrimSpace(g.ID)
		if id == "" {
			id = strings.Join(g.Words, "|")
		}
		if id == "" {
			continue
		}
		for _, w := range g.Words {
			key := normalizeTitleText(w)
			if key == "" {
				continue
			}
			if _, exists := out[key]; exists {
				continue
			}
			out[key] = id
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type titleTokenSets struct {
	normal        map[string]struct{}
	model         map[string]struct{}
	feature       map[string]struct{}
	synonymGroups map[string]map[string]struct{}
}

// titleDicts — словари конфига (стоп-/модельные/фичевые слова), собранные один
// раз на проход: раньше они пересобирались на КАЖДУЮ пару скоринга.
type titleDicts struct {
	stops, models, features map[string]struct{}
}

func newTitleDicts(cfg TitleConfig) titleDicts {
	d := titleDicts{
		stops:    make(map[string]struct{}, len(cfg.Stopwords)),
		models:   make(map[string]struct{}, len(cfg.ModelWords)),
		features: make(map[string]struct{}, len(cfg.FeatureWords)),
	}
	for _, w := range cfg.Stopwords {
		d.stops[w] = struct{}{}
	}
	for _, w := range cfg.ModelWords {
		d.models[w] = struct{}{}
	}
	for _, w := range cfg.FeatureWords {
		d.features[w] = struct{}{}
	}
	return d
}

func tokenSets(name string, cfg TitleConfig) titleTokenSets {
	return tokenSetsWithDicts(normalizeTitleText(name), newTitleDicts(cfg), cfg.MinTokenLength)
}

// tokenSetsWithDicts токенизирует УЖЕ нормализованный тайтл готовыми словарями.
func tokenSetsWithDicts(normName string, d titleDicts, minTokenLength int) titleTokenSets {
	out := titleTokenSets{
		normal:  make(map[string]struct{}),
		model:   make(map[string]struct{}),
		feature: make(map[string]struct{}),
	}
	for _, t := range strings.Fields(normName) {
		if _, ok := d.stops[t]; ok {
			continue
		}
		if _, ok := d.models[t]; ok {
			out.model[t] = struct{}{}
			continue
		}
		if _, ok := d.features[t]; ok {
			out.feature[t] = struct{}{}
			continue
		}
		if len([]rune(t)) >= minTokenLength {
			out.normal[t] = struct{}{}
		}
	}
	return out
}

// precomputeTitle кладёт нормализованный тайтл и его токены на строку товара —
// скоринг пары дальше работает только пересечениями множеств.
func precomputeTitle(row *ProductRow, d titleDicts, cfg TitleConfig) {
	row.TitleNorm = normalizeTitleText(row.Name)
	ts := tokenSetsWithDicts(row.TitleNorm, d, cfg.MinTokenLength)
	if len(cfg.SynonymIndex) > 0 && cfg.SynonymMax > 0 {
		// Группы и исходные словоформы зависят только от одного товара и
		// стратегии. Строим их один раз на товар, а не для каждой пары.
		ts.synonymGroups = collectGroupSurfaces(ts, cfg.SynonymIndex)
	}
	row.TitleTokens = &ts
}

func tokenIntersectionSize(a, b map[string]struct{}) int {
	n := 0
	for k := range a {
		if _, ok := b[k]; ok {
			n++
		}
	}
	return n
}

// titleTokenOverlap — схожесть наборов токенов названия.
// При ignoreOneSided: |base∩cand|/|base| — лишние слова только у кандидата не штрафуют.
func titleTokenOverlap(base, cand map[string]struct{}, ignoreOneSided bool) float64 {
	if len(base) == 0 && len(cand) == 0 {
		return 0
	}
	inter := tokenIntersectionSize(base, cand)
	if ignoreOneSided {
		if len(base) == 0 {
			return 0
		}
		return float64(inter) / float64(len(base))
	}
	union := len(base) + len(cand) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func matchedDictionaryScore(base, cand map[string]struct{}, perToken, max int) float64 {
	if perToken <= 0 {
		perToken = 5
	}
	score := 0.0
	for t := range base {
		if _, ok := cand[t]; ok {
			score += float64(perToken)
		}
	}
	if max > 0 && score > float64(max) {
		score = float64(max)
	}
	return score
}

func collectGroupSurfaces(sets titleTokenSets, index map[string]string) map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{})
	add := func(token string) {
		gid, ok := index[token]
		if !ok {
			return
		}
		m := out[gid]
		if m == nil {
			m = make(map[string]struct{}, 2)
			out[gid] = m
		}
		m[token] = struct{}{}
	}
	for t := range sets.normal {
		add(t)
	}
	for t := range sets.model {
		add(t)
	}
	for t := range sets.feature {
		add(t)
	}
	return out
}

func matchedSynonymScore(base, cand titleTokenSets, index map[string]string, perToken, max int) float64 {
	return matchedSynonymScorePrepared(&base, &cand, index, perToken, max)
}

func matchedSynonymScorePrepared(base, cand *titleTokenSets, index map[string]string, perToken, max int) float64 {
	if len(index) == 0 || max <= 0 {
		return 0
	}
	if perToken <= 0 {
		perToken = 5
	}
	baseGroups := base.synonymGroups
	if baseGroups == nil {
		baseGroups = collectGroupSurfaces(*base, index)
	}
	if len(baseGroups) == 0 {
		return 0
	}
	candGroups := cand.synonymGroups
	if candGroups == nil {
		candGroups = collectGroupSurfaces(*cand, index)
	}
	if len(candGroups) == 0 {
		return 0
	}
	score := 0.0
	for gid, baseWords := range baseGroups {
		candWords, ok := candGroups[gid]
		if !ok {
			continue
		}
		exact := false
		for w := range baseWords {
			if _, ok := candWords[w]; ok {
				exact = true
				break
			}
		}
		if exact {
			continue
		}
		score += float64(perToken)
		if score >= float64(max) {
			return float64(max)
		}
	}
	return score
}

// TitleScoreResult — итог title-скоринга с gate (D).
type TitleScoreResult struct {
	Score int
	Raw   float64
	Gate  string
}

func applyTitleFeatureGate(featureScore float64, cat categoryHit) (float64, string) {
	if featureScore <= 0 {
		return 0, ""
	}
	if !cat.OK {
		return 0, "feature_blocked_weak_category"
	}
	switch cat.Kind {
	case "leaf", "shared_leaf", "parent_deep":
		return featureScore, ""
	case "parent":
		return featureScore * 0.5, "feature_half_mid_parent"
	default:
		return 0, "feature_blocked_weak_category"
	}
}

func scoreTitlePair(baseNorm, candNorm string, base, cand *titleTokenSets, cat categoryHit, cfg TitleConfig) TitleScoreResult {
	out := TitleScoreResult{}
	if !cfg.Enabled {
		return out
	}
	if baseNorm == "" || candNorm == "" {
		return out
	}
	score := 0.0
	if baseNorm == candNorm {
		score += float64(cfg.ExactScore)
	}
	score += titleTokenOverlap(base.normal, cand.normal, cfg.IgnoreModifierWords) * float64(cfg.JaccardMax)
	score += matchedDictionaryScore(base.model, cand.model, cfg.ModelTokenScore, cfg.ModelMax)

	featureRaw := matchedDictionaryScore(base.feature, cand.feature, cfg.FeatureTokenScore, cfg.FeatureMax)
	featureScore := featureRaw
	gate := ""
	if !cfg.IgnoreCategoryMismatch {
		featureScore, gate = applyTitleFeatureGate(featureRaw, cat)
	}
	score += featureScore

	score += matchedSynonymScorePrepared(base, cand, cfg.SynonymIndex, cfg.SynonymTokenScore, cfg.SynonymMax)
	cap := float64(cfg.JaccardMax + cfg.ModelMax + cfg.FeatureMax + cfg.ExactScore + cfg.SynonymMax)
	if !cfg.IgnoreCategoryMismatch && !cat.OK && score > float64(cfg.NoCategoryCap) {
		score = float64(cfg.NoCategoryCap)
		if gate == "" {
			gate = "no_category_cap"
		}
	}
	if score > cap {
		score = cap
	}
	out.Raw = score
	out.Score = int(score + 0.5)
	out.Gate = gate
	return out
}

// CappedTitleScoreWithCategory — сопоставление названий с учётом близости категории (D).
func CappedTitleScoreWithCategory(baseName, candName string, cat categoryHit, cfg TitleConfig) TitleScoreResult {
	if !cfg.Enabled {
		return TitleScoreResult{}
	}
	d := newTitleDicts(cfg)
	bn := normalizeTitleText(baseName)
	cn := normalizeTitleText(candName)
	base := tokenSetsWithDicts(bn, d, cfg.MinTokenLength)
	cand := tokenSetsWithDicts(cn, d, cfg.MinTokenLength)
	return scoreTitlePair(bn, cn, &base, &cand, cat, cfg)
}

// cappedTitleScoreSetsWithCategory — горячий путь с прекомпьют-токенами.
func cappedTitleScoreSetsWithCategory(baseNorm, candNorm string, base, cand *titleTokenSets, cat categoryHit, cfg TitleConfig) TitleScoreResult {
	return scoreTitlePair(baseNorm, candNorm, base, cand, cat, cfg)
}

// CappedTitleScore — legacy API (tests): hasCategoryMatch → leaf/deep path.
func CappedTitleScore(baseName, candName string, hasCategoryMatch bool, cfg TitleConfig) float64 {
	cat := categoryHit{}
	if hasCategoryMatch {
		cat = categoryHit{OK: true, Kind: "leaf", Depth: 3}
	}
	return CappedTitleScoreWithCategory(baseName, candName, cat, cfg).Raw
}

// cappedTitleScoreSets — legacy precompute path с bool category match.
func cappedTitleScoreSets(baseNorm, candNorm string, base, cand *titleTokenSets, hasCategoryMatch bool, cfg TitleConfig) float64 {
	cat := categoryHit{}
	if hasCategoryMatch {
		cat = categoryHit{OK: true, Kind: "leaf", Depth: 3}
	}
	return cappedTitleScoreSetsWithCategory(baseNorm, candNorm, base, cand, cat, cfg).Raw
}
