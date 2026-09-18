package similar

import "testing"

func TestScorePairLeafCategory(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "1", Name: "Джинсы A", LeafCategory: "Джинсы",
		CategoryPath: `["Женщины","Джинсы"]`, ImageURL: "http://x",
	}
	cand := ProductRow{
		SKU: "2", Name: "Джинсы B", LeafCategory: "Джинсы",
		CategoryPath: `["Женщины","Джинсы"]`, ImageURL: "http://y",
	}
	b := ScorePair(base, cand, cfg)
	if b.Score < cfg.Weights.CategoryLeaf {
		t.Fatalf("score too low: %d", b.Score)
	}
}

func TestPickSimilarDedupesByGroup(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "1", GroupID: "G1", Name: "Base", LeafCategory: "Платья",
		CategoryPath: `["Женщины","Платья"]`, ImageURL: "http://b",
	}
	candidates := []ProductRow{
		{SKU: "2-s", GroupID: "G2", Name: "Брюки S", LeafCategory: "Брюки", CategoryPath: `["Женщины","Брюки"]`, ImageURL: "http://a", Popularity: 50},
		{SKU: "2-m", GroupID: "G2", Name: "Брюки M", LeafCategory: "Брюки", CategoryPath: `["Женщины","Брюки"]`, ImageURL: "http://a", Popularity: 100},
		{SKU: "3", GroupID: "G3", Name: "Платье", LeafCategory: "Платья", CategoryPath: `["Женщины","Платья"]`, ImageURL: "http://c", Popularity: 1},
	}
	items, _, _ := PickSimilar(base, candidates, nil, 5, cfg)
	if len(items) != 2 {
		t.Fatalf("expected 2 unique groups, got %d: %+v", len(items), items)
	}
	if items[0].SKU != "G2" && items[0].SKU != "G3" {
		t.Fatalf("unexpected first sku %s", items[0].SKU)
	}
}

func TestPickSimilarExcludesBaseModelColorVariants(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "v2", GroupID: "GKT026680-2", SalesSKU: "GKT026680-2", Name: "Футболка белая",
		LeafCategory: "Футболки", CategoryPath: `["Женщины","Футболки"]`, ImageURL: "http://b",
	}
	candidates := []ProductRow{
		{SKU: "v1", GroupID: "GKT026680-1", Name: "Футболка розовая", LeafCategory: "Футболки", CategoryPath: `["Женщины","Футболки"]`, ImageURL: "http://a"},
		{SKU: "v3", GroupID: "GKT026680-3", Name: "Футболка зеленая", LeafCategory: "Футболки", CategoryPath: `["Женщины","Футболки"]`, ImageURL: "http://c"},
		{SKU: "other", GroupID: "GKT999999-1", Name: "Другая", LeafCategory: "Футболки", CategoryPath: `["Женщины","Футболки"]`, ImageURL: "http://d"},
	}
	items, _, _ := PickSimilar(base, candidates, nil, 5, cfg)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %+v", len(items), items)
	}
	if items[0].SKU != "GKT999999-1" {
		t.Fatalf("expected other model, got %s", items[0].SKU)
	}
}

func TestPickSimilarKeepsCandidateModelColorVariants(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "base", GroupID: "GSW009030-2", Name: "Базовый джемпер",
		LeafCategory: "Женские джемперы", CategoryPath: `["Женщины","Женские джемперы"]`, ImageURL: "http://base",
	}
	candidates := []ProductRow{
		{SKU: "beige", GroupID: "GSW008762-1", Name: "Бежевый джемпер", LeafCategory: "Женские джемперы", CategoryPath: `["Женщины","Женские джемперы"]`, ImageURL: "http://beige"},
		{SKU: "gray", GroupID: "GSW008762-2", Name: "Серый джемпер", LeafCategory: "Женские джемперы", CategoryPath: `["Женщины","Женские джемперы"]`, ImageURL: "http://gray"},
	}

	items, _, _ := PickSimilar(base, candidates, nil, 5, cfg)
	if len(items) != 2 {
		t.Fatalf("expected both color variants, got %d: %+v", len(items), items)
	}
	got := map[string]bool{}
	for _, item := range items {
		got[item.SKU] = true
	}
	if !got["GSW008762-1"] || !got["GSW008762-2"] {
		t.Fatalf("expected both GSW008762 variants, got %+v", items)
	}
}

func TestPickSimilarCascade(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "1", Name: "Base", LeafCategory: "Платья",
		CategoryPath: `["Женщины","Платья"]`, ImageURL: "http://b",
	}
	candidates := []ProductRow{
		{SKU: "2", Name: "Other", LeafCategory: "Джинсы", CategoryPath: `["Женщины","Джинсы"]`, ImageURL: "http://a", Popularity: 100},
		{SKU: "3", Name: "Match", LeafCategory: "Платья", CategoryPath: `["Женщины","Платья"]`, ImageURL: "http://c", Popularity: 1},
	}
	items, _, _ := PickSimilar(base, candidates, nil, 5, cfg)
	if len(items) == 0 {
		t.Fatal("expected items")
	}
	if items[0].SKU != "3" {
		t.Fatalf("expected sku 3 first, got %s", items[0].SKU)
	}
}

func TestScorePairDetailed_breakdownSums(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "1", Name: "Джинсы slim", LeafCategory: "Джинсы",
		CategoryPath: `["Женщины","Джинсы"]`, ImageURL: "http://x",
		NormalizedColor: "синий", ProductType: "Джинсы", Gender: "Женщины",
		Price: 1000,
	}
	cand := ProductRow{
		SKU: "2", Name: "Джинсы slim fit", LeafCategory: "Джинсы",
		CategoryPath: `["Женщины","Джинсы"]`, ImageURL: "http://y",
		NormalizedColor: "синий", ProductType: "Джинсы", Gender: "Женщины",
		Price: 1100, Popularity: 3,
	}
	b := scorePairDetailed(base, cand, cfg)
	if int(b.BaseScoreWithoutTitle)+b.Title != b.Score {
		t.Fatalf("score mismatch: base=%v title=%d total=%d",
			b.BaseScoreWithoutTitle, b.Title, b.Score)
	}
	if b.CategoryKind != "leaf" {
		t.Fatalf("category kind: %q", b.CategoryKind)
	}
}

func TestScorePair_IgnoresRemovedBuiltinWeights(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Weights.Color = 200
	cfg.Weights.ProductType = 200
	cfg.Weights.FitOrSilhouette = 200
	cfg.Weights.SeasonOrCollection = 200
	cfg.Weights.PopularityMax = 200
	cfg.Weights.GenderOrAgeGroup = 200
	cfg.Rules.Rules = nil

	base := ProductRow{
		SKU: "1", Name: "Base", ImageURL: "http://x",
		NormalizedColor: "синий", ProductType: "Джинсы",
		Fit: "slim", Season: "лето",
	}
	cand := ProductRow{
		SKU: "2", Name: "Other", ImageURL: "http://y",
		NormalizedColor: "синий", ProductType: "Джинсы",
		Fit: "slim", Season: "лето", Popularity: 1000,
	}

	b := scorePairDetailed(base, cand, cfg)
	if b.Color != 0 || b.ProductType != 0 || b.FitOrSilhouette != 0 || b.Season != 0 || b.Popularity != 0 || b.Gender != 0 {
		t.Fatalf("removed built-in weights must not score: %+v", b.ScoreBreakdown)
	}
}

func TestScorePair_sharedCatalogLeaf(t *testing.T) {
	cfg := DefaultConfig()
	// Две ветки «Каталог» как у GKT027652 в фиде.
	multiBranchFeed := "2:Коллекции | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"30:Женские базовые футболки | 5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | " +
		"3365:Женские укороченные футболки"

	base := ProductRow{
		SKU: "base", Name: "База", ImageURL: "http://b",
		LeafCategory:   "Женские укороченные футболки",
		CategoryPath:   `["Женщины","Женские футболки и лонгсливы","Женские укороченные футболки"]`,
		CategoriesFeed: multiBranchFeed,
	}
	cand := ProductRow{
		SKU: "cand", Name: "Кандидат", ImageURL: "http://c",
		LeafCategory:   "Женские базовые футболки",
		CategoryPath:   `["Женщины","Женские футболки и лонгсливы","Женские базовые футболки"]`,
		CategoriesFeed: "5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | 30:Женские базовые футболки",
	}
	enrichProductRow(&base)
	enrichProductRow(&cand)

	b := scorePairDetailed(base, cand, cfg)
	if b.CategoryKind != "shared_leaf" {
		t.Fatalf("category kind: got %q want shared_leaf (labels base=%v cand=%v)",
			b.CategoryKind, base.CatalogLeafLabels, cand.CatalogLeafLabels)
	}
	if b.Category != cfg.Weights.CategorySharedLeaf {
		t.Fatalf("category score: got %d want %d", b.Category, cfg.Weights.CategorySharedLeaf)
	}
}

func TestScorePair_teenSharedCatalogLeafPrecedesTeenFallback(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "GAC026313-2", Name: "Чёрные спортивные джоггеры", Gender: "Девочки", ImageURL: "http://base",
		CategoriesFeed: "5:Каталог | 1628:Подростки | 322:Девочки-подростки | 343:Спортивная одежда для девочек-подростков | 345:Спортивные брюки для девочек-подростков | " +
			"5:Каталог | 1628:Подростки | 322:Девочки-подростки | 378:Школьная одежда для девочек-подростков | 4164:Чёрная форма",
	}
	cand := ProductRow{
		SKU: "GAC025534-1", Name: "Чёрные спортивные брюки", Gender: "Девочки", ImageURL: "http://cand",
		CategoriesFeed: "5:Каталог | 1628:Подростки | 322:Девочки-подростки | 343:Спортивная одежда для девочек-подростков | 345:Спортивные брюки для девочек-подростков | " +
			"5:Каталог | 1628:Подростки | 322:Девочки-подростки | 328:Брюки для девочек-подростков | 1381:Широкие брюки для девочек-подростков",
	}
	enrichProductRow(&base)
	enrichProductRow(&cand)

	hit := categoryMatchHit(base, cand, cfg)
	if hit.Kind != "shared_leaf" {
		t.Fatalf("got %+v want shared_leaf before teen_subsection", hit)
	}
	if hit.Score != cfg.Weights.CategorySharedLeaf || hit.Common != "Спортивные брюки для девочек-подростков" {
		t.Fatalf("unexpected shared leaf hit: %+v", hit)
	}
}

func TestScorePair_parallelCatalogLeafSurvivesDeeperSiblingBranch(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "GRT000918-1", Name: "Тёмно-коричневые свободные спортивные брюки", Gender: "Женщины", ImageURL: "http://base",
		CategoriesFeed: "5:Каталог | 9:Женщины | 11:Женские брюки | " +
			"5:Каталог | 9:Женщины | 26:Женская спортивная одежда | 25:Женские спортивные брюки | " +
			"5:Каталог | 9:Женщины | 11:Женские брюки | 3180:Женские свободные брюки | 2319:Женские свободные брюки",
	}
	cand := ProductRow{
		SKU: "GAC026347-2", Name: "Тёмно-коричневые широкие брюки", Gender: "Женщины", ImageURL: "http://cand",
		CategoriesFeed: "5:Каталог | 9:Женщины | 26:Женская спортивная одежда | 25:Женские спортивные брюки | " +
			"5:Каталог | 9:Женщины | 11:Женские брюки | 2478:Женские широкие брюки",
	}
	weak := ProductRow{
		SKU: "GRT000804-2", Name: "Тёмно-коричневые свободные спортивные брюки", Gender: "Женщины", ImageURL: "http://weak",
		CategoriesFeed: "5:Каталог | 9:Женщины | 11:Женские брюки | " +
			"5:Каталог | 9:Женщины | 33:Женские легинсы",
	}
	enrichProductRow(&base)
	enrichProductRow(&cand)
	enrichProductRow(&weak)

	hit := categoryMatchHit(base, cand, cfg)
	if hit.Kind != "shared_leaf" || hit.Common != "Женские спортивные брюки" {
		t.Fatalf("sport branch must survive deeper sibling: %+v (base leaves=%v)", hit, base.CatalogLeafLabels)
	}
	if hit.Score != cfg.Weights.CategorySharedLeaf {
		t.Fatalf("category score: got %d want %d", hit.Score, cfg.Weights.CategorySharedLeaf)
	}
	if weakHit := categoryMatchHit(base, weak, cfg); weakHit.Kind == "shared_leaf" {
		t.Fatalf("a broad parent path must not become shared leaf: %+v", weakHit)
	}
}

func TestScorePair_sharedCatalogLeaf_disabledWhenWeightZero(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Weights.CategorySharedLeaf = 0
	base := ProductRow{
		SKU: "base", ImageURL: "http://b",
		LeafCategory:   "Женские укороченные футболки",
		CategoriesFeed: "5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | 3365:Женские укороченные футболки",
	}
	cand := ProductRow{
		SKU: "cand", ImageURL: "http://c",
		LeafCategory:   "Женские базовые футболки",
		CategoriesFeed: "5:Каталог | 9:Женщины | 28:Женские футболки и лонгсливы | 30:Женские базовые футболки",
	}
	enrichProductRow(&base)
	enrichProductRow(&cand)
	b := scorePairDetailed(base, cand, cfg)
	if b.CategoryKind == "shared_leaf" {
		t.Fatalf("expected no shared_leaf when weight is 0, got %q", b.CategoryKind)
	}
}

func TestCategoryMatch_depthAwareParent(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	windbreaker := ProductRow{
		LeafCategory: "Ветровки для девочек",
		CategoryPath: `["Дети","Девочки","Верхняя одежда для девочек","Куртки для девочек","Ветровки для девочек"]`,
	}
	jacket := ProductRow{
		LeafCategory: "Куртки-бомбер для девочек",
		CategoryPath: `["Дети","Девочки","Верхняя одежда для девочек","Куртки для девочек","Куртки-бомбер для девочек"]`,
	}
	skirt := ProductRow{
		LeafCategory: "Джинсовые юбки для девочек",
		CategoryPath: `["Дети","Девочки","Юбки для девочек","Джинсовые юбки для девочек"]`,
	}
	deep := categoryMatchHit(windbreaker, jacket, cfg)
	if !deep.OK || deep.Kind != "parent_deep" {
		t.Fatalf("jacket: got %+v want parent_deep", deep)
	}
	shallow := categoryMatchHit(windbreaker, skirt, cfg)
	if !shallow.OK || shallow.Kind != "parent_shallow" {
		t.Fatalf("skirt: got %+v want parent_shallow", shallow)
	}
	if deep.Score <= shallow.Score {
		t.Fatalf("deep %d should beat shallow %d", deep.Score, shallow.Score)
	}
}

func TestPickSimilarExplain_includesBreakdown(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "1", GroupID: "G1", Name: "Base", LeafCategory: "Платья",
		CategoryPath: `["Женщины","Платья"]`, ImageURL: "http://b",
	}
	candidates := []ProductRow{
		{SKU: "3", GroupID: "G3", Name: "Match", LeafCategory: "Платья",
			CategoryPath: `["Женщины","Платья"]`, ImageURL: "http://c", Popularity: 1},
	}
	items, ranked, _, _ := PickSimilarExplain(base, candidates, nil, 5, cfg)
	if len(ranked) == 0 {
		t.Fatal("expected ranked candidates")
	}
	if len(items) != 1 {
		t.Fatalf("items: %d", len(items))
	}
	if items[0].Breakdown.Category == 0 {
		t.Fatal("expected category score in breakdown")
	}
	if len(items[0].Breakdown.ScoreNotes) == 0 {
		t.Fatal("expected score notes in explain breakdown")
	}
}

func TestTeenSubsectionCategoryHit_girlJeansAndPants(t *testing.T) {
	cfg := DefaultConfig()
	girlPants := ProductRow{
		SKU: "GPT011623-1", Name: "Чёрные брюки клёш",
		RootSection: "teens", Gender: "Девочки",
		LeafCategory: "Брюки для девушек",
		CategoryPath: `["Брюки для девушек"]`, ImageURL: "http://a",
		CategoriesFeed: `5:Каталог | 2715:Брюки для девушек`,
	}
	girlJeans := ProductRow{
		SKU: "GJN038423-1", Name: "Свободные джинсы",
		RootSection: "teens", Gender: "Девочки",
		LeafCategory: "Джинсы для девушек",
		CategoryPath: `["Джинсы для девушек"]`, ImageURL: "http://b",
		CategoriesFeed: `5:Каталог | 2714:Джинсы для девушек`,
	}
	enrichProductRow(&girlPants)
	enrichProductRow(&girlJeans)

	if hit, ok := teenSubsectionCategoryHit(girlPants, girlJeans, cfg); !ok {
		t.Fatal("expected teen affinity between girl pants and girl jeans")
	} else if hit.Kind != "teen_subsection" {
		t.Fatalf("cross-class kind: %s", hit.Kind)
	}

	girlMatch := ProductRow{
		SKU: "GAC026064-1", Name: "Чёрные спортивные брюки клёш",
		RootSection: "teens", Gender: "Унисекс",
		LeafCategory:   "Спортивные брюки для девушек",
		CategoryPath:   `["Спортивная одежда для девушек","Спортивные брюки для девушек"]`,
		ImageURL:       "http://c",
		CategoriesFeed: `5:Каталог | 2728:Спортивные брюки для девушек`,
	}
	enrichProductRow(&girlMatch)
	if hit, ok := teenSubsectionCategoryHit(girlPants, girlMatch, cfg); !ok {
		t.Fatal("expected teen girl pants affinity between girl products")
	} else if hit.Kind != "teen_subsection_class" {
		t.Fatalf("kind: %s", hit.Kind)
	}

	boy := ProductRow{
		SKU: "BSU000511-1", Name: "Серые школьные брюки",
		RootSection: "teens", Gender: "Мальчики",
		LeafCategory: "Свободные брюки для мальчиков-подростков",
		CategoryPath: `["Подростки","Мальчики-подростки","Брюки для мальчиков-подростков","Свободные брюки для мальчиков-подростков"]`,
		ImageURL:     "http://d",
	}
	enrichProductRow(&boy)
	if hit, ok := teenSubsectionCategoryHit(girlPants, boy, cfg); ok {
		t.Fatalf("boy pants must not get teen affinity: %+v", hit)
	}
}
