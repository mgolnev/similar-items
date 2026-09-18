package similar

// StrategyConfig — веса и пороги (default из admin-ui-fields-and-defaults.md).
type StrategyConfig struct {
	Weights struct {
		CategoryLeaf          int
		CategorySharedLeaf    int
		CategoryParentDeep    int // общий узел path depth ≥ 3
		CategoryParent        int // depth == 2
		CategoryParentShallow int // depth == 1
		CategoryBranch        int
		Color                 int
		ProductType           int
		GenderOrAgeGroup      int
		FitOrSilhouette       int
		SeasonOrCollection    int
		ArticlePrefix         int
		Price                 int
		PopularityMax         int
	}
	PriceRangePercent int
	Title             TitleConfig
	Thresholds        ThresholdConfig
	PoolLimit         int
	Rules             RulesConfig
	Blocking          BlockingConfig
}

// BlockingConfig — candidate blocking: пре-фильтр кандидатов перед ранжированием.
// Enabled=true: ранжируем только кандидатов, делящих с базой leaf/color/parent/article-prefix.
// MinCandidates: если блок меньше — берём полный пул (widen-fallback).
type BlockingConfig struct {
	Enabled       bool
	MinCandidates int
}

// SynonymGroup — класс эквивалентности словоформ названия (симметричные синонимы).
type SynonymGroup struct {
	ID    string
	Words []string
}

// TitleConfig — сопоставление названий.
type TitleConfig struct {
	Enabled                bool
	ExactScore             int
	JaccardMax             int
	ModelMax               int
	ModelTokenScore        int
	FeatureMax             int
	FeatureTokenScore      int
	SynonymMax             int
	SynonymTokenScore      int
	NoCategoryCap          int
	IgnoreCategoryMismatch bool
	MinTokenLength         int
	Stopwords              []string
	ModelWords             []string
	FeatureWords           []string
	SynonymGroups          []SynonymGroup
	// SynonymIndex — word→groupID, собирается при загрузке стратегии (не из JSON).
	SynonymIndex        map[string]string
	IgnoreModifierWords bool
}

// ThresholdConfig — каскад отбора.
type ThresholdConfig struct {
	StrictScoreMin         int
	StrictBaseWithoutTitle int
	StrictStrategy         string
	ParentScoreMin         int
	ParentBaseWithoutTitle int
	ParentStrategy         string
	BranchScoreMin         int
	BranchStrategy         string
	PopularStrategy        string
}

// DefaultConfig — встроенный дефолт приложения.
func DefaultConfig() StrategyConfig {
	cfg := StrategyConfig{PoolLimit: 0, PriceRangePercent: 25}
	cfg.Blocking = BlockingConfig{Enabled: false, MinCandidates: 60}
	cfg.Weights.CategoryLeaf = 100
	cfg.Weights.CategorySharedLeaf = 80
	cfg.Weights.CategoryParentDeep = 70
	cfg.Weights.CategoryParent = 60
	cfg.Weights.CategoryParentShallow = 30
	cfg.Weights.CategoryBranch = 35
	cfg.Weights.GenderOrAgeGroup = 0
	cfg.Weights.ArticlePrefix = 12
	cfg.Weights.Price = 10

	cfg.Title = TitleConfig{
		Enabled:             true,
		ExactScore:          10,
		JaccardMax:          20,
		ModelMax:            15,
		ModelTokenScore:     5,
		FeatureMax:          15,
		FeatureTokenScore:   5,
		NoCategoryCap:       10,
		MinTokenLength:      3,
		IgnoreModifierWords: true,
		Stopwords:           []string{"clever", "товар", "арт", "для", "из"},
		ModelWords: []string{
			"wide", "straight", "slim", "skinny", "mom", "cargo", "oversize", "basic",
			"flare", "regular", "relaxed", "baggy",
			"палаццо", "клеш", "карго", "оверсайз", "базовый", "прямой", "широкий", "зауженный",
		},
		FeatureWords: []string{
			"принт", "принтом", "принтами",
			"лампас", "лампасы", "лампасами",
			"завязки", "завязками", "завязок",
			"рубчик", "рубчика",
			"вышивка", "вышивкой",
			"капюшон", "капюшоном",
			"полоска", "полоски", "полосками",
			"страз", "стразами",
			"блестки", "блестками",
			"аппликация", "аппликацией",
		},
	}

	cfg.Thresholds.StrictScoreMin = 80
	cfg.Thresholds.StrictBaseWithoutTitle = 50
	cfg.Thresholds.StrictStrategy = "same_category_color_then_parent_fallback"
	cfg.Thresholds.ParentScoreMin = 50
	cfg.Thresholds.ParentBaseWithoutTitle = 25
	cfg.Thresholds.ParentStrategy = "parent_category_fallback"
	cfg.Thresholds.BranchScoreMin = 25
	cfg.Thresholds.BranchStrategy = "branch_fallback"
	cfg.Thresholds.PopularStrategy = "popular_in_section"
	return cfg
}
