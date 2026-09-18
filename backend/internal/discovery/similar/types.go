package similar

// ProductRow — товар для скоринга пары.
type ProductRow struct {
	ID                  string
	SKU                 string
	SalesSKU            string
	GroupID             string
	Name                string
	CategoryPath        string // JSON array
	LeafCategory        string
	Color               string
	NormalizedColor     string
	ProductType         string
	Gender              string
	Fit                 string
	Silhouette          string
	Season              string
	Collection          string
	Price               float64
	OldPrice            float64
	Popularity          float64
	ImageURL            string
	RootSection         string
	Attributes          string
	CategoriesFeed      string   // сырое поле products.categories
	CatalogLeafLabels   []string // листья всех веток «Каталог» из фида
	CatalogLeafNorms    []string // нормализованные CatalogLeafLabels для pair hot path
	CategoryParts       []string
	ParentCategory      string
	TopBranch           string
	GroupKey            string
	ArticleFamily       string
	ArticlePrefix       string
	RootSectionNorm     string
	TeenSubsection      string
	TeenProductClass    string
	TeenCategoryReady   bool
	BoyTeen             bool // подросток-мальчик (gender/teen правила); enrichDerivedFields
	genderAudience      int8 // -1 male, +1 female, 0 unknown/unisex; enrichDerivedFields
	genderAudienceReady bool

	// Прекомпьют (enrichProductRow / precomputeTitle): срезает O(N²)-стоимость
	// скоринга пары — без них каждая пара заново парсила Attributes-JSON и
	// токенизировала тайтлы (~73M раз за прогрев каталога).
	AttrMap     map[string]string // распарсенный Attributes JSON
	TitleNorm   string            // нормализованный тайтл
	TitleTokens *titleTokenSets   // токены тайтла под активный TitleConfig
	// ruleFeats выровнен по индексам cfg.Rules.Rules. Dense slice убирает два
	// string-map lookup на каждое правило каждой пары.
	ruleFeats []ruleFeat

	// Числовые признаки имеют смысл только внутри preparedIDs. Публичные строки
	// сохраняются для API/admin explain, но pair hot path сравнивает uint32.
	preparedIDs        *preparedIDSpace
	groupKeyID         uint32
	modelID            uint32
	articleFamilyID    uint32
	articlePrefixID    uint32
	leafCategoryID     uint32
	parentCategoryID   uint32
	topBranchID        uint32
	teenSubsectionID   uint32
	teenProductClassID uint32
	categoryPartIDs    []uint32
	catalogLeafIDs     []uint32
}

// Item — элемент выдачи similar API.
type Item struct {
	ID       string  `json:"id"`
	SKU      string  `json:"sku"`
	Name     string  `json:"name"`
	ImageURL string  `json:"image_url"`
	Score    float64 `json:"score"`
	Price    float64 `json:"price,omitempty"`
	OldPrice float64 `json:"oldprice,omitempty"`
}

// Result — ответ similar API.
type Result struct {
	Items        []Item `json:"items"`
	Strategy     string `json:"strategy"`
	FallbackUsed bool   `json:"fallback_used"`
}

// ScoreBreakdown — баллы по компонентам пары (admin preview).
type ScoreBreakdown struct {
	Total            int            `json:"total"`
	BaseWithoutTitle float64        `json:"base_without_title"`
	Category         int            `json:"category"`
	CategoryKind     string         `json:"category_kind"`
	CategoryCommon   string         `json:"category_common,omitempty"`
	CategoryDepth    int            `json:"category_depth,omitempty"`
	Color            int            `json:"color"`
	ProductType      int            `json:"product_type"`
	Gender           int            `json:"gender"`
	FitOrSilhouette  int            `json:"fit_or_silhouette"`
	Season           int            `json:"season"`
	ArticlePrefix    int            `json:"article_prefix"`
	Price            int            `json:"price"`
	Popularity       int            `json:"popularity"`
	Title            int            `json:"title"`
	TitleRaw         float64        `json:"title_raw,omitempty"`
	TitleGate        string         `json:"title_gate,omitempty"`
	RuleScores       map[string]int `json:"rule_scores,omitempty"`
	ScoreNotes       []string       `json:"score_notes,omitempty"`
}

// RuleScoreDetail — диагностика одного attribute_rule в admin preview.
type RuleScoreDetail struct {
	Score     int    `json:"score"`
	BaseValue string `json:"base_value,omitempty"`
	CandValue string `json:"cand_value,omitempty"`
	Status    string `json:"status"` // match | conflict | mismatch | missing_base | missing_cand | missing_both | scope
}

// ExplainItem — похожий товар с разбивкой скоринга.
type ExplainItem struct {
	ID          string                     `json:"id"`
	SKU         string                     `json:"sku"`
	Name        string                     `json:"name"`
	ImageURL    string                     `json:"image_url"`
	Score       float64                    `json:"score"`
	Breakdown   ScoreBreakdown             `json:"breakdown"`
	RuleDetails map[string]RuleScoreDetail `json:"rule_details,omitempty"`
}

// BaseProductSummary — базовый товар для preview.
type BaseProductSummary struct {
	ID           string                     `json:"id"`
	SKU          string                     `json:"sku"`
	Name         string                     `json:"name"`
	LeafCategory string                     `json:"leaf_category"`
	ImageURL     string                     `json:"image_url"`
	RuleDetails  map[string]RuleScoreDetail `json:"rule_details,omitempty"`
}

// ExplainResult — ответ admin preview similar.
type ExplainResult struct {
	BaseProduct      BaseProductSummary `json:"base_product"`
	Items            []ExplainItem      `json:"items"`
	RankedCandidates []ExplainItem      `json:"ranked_candidates"`
	Strategy         string             `json:"strategy"`
	FallbackUsed     bool               `json:"fallback_used"`
}

// Params — входные параметры.
type Params struct {
	ProductID         string
	Section           string
	Limit             int
	ExcludeProductIDs []string
	ConfigOverride    *StrategyConfig
}

// BatchParams — вход для поиска похожих по нескольким артикулам.
type BatchParams struct {
	ProductIDs        []string
	Section           string
	Limit             int
	ExcludeProductIDs []string
}

// BatchResultItem — похожие для одного артикула.
type BatchResultItem struct {
	Article      string `json:"article"`
	Items        []Item `json:"items"`
	Strategy     string `json:"strategy,omitempty"`
	FallbackUsed bool   `json:"fallback_used"`
	Error        string `json:"error,omitempty"`
}

// BatchResult — ответ batch similar API.
type BatchResult struct {
	Results []BatchResultItem `json:"results"`
}
