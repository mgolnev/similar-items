package strategy

// SynonymGroup — группа равнозначных словоформ для title-скоринга.
type SynonymGroup struct {
	ID    string   `json:"id"`
	Words []string `json:"words"`
}

// Document — JSON стратегии similar + search (admin-ui-fields-and-defaults.md).
type Document struct {
	SchemaVersion int  `json:"schema_version"`
	Enabled       bool `json:"enabled"`

	// Blocking — опциональный override candidate-blocking. Отсутствует → дефолт (выключен).
	Blocking *struct {
		Enabled       bool `json:"enabled"`
		MinCandidates int  `json:"min_candidates"`
	} `json:"blocking,omitempty"`

	Weights struct {
		CategoryLeaf          int `json:"category_leaf"`
		CategorySharedLeaf    int `json:"category_shared_leaf"`
		CategoryParentDeep    int `json:"category_parent_deep"`
		CategoryParent        int `json:"category_parent"`
		CategoryParentShallow int `json:"category_parent_shallow"`
		CategoryBranch        int `json:"category_branch"`
		Color                 int `json:"color"`
		ProductType           int `json:"product_type"`
		GenderOrAgeGroup      int `json:"gender_or_age_group"`
		FitOrSilhouette       int `json:"fit_or_silhouette"`
		SeasonOrCollection    int `json:"season_or_collection"`
		ArticlePrefix         int `json:"article_prefix"`
		Price                 int `json:"price"`
		PopularityMax         int `json:"popularity_max"`
	} `json:"weights"`

	Price struct {
		RangePercent int `json:"range_percent"`
	} `json:"price"`

	Title struct {
		Enabled                    bool           `json:"enabled"`
		MaxScore                   int            `json:"max_score"`
		ExactScore                 int            `json:"exact_score"`
		JaccardMaxScore            int            `json:"jaccard_max_score"`
		ModelTokenScore            int            `json:"model_token_score"`
		ModelMaxScore              int            `json:"model_max_score"`
		FeatureTokenScore          int            `json:"feature_token_score"`
		FeatureMaxScore            int            `json:"feature_max_score"`
		SynonymTokenScore          int            `json:"synonym_token_score"`
		SynonymMaxScore            int            `json:"synonym_max_score"`
		NoCategoryCap              int            `json:"no_category_cap"`
		IgnoreCategoryMismatch     bool           `json:"ignore_category_mismatch"`
		MinTokenLength             int            `json:"min_token_length"`
		Stopwords                  []string       `json:"stopwords"`
		ModelWords                 []string       `json:"model_words"`
		FeatureWords               []string       `json:"feature_words"`
		SynonymGroups              []SynonymGroup `json:"synonym_groups"`
		IgnoreModifierWordsEnabled bool           `json:"ignore_modifier_words_enabled"`
	} `json:"title"`

	Thresholds struct {
		Strict struct {
			ScoreMin            int    `json:"score_min"`
			BaseWithoutTitleMin int    `json:"base_without_title_min"`
			Strategy            string `json:"strategy"`
		} `json:"strict"`
		Parent struct {
			ScoreMin            int    `json:"score_min"`
			BaseWithoutTitleMin int    `json:"base_without_title_min"`
			Strategy            string `json:"strategy"`
		} `json:"parent"`
		Branch struct {
			ScoreMin int    `json:"score_min"`
			Strategy string `json:"strategy"`
		} `json:"branch"`
		Popular struct {
			Enabled  bool   `json:"enabled"`
			Strategy string `json:"strategy"`
		} `json:"popular"`
	} `json:"thresholds"`

	AttributeRules        []AttributeRule     `json:"attribute_rules"`
	ColorFamilyDictionary map[string][]string `json:"color_family_dictionary"`
	ColorFamilyPalette    map[string]string   `json:"color_family_palette"`
	SearchSynonyms        map[string][]string `json:"search_synonyms"`
	Search                SearchConfig        `json:"search"`
	UpdatedReason         string              `json:"updated_reason"`
}

// SearchConfig — секция настроек поиска (отдельно от similar):
// pins/boosts/popularity_weight/словарь типов одежды.
type SearchConfig struct {
	// PopularityWeight — вес популярности в финальном скоре (0..1).
	// 0 — только релевантность; 1 — популярность равна релевантности.
	PopularityWeight float64       `json:"popularity_weight"`
	Pins             []SearchPin   `json:"pins"`
	Boosts           []SearchBoost `json:"boosts"`
	// ProductTypeDictionary — словарь типов одежды (стемы), которые intent должен распознавать как product.
	// Если непустой — расширяет хардкодные productStems из internal/discovery/search/token.go.
	ProductTypeDictionary []string `json:"product_type_dictionary"`
}

// SearchPin — пиннинг товара на конкретную позицию для exact-query match.
type SearchPin struct {
	Query    string `json:"query"`
	SKU      string `json:"sku"`
	Position int    `json:"position"` // 1-индексированная позиция.
}

// SearchBoost — множитель к финальному скору при выполнении When.
type SearchBoost struct {
	When       SearchMatch `json:"when"`
	Multiplier float64     `json:"multiplier"`
	// Scope — необязательный фильтр scope: "all" (по умолчанию) или название секции.
	Scope string `json:"scope,omitempty"`
}

// SearchMatch — условия совпадения для буста (любое непустое поле сравнивается с кандидатом).
type SearchMatch struct {
	LeafCategory    string `json:"leaf_category,omitempty"`
	ProductType     string `json:"product_type,omitempty"`
	NormalizedColor string `json:"normalized_color,omitempty"`
	AttrKey         string `json:"attr_key,omitempty"`
	AttrValue       string `json:"attr_value,omitempty"`
	SKU             string `json:"sku,omitempty"`
}
