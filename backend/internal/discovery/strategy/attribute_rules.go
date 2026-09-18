package strategy

// ColorSimilarityRule — подполя rule.color для match_type=color_similarity.
type ColorSimilarityRule struct {
	RawExactScore     int   `json:"raw_exact_score"`
	PrimaryExactScore int   `json:"primary_exact_score"`
	AliasExactScore   int   `json:"alias_exact_score"`
	FamilyScore       int   `json:"family_score"`
	ShadeOverlapScore int   `json:"shade_overlap_score"`
	ConflictPenalty   int   `json:"conflict_penalty"`
	GateOnConflict    *bool `json:"gate_on_conflict"`
}

// DefaultColorSimilarityRule — admin-ui-fields-and-defaults.md.
func DefaultColorSimilarityRule() ColorSimilarityRule {
	return ColorSimilarityRule{
		RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
		FamilyScore: 18, ShadeOverlapScore: 6, ConflictPenalty: -20,
	}
}

// AttributeRuleScope — ограничение правила по секциям.
type AttributeRuleScope struct {
	RootSections []string `json:"root_sections"`
}

// AttributeRule — расширяемое правило скоринга similar.
type AttributeRule struct {
	ID              string               `json:"id"`
	Enabled         bool                 `json:"enabled"`
	LabelRu         string               `json:"label_ru"`
	Source          string               `json:"source"`
	Key             string               `json:"key"`
	MatchType       string               `json:"match_type"`
	Weight          int                  `json:"weight"`
	MaxScore        int                  `json:"max_score"`
	MissingPolicy   string               `json:"missing_policy"`
	BonusSame       *int                 `json:"bonus_same,omitempty"`
	PenaltyConflict *int                 `json:"penalty_conflict,omitempty"`
	GateOnConflict  *bool                `json:"gate_on_conflict,omitempty"`
	Scope           *AttributeRuleScope  `json:"scope,omitempty"`
	Color           *ColorSimilarityRule `json:"color,omitempty"`
}

// DefaultColorFamilyDictionary — встроенный словарь цветовых семейств.
func DefaultColorFamilyDictionary() map[string][]string {
	return map[string][]string{
		"blue":   {"синий", "голубой", "деним", "индиго", "navy", "blue"},
		"black":  {"черный", "чёрный", "black"},
		"white":  {"белый", "молочный", "айвори", "ivory", "white"},
		"gray":   {"серый", "графит", "меланж", "gray", "grey"},
		"beige":  {"бежевый", "кремовый", "песочный", "beige"},
		"brown":  {"коричневый", "шоколадный", "brown"},
		"green":  {"зеленый", "зелёный", "хаки", "оливковый", "green"},
		"red":    {"красный", "бордовый", "red"},
		"pink":   {"розовый", "фуксия", "pink"},
		"purple": {"фиолетовый", "сиреневый", "лиловый", "purple"},
		"yellow": {"желтый", "жёлтый", "yellow"},
		"orange": {"оранжевый", "orange"},
	}
}

// DefaultColorFamilyPalette — визуальная палитра цветовых семейств для витрины.
func DefaultColorFamilyPalette() map[string]string {
	return map[string]string{
		"blue":   "#2F5DA8",
		"black":  "#1F1F1F",
		"white":  "#F7F4ED",
		"gray":   "#9CA3AF",
		"beige":  "#D8C3A5",
		"brown":  "#7A4A2E",
		"green":  "#6B7F3A",
		"red":    "#C62828",
		"pink":   "#E78AA9",
		"purple": "#8B5FBF",
		"yellow": "#F2C94C",
		"orange": "#E9852C",
	}
}
