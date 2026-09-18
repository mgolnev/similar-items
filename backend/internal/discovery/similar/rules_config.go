package similar

// ColorSimilarityConfig — параметры match_type=color_similarity.
type ColorSimilarityConfig struct {
	RawExactScore     int
	PrimaryExactScore int
	AliasExactScore   int
	FamilyScore       int
	ShadeOverlapScore int
	ConflictPenalty   int
}

// AttributeRuleConfig — runtime-правило из similar_strategy.
type AttributeRuleConfig struct {
	ID            string
	Enabled       bool
	LabelRu       string
	Source        string
	Key           string
	MatchType     string
	Weight        int
	MaxScore      int
	MissingPolicy string
	ScopeSections []string
	Color         *ColorSimilarityConfig
}

// RulesConfig — расширяемые правила и словарь цветов.
type RulesConfig struct {
	Rules         []AttributeRuleConfig
	ColorFamilies map[string][]string
}

func (cfg StrategyConfig) hasColorSimilarityRule() bool {
	for _, r := range cfg.Rules.Rules {
		if r.Enabled && r.MatchType == "color_similarity" {
			return true
		}
	}
	return false
}
