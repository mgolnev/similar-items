package similar

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

// PairBreakdown — разбивка скоринга пары.
type PairBreakdown struct {
	Score                 int
	BaseScoreWithoutTitle float64
	ScoreBreakdown
}

func parseCategoryPath(raw string) []string {
	if raw == "" {
		return nil
	}
	var path []string
	if err := json.Unmarshal([]byte(raw), &path); err != nil {
		return nil
	}
	return catalog.NormalizeCategoryPathForSimilarity(path)
}

func normalizeCategorySegment(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "ё", "е"))
}

func categoryPathIdentity(path []string) string {
	parts := make([]string, 0, len(path))
	for _, segment := range path {
		if normalized := normalizeCategorySegment(segment); normalized != "" {
			parts = append(parts, normalized)
		}
	}
	return strings.Join(parts, "\x1f")
}

func categoryLeafIdentity(path []string, leaf string) string {
	if key := categoryPathIdentity(path); key != "" {
		return key
	}
	return normalizeCategorySegment(leaf)
}

func categoryLeafIdentityPtr(row *ProductRow) string {
	return categoryLeafIdentity(categoryPartsPtr(row), row.LeafCategory)
}

func parentCategory(path []string) string {
	if len(path) < 2 {
		return ""
	}
	return path[len(path)-2]
}

func topBranch(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[0]
}

func deepestCommonCategory(a, b []string) (value string, index int, ok bool) {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			break
		}
		value = strings.TrimSpace(a[i])
		index = i
		ok = true
	}
	if !ok || index == 0 {
		return "", 0, false
	}
	return value, index, true
}

// displayArticle — артикул цветоразмера для UI (offer.id / products.sku), напр. GKT024636-1.
func displayArticle(row ProductRow) string {
	if s := strings.TrimSpace(row.SKU); s != "" {
		return s
	}
	if s := strings.TrimSpace(row.GroupID); s != "" {
		return s
	}
	return strings.TrimSpace(row.SalesSKU)
}

// productGroupKey — ключ карточки каталога (как COALESCE(group_id, sales_sku, sku)).
func productGroupKey(row ProductRow) string {
	if key := strings.TrimSpace(row.GroupKey); key != "" {
		return key
	}
	if g := strings.TrimSpace(row.GroupID); g != "" {
		return g
	}
	if s := strings.TrimSpace(row.SalesSKU); s != "" {
		return s
	}
	return strings.TrimSpace(row.SKU)
}

// modelKey коллапсирует цветомодификации одной модели: групповой ключ без
// хвоста "-N" ("BWT001950-1" → "BWT001950"). Ключи без дефиса (или с дефисом
// в нулевой позиции) возвращаются как есть.
func modelKey(row ProductRow) string {
	id := productGroupKey(row)
	if i := strings.LastIndexByte(id, '-'); i > 0 {
		return id[:i]
	}
	return id
}

// filterBaseModelVariants выкидывает цветомодификации базовой (запрошенной)
// модели. Цветомодификации остальных моделей остаются самостоятельными
// кандидатами и могут одновременно попасть в выдачу.
func filterBaseModelVariants(scored []scoredPair, baseModelKey string) []scoredPair {
	out := scored[:0:0]
	for _, p := range scored {
		mk := modelKey(*p.row)
		if mk == baseModelKey {
			continue
		}
		out = append(out, p)
	}
	return out
}

func articleFamilyKey(row ProductRow) string {
	if fam := strings.TrimSpace(row.ArticleFamily); fam != "" {
		return fam
	}
	for _, raw := range []string{row.GroupID, row.SalesSKU, row.SKU} {
		if fam := catalog.ExtractArticleFamily(raw); fam != "" {
			return fam
		}
	}
	return ""
}

func sameArticleFamily(a, b ProductRow) bool {
	fa, fb := articleFamilyKey(a), articleFamilyKey(b)
	return fa != "" && fa == fb
}

func productGroupKeyPtr(row *ProductRow) string {
	if row.GroupKey != "" {
		return row.GroupKey
	}
	return productGroupKey(*row)
}

func articleFamilyKeyPtr(row *ProductRow) string {
	if row.ArticleFamily != "" {
		return row.ArticleFamily
	}
	return articleFamilyKey(*row)
}

func articlePrefixPtr(row *ProductRow) string {
	if row.ArticlePrefix != "" {
		return row.ArticlePrefix
	}
	return articlePrefix(*row)
}

func skipSimilarCandidateRows(base, cand *ProductRow, exclude map[string]struct{}) bool {
	if explicitGenderConflictRows(base, cand) {
		return true
	}
	useIDs := sharePreparedIDSpace(base, cand)
	sameGroup := false
	if useIDs && base.groupKeyID != 0 && cand.groupKeyID != 0 {
		sameGroup = base.groupKeyID == cand.groupKeyID
	} else {
		sameGroup = productGroupKeyPtr(base) == productGroupKeyPtr(cand)
	}
	if sameGroup || cand.SKU == base.SKU {
		return true
	}
	sameFamily := false
	if useIDs && base.articleFamilyID != 0 && cand.articleFamilyID != 0 {
		sameFamily = base.articleFamilyID == cand.articleFamilyID
	} else {
		fa, fb := articleFamilyKeyPtr(base), articleFamilyKeyPtr(cand)
		sameFamily = fa != "" && fa == fb
	}
	if sameFamily {
		return true
	}
	if _, ok := exclude[cand.SKU]; ok {
		return true
	}
	if _, ok := exclude[productGroupKeyPtr(cand)]; ok {
		return true
	}
	return strings.TrimSpace(cand.ImageURL) == ""
}

// explicitGenderConflict блокирует только надёжно распознанную пару разных
// полов. Пустой/унисекс/неизвестный пол не блокирует кандидата.
func explicitGenderConflict(baseGender, candGender string) bool {
	return genderAudience(baseGender)*genderAudience(candGender) == -1
}

func explicitGenderConflictRows(base, cand *ProductRow) bool {
	baseAudience := base.genderAudience
	if !base.genderAudienceReady {
		baseAudience = int8(genderAudience(base.Gender))
	}
	candAudience := cand.genderAudience
	if !cand.genderAudienceReady {
		candAudience = int8(genderAudience(cand.Gender))
	}
	return baseAudience*candAudience == -1
}

func genderAudience(value string) int {
	n := normalizeCategorySegment(value)
	if n == "" || strings.Contains(n, "унисекс") || strings.Contains(n, "unisex") {
		return 0
	}
	female := strings.Contains(n, "женщ") || strings.Contains(n, "девоч") ||
		strings.Contains(n, "девуш") || n == "female" || n == "women" || n == "girls"
	male := strings.Contains(n, "мужчин") || strings.Contains(n, "мальчик") ||
		n == "male" || n == "men" || n == "boys"
	if female == male {
		return 0
	}
	if female {
		return 1
	}
	return -1
}

func skipSimilarCandidate(base, cand ProductRow, exclude map[string]struct{}) bool {
	return skipSimilarCandidateRows(&base, &cand, exclude)
}

func dedupeScoredByGroup(in []scoredPair) []scoredPair {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]scoredPair, 0, len(in))
	for _, s := range in {
		key := productGroupKey(*s.row)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func articlePrefix(row ProductRow) string {
	if prefix := strings.TrimSpace(row.ArticlePrefix); prefix != "" {
		return prefix
	}
	id := strings.TrimSpace(row.SalesSKU)
	if id == "" {
		id = strings.TrimSpace(row.GroupID)
	}
	if id == "" {
		id = row.SKU
	}
	var letters []rune
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			letters = append(letters, r)
			if len(letters) >= 3 {
				break
			}
		}
	}
	if len(letters) < 3 {
		return ""
	}
	return strings.ToUpper(string(letters))
}

func categoryParts(row ProductRow) []string {
	if len(row.CategoryParts) > 0 {
		return row.CategoryParts
	}
	return parseCategoryPath(row.CategoryPath)
}

func categoryPartsPtr(row *ProductRow) []string {
	if len(row.CategoryParts) > 0 {
		return row.CategoryParts
	}
	return parseCategoryPath(row.CategoryPath)
}

func rowParentCategory(row ProductRow, path []string) string {
	if row.ParentCategory != "" {
		return row.ParentCategory
	}
	return parentCategory(path)
}

func rowTopBranch(row ProductRow, path []string) string {
	if row.TopBranch != "" {
		return row.TopBranch
	}
	return topBranch(path)
}

func rowTopBranchPtr(row *ProductRow, path []string) string {
	if row.TopBranch != "" {
		return row.TopBranch
	}
	return topBranch(path)
}

func catalogLeafNorms(row ProductRow) []string {
	if len(row.CatalogLeafNorms) > 0 {
		return row.CatalogLeafNorms
	}
	if len(row.CatalogLeafLabels) == 0 {
		leaf := strings.TrimSpace(strings.ToLower(row.LeafCategory))
		if leaf == "" {
			return nil
		}
		return []string{strings.ReplaceAll(leaf, "ё", "е")}
	}
	norms := make([]string, 0, len(row.CatalogLeafLabels))
	for _, label := range row.CatalogLeafLabels {
		key := strings.TrimSpace(strings.ToLower(label))
		key = strings.ReplaceAll(key, "ё", "е")
		if key != "" {
			norms = append(norms, key)
		}
	}
	return norms
}

func catalogLeafNormsPtr(row *ProductRow) []string {
	if len(row.CatalogLeafNorms) > 0 {
		return row.CatalogLeafNorms
	}
	return catalogLeafNorms(*row)
}

// catalogLeafKeys нужен category audit, где set строится один раз на базовый
// товар. Pair hot path использует allocation-free catalogLeafNorms.
func catalogLeafKeys(row ProductRow) map[string]struct{} {
	norms := catalogLeafNorms(row)
	keys := make(map[string]struct{}, len(norms))
	for _, key := range norms {
		keys[key] = struct{}{}
	}
	return keys
}

// sharedCatalogLeaf — общий leaf из альтернативных веток каталога (не основной leaf_category).
func sharedCatalogLeaf(base, cand ProductRow) (string, bool) {
	return sharedCatalogLeafRows(&base, &cand)
}

func sharedCatalogLeafRows(base, cand *ProductRow) (string, bool) {
	useIDs := sharePreparedIDSpace(base, cand)
	sameDirectLeaf := false
	if useIDs && base.leafCategoryID != 0 && cand.leafCategoryID != 0 {
		sameDirectLeaf = base.leafCategoryID == cand.leafCategoryID
	} else {
		baseLeaf := categoryLeafIdentityPtr(base)
		candLeaf := categoryLeafIdentityPtr(cand)
		sameDirectLeaf = baseLeaf != "" && candLeaf != "" && baseLeaf == candLeaf
	}
	if sameDirectLeaf {
		return "", false
	}
	baseNorms := catalogLeafNormsPtr(base)
	candNorms := catalogLeafNormsPtr(cand)
	if useIDs &&
		len(base.catalogLeafIDs) == len(baseNorms) &&
		len(cand.catalogLeafIDs) == len(candNorms) {
		for candIndex, candID := range cand.catalogLeafIDs {
			for _, baseID := range base.catalogLeafIDs {
				if candID != 0 && candID == baseID {
					if candIndex < len(cand.CatalogLeafLabels) {
						return cand.CatalogLeafLabels[candIndex], true
					}
					return cand.LeafCategory, true
				}
			}
		}
		return "", false
	}
	for candIndex, candKey := range candNorms {
		for _, baseKey := range baseNorms {
			if candKey == baseKey {
				if candIndex < len(cand.CatalogLeafLabels) {
					return cand.CatalogLeafLabels[candIndex], true
				}
				return cand.LeafCategory, true
			}
		}
	}
	return "", false
}

// categoryHit — результат каскада категории для пары.
type categoryHit struct {
	OK     bool
	Score  int
	Kind   string
	Common string
	Depth  int
}

func parentScoreByDepth(depth int, cfg StrategyConfig) (score int, kind string) {
	deep := cfg.Weights.CategoryParentDeep
	mid := cfg.Weights.CategoryParent
	shallow := cfg.Weights.CategoryParentShallow
	if deep <= 0 {
		deep = mid + 20
		if deep >= cfg.Weights.CategoryLeaf {
			deep = cfg.Weights.CategoryLeaf - 1
		}
	}
	if shallow <= 0 {
		shallow = mid / 2
		if shallow < cfg.Weights.CategoryBranch {
			shallow = cfg.Weights.CategoryBranch
		}
	}
	if depth < 1 {
		depth = 1
	}
	switch {
	case depth >= 3:
		return deep, "parent_deep"
	case depth == 2:
		return mid, "parent"
	default:
		return shallow, "parent_shallow"
	}
}

func pickParentHit(bp, cp []string, cfg StrategyConfig) (categoryHit, bool) {
	var best categoryHit
	found := false

	parentB, parentC := parentCategory(bp), parentCategory(cp)
	if parentB != "" && parentC != "" && parentB == parentC {
		depth := len(bp) - 2
		if d := len(cp) - 2; d > depth {
			depth = d
		}
		score, kind := parentScoreByDepth(depth, cfg)
		best = categoryHit{OK: true, Score: score, Kind: kind, Common: parentB, Depth: depth}
		found = true
	}
	if common, idx, ok := deepestCommonCategory(bp, cp); ok && idx > 0 {
		score, kind := parentScoreByDepth(idx, cfg)
		hit := categoryHit{OK: true, Score: score, Kind: kind, Common: common, Depth: idx}
		if !found || hit.Score > best.Score {
			best = hit
			found = true
		}
	}
	return best, found
}

func pickParentHitIDs(base, cand *ProductRow, cfg StrategyConfig) (categoryHit, bool) {
	bp, cp := base.categoryPartIDs, cand.categoryPartIDs
	var best categoryHit
	found := false

	if base.parentCategoryID != 0 && base.parentCategoryID == cand.parentCategoryID {
		depth := len(bp) - 2
		if d := len(cp) - 2; d > depth {
			depth = d
		}
		score, kind := parentScoreByDepth(depth, cfg)
		best = categoryHit{
			OK: true, Score: score, Kind: kind,
			Common: base.ParentCategory, Depth: depth,
		}
		found = true
	}

	max := len(bp)
	if len(cp) < max {
		max = len(cp)
	}
	commonIndex := -1
	for i := 0; i < max; i++ {
		if bp[i] == 0 || bp[i] != cp[i] {
			break
		}
		commonIndex = i
	}
	if commonIndex > 0 {
		score, kind := parentScoreByDepth(commonIndex, cfg)
		common := ""
		if commonIndex < len(base.CategoryParts) {
			common = strings.TrimSpace(base.CategoryParts[commonIndex])
		}
		hit := categoryHit{
			OK: true, Score: score, Kind: kind,
			Common: common, Depth: commonIndex,
		}
		if !found || hit.Score > best.Score {
			best = hit
			found = true
		}
	}
	return best, found
}

func categoryMatchHit(base, cand ProductRow, cfg StrategyConfig) categoryHit {
	return categoryMatchHitRows(&base, &cand, &cfg)
}

func categoryMatchHitRows(base, cand *ProductRow, cfg *StrategyConfig) categoryHit {
	bp := categoryPartsPtr(base)
	cp := categoryPartsPtr(cand)
	leafB := base.LeafCategory
	branchB, branchC := rowTopBranchPtr(base, bp), rowTopBranchPtr(cand, cp)
	useIDs := sharePreparedIDSpace(base, cand)

	var leafMatch bool
	if useIDs && base.leafCategoryID != 0 && cand.leafCategoryID != 0 {
		leafMatch = base.leafCategoryID != 0 && base.leafCategoryID == cand.leafCategoryID
	} else {
		baseLeaf := categoryLeafIdentityPtr(base)
		candLeaf := categoryLeafIdentityPtr(cand)
		leafMatch = baseLeaf != "" && candLeaf != "" && baseLeaf == candLeaf
	}
	if leafMatch {
		return categoryHit{
			OK: true, Score: cfg.Weights.CategoryLeaf, Kind: "leaf",
			Common: leafB, Depth: len(bp) - 1,
		}
	}
	if shared, matched := sharedCatalogLeafRows(base, cand); matched && cfg.Weights.CategorySharedLeaf > 0 {
		_ = shared
		return categoryHit{
			OK: true, Score: cfg.Weights.CategorySharedLeaf, Kind: "shared_leaf",
			Common: shared, Depth: 0,
		}
	}
	var parentHit categoryHit
	var parentMatched bool
	if useIDs {
		parentHit, parentMatched = pickParentHitIDs(base, cand, *cfg)
	} else {
		parentHit, parentMatched = pickParentHit(bp, cp, *cfg)
	}
	if parentMatched {
		return parentHit
	}
	// Teen affinity — fallback для неполных/разъехавшихся teen-путей. Она не
	// должна перебивать более точный общий catalog leaf или общего родителя.
	if hit, ok := teenSubsectionCategoryHitRows(base, cand, cfg); ok {
		return hit
	}
	var branchMatch bool
	if useIDs && base.topBranchID != 0 && cand.topBranchID != 0 {
		branchMatch = base.topBranchID != 0 && base.topBranchID == cand.topBranchID
	} else {
		branchMatch = branchB != "" && branchC != "" && branchB == branchC
	}
	if branchMatch {
		return categoryHit{
			OK: true, Score: cfg.Weights.CategoryBranch, Kind: "branch",
			Common: branchB, Depth: 0,
		}
	}
	return categoryHit{}
}

func teenSubsectionCategoryHit(base, cand ProductRow, cfg StrategyConfig) (categoryHit, bool) {
	return teenSubsectionCategoryHitRows(&base, &cand, &cfg)
}

func teenSubsectionCategoryHitRows(base, cand *ProductRow, cfg *StrategyConfig) (categoryHit, bool) {
	baseRoot := base.RootSectionNorm
	if baseRoot == "" {
		baseRoot = strings.TrimSpace(strings.ToLower(base.RootSection))
	}
	candRoot := cand.RootSectionNorm
	if candRoot == "" {
		candRoot = strings.TrimSpace(strings.ToLower(cand.RootSection))
	}
	if baseRoot != "teens" || candRoot != "teens" {
		return categoryHit{}, false
	}
	baseSub := base.TeenSubsection
	if !base.TeenCategoryReady {
		baseSub = string(catalog.ResolveTeenSubsection(base.CategoryParts, base.LeafCategory, base.Gender, base.CategoriesFeed))
	}
	candSub := cand.TeenSubsection
	if !cand.TeenCategoryReady {
		candSub = string(catalog.ResolveTeenSubsection(cand.CategoryParts, cand.LeafCategory, cand.Gender, cand.CategoriesFeed))
	}
	var subsectionMatch bool
	if sharePreparedIDSpace(base, cand) &&
		base.teenSubsectionID != 0 &&
		cand.teenSubsectionID != 0 {
		subsectionMatch = base.teenSubsectionID != 0 && base.teenSubsectionID == cand.teenSubsectionID
	} else {
		subsectionMatch = baseSub != "" && candSub != "" && baseSub == candSub
	}
	if !subsectionMatch {
		return categoryHit{}, false
	}
	leafB := strings.TrimSpace(base.LeafCategory)
	leafC := strings.TrimSpace(cand.LeafCategory)
	if leafB != "" && leafB == leafC {
		return categoryHit{}, false
	}

	baseClass := base.TeenProductClass
	if !base.TeenCategoryReady {
		baseClass = catalog.TeenProductClass(base.LeafCategory, base.CategoryParts)
	}
	candClass := cand.TeenProductClass
	if !cand.TeenCategoryReady {
		candClass = catalog.TeenProductClass(cand.LeafCategory, cand.CategoryParts)
	}
	var classMatch bool
	if sharePreparedIDSpace(base, cand) &&
		base.teenProductClassID != 0 &&
		cand.teenProductClassID != 0 {
		classMatch = base.teenProductClassID != 0 && base.teenProductClassID == cand.teenProductClassID
	} else {
		classMatch = baseClass != "" && baseClass == candClass
	}
	if classMatch {
		score := cfg.Weights.CategorySharedLeaf
		if score <= 0 {
			score = cfg.Weights.CategoryParent
		}
		common := leafB
		if common == "" {
			common = baseClass
		}
		return categoryHit{
			OK: true, Score: score, Kind: "teen_subsection_class",
			Common: common, Depth: 1,
		}, true
	}

	score := cfg.Weights.CategoryParentShallow
	if score <= 0 {
		score = cfg.Weights.CategoryBranch
	}
	return categoryHit{
		OK: true, Score: score, Kind: "teen_subsection",
		Common: "Подростки", Depth: 1,
	}, true
}

func buildScoreNotes(cat categoryHit, title TitleScoreResult, ruleScores map[string]int, cfg StrategyConfig) []string {
	var notes []string
	if cat.OK {
		switch cat.Kind {
		case "leaf":
			notes = append(notes, "категория: тот же лист «"+cat.Common+"» → +"+itoa(cat.Score))
		case "shared_leaf":
			notes = append(notes, "категория: общий leaf «"+cat.Common+"» → +"+itoa(cat.Score))
		case "parent_deep", "parent", "parent_shallow":
			label := "родитель"
			if cat.Kind == "parent_deep" {
				label = "глубокий родитель"
			} else if cat.Kind == "parent_shallow" {
				label = "мелкий родитель"
			}
			notes = append(notes, "категория: "+label+" «"+cat.Common+"» (глубина "+itoa(cat.Depth)+") → +"+itoa(cat.Score))
		case "branch":
			notes = append(notes, "категория: ветка «"+cat.Common+"» → +"+itoa(cat.Score))
		case "teen_subsection_class":
			notes = append(notes, "категория: подростки, тот же тип «"+cat.Common+"» → +"+itoa(cat.Score))
		case "teen_subsection":
			notes = append(notes, "категория: подростки («"+cat.Common+"») → +"+itoa(cat.Score))
		}
	}
	if title.Gate != "" {
		switch title.Gate {
		case "feature_half_mid_parent":
			notes = append(notes, "название: feature ×½ (parent mid)")
		case "feature_blocked_weak_category":
			notes = append(notes, "название: feature=0 (слабая категория)")
		case "no_category_cap":
			notes = append(notes, "название: cap без категории → "+itoa(cfg.Title.NoCategoryCap))
		}
	} else if title.Score > 0 {
		notes = append(notes, "название: +"+itoa(title.Score))
	}
	for _, rule := range cfg.Rules.Rules {
		if !rule.Enabled {
			continue
		}
		s := ruleScores[rule.ID]
		if s == 0 {
			continue
		}
		label := rule.LabelRu
		if label == "" {
			label = rule.ID
		}
		prefix := ""
		if s > 0 {
			prefix = "+"
		}
		notes = append(notes, label+": "+prefix+itoa(s))
	}
	return notes
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

// ScorePair вычисляет скоринг пары товаров.
func ScorePair(base, cand ProductRow, cfg StrategyConfig) PairBreakdown {
	return scorePairDetailed(base, cand, cfg)
}

func scorePairDetailed(base, cand ProductRow, cfg StrategyConfig) PairBreakdown {
	cat := categoryMatchHit(base, cand, cfg)
	categoryScore := cat.Score
	categoryKind := cat.Kind

	articleScore := 0
	if p := articlePrefix(base); p != "" && p == articlePrefix(cand) {
		articleScore = cfg.Weights.ArticlePrefix
	}

	priceScore := 0
	if base.Price > 0 && cand.Price > 0 {
		rp := float64(cfg.PriceRangePercent) / 100
		lo := base.Price * (1 - rp)
		hi := base.Price * (1 + rp)
		if cand.Price >= lo && cand.Price <= hi {
			priceScore = cfg.Weights.Price
		}
	}

	var titleRes TitleScoreResult
	if base.TitleTokens != nil && cand.TitleTokens != nil {
		titleRes = cappedTitleScoreSetsWithCategory(base.TitleNorm, cand.TitleNorm, base.TitleTokens, cand.TitleTokens, cat, cfg.Title)
	} else {
		titleRes = CappedTitleScoreWithCategory(base.Name, cand.Name, cat, cfg.Title)
	}
	titleScore := titleRes.Score
	rulesScore, ruleScores := scoreAttributeRules(base, cand, cfg)
	displayColor := 0
	if s, ok := ruleScores["color_similarity"]; ok && s != 0 {
		displayColor = s
	}
	attrScore := articleScore + rulesScore
	baseWithoutTitle := float64(categoryScore + attrScore + priceScore)
	raw := baseWithoutTitle + float64(titleScore)
	score := int(raw + 0.5)
	if score < 0 {
		score = 0
	}
	return PairBreakdown{
		Score:                 score,
		BaseScoreWithoutTitle: baseWithoutTitle,
		ScoreBreakdown: ScoreBreakdown{
			Total:            score,
			BaseWithoutTitle: baseWithoutTitle,
			Category:         categoryScore,
			CategoryKind:     categoryKind,
			CategoryCommon:   cat.Common,
			CategoryDepth:    cat.Depth,
			Color:            displayColor,
			RuleScores:       ruleScores,
			ProductType:      0,
			Gender:           0,
			FitOrSilhouette:  0,
			Season:           0,
			ArticlePrefix:    articleScore,
			Price:            priceScore,
			Popularity:       0,
			Title:            titleScore,
			TitleRaw:         titleRes.Raw,
			TitleGate:        titleRes.Gate,
		},
	}
}

type fastPairScore struct {
	score                 int
	baseScoreWithoutTitle float64
}

// scorePairFast зеркалит формулу scorePairDetailed, но не строит
// ScoreBreakdown/RuleScores. Используется только там, где потребителю нужны
// итоговый score и каскадный baseWithoutTitle; admin explain остаётся на
// подробном пути.
func scorePairFast(base, cand *ProductRow, cfg *StrategyConfig) fastPairScore {
	cat := categoryMatchHitRows(base, cand, cfg)
	categoryScore := cat.Score

	articleScore := 0
	prefixMatch := false
	if sharePreparedIDSpace(base, cand) {
		prefixMatch = base.articlePrefixID != 0 && base.articlePrefixID == cand.articlePrefixID
	} else if p := articlePrefixPtr(base); p != "" && p == articlePrefixPtr(cand) {
		prefixMatch = true
	}
	if prefixMatch {
		articleScore = cfg.Weights.ArticlePrefix
	}

	priceScore := 0
	if base.Price > 0 && cand.Price > 0 {
		rp := float64(cfg.PriceRangePercent) / 100
		lo := base.Price * (1 - rp)
		hi := base.Price * (1 + rp)
		if cand.Price >= lo && cand.Price <= hi {
			priceScore = cfg.Weights.Price
		}
	}

	var titleScore int
	if base.TitleTokens != nil && cand.TitleTokens != nil {
		titleScore = cappedTitleScoreSetsWithCategory(
			base.TitleNorm,
			cand.TitleNorm,
			base.TitleTokens,
			cand.TitleTokens,
			cat,
			cfg.Title,
		).Score
	} else {
		titleScore = CappedTitleScoreWithCategory(
			base.Name,
			cand.Name,
			cat,
			cfg.Title,
		).Score
	}

	rulesScore := scoreAttributeRulesTotal(base, cand, cfg)
	baseWithoutTitle := float64(categoryScore + articleScore + rulesScore + priceScore)
	score := int(baseWithoutTitle + float64(titleScore) + 0.5)
	if score < 0 {
		score = 0
	}
	return fastPairScore{
		score:                 score,
		baseScoreWithoutTitle: baseWithoutTitle,
	}
}

// enrichBreakdownWithNotes — только admin preview/explain; не вызывать в RankSimilarCandidates.
func enrichBreakdownWithNotes(b ScoreBreakdown, cfg StrategyConfig) ScoreBreakdown {
	if len(b.ScoreNotes) > 0 {
		return b
	}
	cat := categoryHit{
		OK:     b.Category > 0 || b.CategoryKind != "",
		Score:  b.Category,
		Kind:   b.CategoryKind,
		Common: b.CategoryCommon,
		Depth:  b.CategoryDepth,
	}
	titleRes := TitleScoreResult{Score: b.Title, Raw: b.TitleRaw, Gate: b.TitleGate}
	b.ScoreNotes = buildScoreNotes(cat, titleRes, b.RuleScores, cfg)
	return b
}

func candColor(p ProductRow) string {
	if p.NormalizedColor != "" {
		return p.NormalizedColor
	}
	return strings.ToLower(strings.TrimSpace(p.Color))
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

type scoredPair struct {
	row                   *ProductRow
	score                 int
	baseScoreWithoutTitle float64
	breakdown             *ScoreBreakdown
}

// PickSimilar отбирает похожие по каскаду порогов.
func PickSimilar(
	base ProductRow,
	candidates []ProductRow,
	exclude map[string]struct{},
	limit int,
	cfg StrategyConfig,
) (items []Item, strategy string, fallbackUsed bool) {
	scored := RankSimilarCandidates(base, candidates, cfg)
	return PickSimilarFromRanked(scored, exclude, modelKey(base), limit, cfg)
}

// RankSimilarCandidates считает и сортирует кандидатов без пользовательских
// exclude на быстром production/builder path. Диагностический breakdown здесь
// намеренно пуст; admin preview использует rankSimilarCandidatesDetailed.
func RankSimilarCandidates(base ProductRow, candidates []ProductRow, cfg StrategyConfig) []scoredPair {
	scored := make([]scoredPair, 0, len(candidates))
	for i := range candidates {
		c := &candidates[i]
		if skipSimilarCandidateRows(&base, c, nil) {
			continue
		}
		b := scorePairFast(&base, c, &cfg)
		scored = append(scored, scoredPair{
			row: c, score: b.score, baseScoreWithoutTitle: b.baseScoreWithoutTitle,
		})
	}

	sortScoredPairs(scored)
	return scored
}

func rankSimilarCandidatesDetailed(base ProductRow, candidates []ProductRow, cfg StrategyConfig) []scoredPair {
	scored := make([]scoredPair, 0, len(candidates))
	for i := range candidates {
		c := &candidates[i]
		if skipSimilarCandidate(base, *c, nil) {
			continue
		}
		b := scorePairDetailed(base, *c, cfg)
		breakdown := b.ScoreBreakdown
		scored = append(scored, scoredPair{
			row: c, score: b.Score, baseScoreWithoutTitle: b.BaseScoreWithoutTitle, breakdown: &breakdown,
		})
	}

	sortScoredPairs(scored)
	return scored
}

func sortScoredPairs(scored []scoredPair) {
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].row.SKU < scored[j].row.SKU
	})
}

func filterRankedByExclude(scored []scoredPair, exclude map[string]struct{}) []scoredPair {
	if len(scored) == 0 {
		return scored
	}
	if len(exclude) == 0 {
		return dedupeScoredByGroup(scored)
	}
	out := make([]scoredPair, 0, len(scored))
	for _, s := range scored {
		key := productGroupKey(*s.row)
		if _, ok := exclude[s.row.SKU]; ok {
			continue
		}
		if _, ok := exclude[key]; ok {
			continue
		}
		out = append(out, s)
	}
	return dedupeScoredByGroup(out)
}

func pickFromScored(scored []scoredPair, limit int, cfg StrategyConfig) (pick []scoredPair, strategy string, fallbackUsed bool) {
	th := cfg.Thresholds
	strategy = th.StrictStrategy

	pick = filterScored(scored, th.StrictScoreMin, th.StrictBaseWithoutTitle)
	if len(pick) < limit {
		pick = filterScored(scored, th.ParentScoreMin, th.ParentBaseWithoutTitle)
		fallbackUsed = true
		strategy = th.ParentStrategy
	}
	if len(pick) < limit {
		pick = filterBranch(scored, th.BranchScoreMin)
		fallbackUsed = true
		strategy = th.BranchStrategy
	}
	if len(pick) < limit {
		pick = scored
		fallbackUsed = true
		strategy = th.PopularStrategy
	}

	if len(pick) > limit {
		pick = pick[:limit]
	}
	return pick, strategy, fallbackUsed
}

// PickSimilarFromRanked применяет exclude, фильтрацию базовой модели и каскад порогов
// к заранее рассчитанному рангу. baseModelKey — modelKey запрошенного товара:
// его цветомодификации никогда не рекомендуем обратно. Цветомодификации других
// моделей считаются отдельными карточками и не схлопываются.
func PickSimilarFromRanked(
	ranked []scoredPair,
	exclude map[string]struct{},
	baseModelKey string,
	limit int,
	cfg StrategyConfig,
) (items []Item, strategy string, fallbackUsed bool) {
	scored := filterBaseModelVariants(filterRankedByExclude(ranked, exclude), baseModelKey)
	pick, strategy, fallbackUsed := pickFromScored(scored, limit, cfg)

	items = make([]Item, len(pick))
	for i, p := range pick {
		key := productGroupKey(*p.row)
		items[i] = Item{
			ID:       key,
			SKU:      key,
			Name:     p.row.Name,
			ImageURL: p.row.ImageURL,
			Score:    float64(p.score),
			Price:    p.row.Price,
			OldPrice: p.row.OldPrice,
		}
	}
	return items, strategy, fallbackUsed
}

// PickSimilarExplain — как PickSimilar, но с полной разбивкой баллов для admin preview.
func PickSimilarExplain(
	base ProductRow,
	candidates []ProductRow,
	exclude map[string]struct{},
	limit int,
	cfg StrategyConfig,
) (items []ExplainItem, ranked []ExplainItem, strategy string, fallbackUsed bool) {
	scored := filterRankedByExclude(rankSimilarCandidatesDetailed(base, candidates, cfg), exclude)
	ranked = topRankedExplain(base, scored, 30, cfg)
	pick, strategy, fallbackUsed := pickFromScored(filterBaseModelVariants(scored, modelKey(base)), limit, cfg)
	items = scoredPairsToExplain(base, pick, cfg)
	return items, ranked, strategy, fallbackUsed
}

func scoredPairsToExplain(base ProductRow, pick []scoredPair, cfg StrategyConfig) []ExplainItem {
	items := make([]ExplainItem, len(pick))
	for i, p := range pick {
		key := productGroupKey(*p.row)
		var breakdown ScoreBreakdown
		if p.breakdown != nil {
			breakdown = *p.breakdown
		}
		items[i] = ExplainItem{
			ID: displayArticle(*p.row), SKU: key, Name: p.row.Name, ImageURL: p.row.ImageURL,
			Score: float64(p.score), Breakdown: enrichBreakdownWithNotes(breakdown, cfg),
			RuleDetails: explainAttributeRules(base, *p.row, cfg),
		}
	}
	return items
}

func topRankedExplain(base ProductRow, scored []scoredPair, limit int, cfg StrategyConfig) []ExplainItem {
	if limit <= 0 {
		limit = 25
	}
	if len(scored) > limit {
		scored = scored[:limit]
	}
	return scoredPairsToExplain(base, scored, cfg)
}

func filterScored(in []scoredPair, scoreMin, baseMin int) []scoredPair {
	var out []scoredPair
	for _, s := range in {
		if s.score >= scoreMin && int(s.baseScoreWithoutTitle) >= baseMin {
			out = append(out, s)
		}
	}
	return out
}

func filterBranch(in []scoredPair, scoreMin int) []scoredPair {
	var out []scoredPair
	for _, s := range in {
		if s.score >= scoreMin {
			out = append(out, s)
		}
	}
	return out
}
