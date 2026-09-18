package strategy

import (
	_ "embed"
	"encoding/json"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

//go:embed fixtures/document_0206.json
var defaultDocument0206JSON []byte

// DefaultDocument — встроенный системный fallback: пресет «0206» (см. fixtures/document_0206.json).
func DefaultDocument() Document {
	var doc Document
	if err := json.Unmarshal(defaultDocument0206JSON, &doc); err == nil && doc.SchemaVersion > 0 {
		doc.SearchSynonyms = nil
		if doc.Search.PopularityWeight == 0 {
			doc.Search = SearchConfig{PopularityWeight: 0.3}
		}
		return ApplyLegacyMigrations(doc)
	}
	return defaultDocumentLegacy()
}

// defaultDocumentLegacy — запасной дефолт, если fixture недоступен.
func defaultDocumentLegacy() Document {
	d := Document{
		SchemaVersion: 1,
		Enabled:       true,
		SearchSynonyms: map[string][]string{
			"fit_slim": {"slim", "слим", "зауженный", "зауженные", "скинни", "skinny"},
		},
		UpdatedReason: "",
	}

	d.Weights.CategoryLeaf = 100
	d.Weights.CategorySharedLeaf = 80
	d.Weights.CategoryParentDeep = 70
	d.Weights.CategoryParent = 60
	d.Weights.CategoryParentShallow = 30
	d.Weights.CategoryBranch = 35
	d.Weights.GenderOrAgeGroup = 0
	d.Weights.ArticlePrefix = 12
	d.Weights.Price = 10

	d.Price.RangePercent = 25

	d.Title.Enabled = true
	d.Title.MaxScore = 30
	d.Title.ExactScore = 10
	d.Title.JaccardMaxScore = 20
	d.Title.ModelTokenScore = 5
	d.Title.ModelMaxScore = 15
	d.Title.FeatureTokenScore = 5
	d.Title.FeatureMaxScore = 15
	d.Title.NoCategoryCap = 10
	d.Title.MinTokenLength = 3
	d.Title.IgnoreModifierWordsEnabled = true
	d.Title.Stopwords = []string{"clever", "товар", "арт", "для", "из"}
	d.Title.ModelWords = []string{
		"wide", "straight", "slim", "skinny", "mom", "cargo", "oversize", "basic",
		"flare", "regular", "relaxed", "baggy",
		"палаццо", "клеш", "карго", "оверсайз", "базовый", "прямой", "широкий", "зауженный",
	}
	d.Title.FeatureWords = []string{
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
	}
	d.Title.SynonymTokenScore = 5
	d.Title.SynonymMaxScore = 15
	for _, g := range similar.DefaultTitleSynonymGroups() {
		d.Title.SynonymGroups = append(d.Title.SynonymGroups, SynonymGroup{
			ID: g.ID, Words: append([]string(nil), g.Words...),
		})
	}

	d.Thresholds.Strict.ScoreMin = 80
	d.Thresholds.Strict.BaseWithoutTitleMin = 50
	d.Thresholds.Strict.Strategy = "same_category_color_then_parent_fallback"
	d.Thresholds.Parent.ScoreMin = 50
	d.Thresholds.Parent.BaseWithoutTitleMin = 25
	d.Thresholds.Parent.Strategy = "parent_category_fallback"
	d.Thresholds.Branch.ScoreMin = 25
	d.Thresholds.Branch.Strategy = "branch_fallback"
	d.Thresholds.Popular.Enabled = true
	d.Thresholds.Popular.Strategy = "popular_in_section"

	d.Search = SearchConfig{
		PopularityWeight:      0.3,
		Pins:                  []SearchPin{},
		Boosts:                []SearchBoost{},
		ProductTypeDictionary: []string{},
	}

	d.ColorFamilyDictionary = DefaultColorFamilyDictionary()
	d.ColorFamilyPalette = DefaultColorFamilyPalette()
	colorRule := DefaultColorSimilarityRule()
	d.AttributeRules = []AttributeRule{
		{
			ID: "color_similarity", Enabled: true, LabelRu: "Цвет (семейства)",
			Source: "column", Key: "color", MatchType: "color_similarity",
			Weight: 30, MaxScore: 30, MissingPolicy: "no_score",
			Color: &colorRule,
		},
		{
			ID: genderRuleID, Enabled: true, LabelRu: "Пол / аудитория",
			Source: "column", Key: "gender", MatchType: "exact",
			Weight: 20, MaxScore: 20, MissingPolicy: "no_score",
		},
		{
			ID: "fit_waist", Enabled: true, LabelRu: "Высота талии",
			Source: "attributes", Key: "Высота талии", MatchType: "exact",
			Weight: 15, MaxScore: 15, MissingPolicy: "no_score",
		},
		{
			ID: "product_type_attr", Enabled: true, LabelRu: "Тип (фид)",
			Source: "attributes", Key: "Тип", MatchType: "exact",
			Weight: 10, MaxScore: 10, MissingPolicy: "no_score",
		},
	}
	for _, g := range catalog.AttributeSlotSetGroups {
		d.AttributeRules = append(d.AttributeRules, defaultSlotSetAttributeRule(g))
	}

	return d
}
