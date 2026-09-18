package similar

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// loadActiveStrategyConfig поднимает StrategyConfig из снапшота активного
// пресета gs-test (testdata/gs_test_active_strategy.json) — бенчим боевой
// конфиг (22 attribute-правила, title-словари), а не пустой DefaultConfig.
func loadActiveStrategyConfig(tb testing.TB) StrategyConfig {
	raw, err := os.ReadFile("testdata/gs_test_active_strategy.json")
	if err != nil {
		tb.Fatalf("read strategy snapshot: %v", err)
	}
	var doc struct {
		Price struct {
			RangePercent int `json:"range_percent"`
		} `json:"price"`
		Title struct {
			Enabled             bool           `json:"enabled"`
			ExactScore          int            `json:"exact_score"`
			JaccardMax          int            `json:"jaccard_max_score"`
			ModelTokenScore     int            `json:"model_token_score"`
			ModelMax            int            `json:"model_max_score"`
			FeatureTokenScore   int            `json:"feature_token_score"`
			FeatureMax          int            `json:"feature_max_score"`
			SynonymTokenScore   int            `json:"synonym_token_score"`
			SynonymMax          int            `json:"synonym_max_score"`
			NoCategoryCap       int            `json:"no_category_cap"`
			MinTokenLength      int            `json:"min_token_length"`
			Stopwords           []string       `json:"stopwords"`
			ModelWords          []string       `json:"model_words"`
			FeatureWords        []string       `json:"feature_words"`
			SynonymGroups       []SynonymGroup `json:"synonym_groups"`
			IgnoreModifierWords bool           `json:"ignore_modifier_words_enabled"`
		} `json:"title"`
		Rules []struct {
			ID            string   `json:"id"`
			Enabled       bool     `json:"enabled"`
			Source        string   `json:"source"`
			Key           string   `json:"key"`
			MatchType     string   `json:"match_type"`
			Weight        int      `json:"weight"`
			MaxScore      int      `json:"max_score"`
			ScopeSections []string `json:"scope_sections"`
			Color         *struct {
				RawExactScore     int `json:"raw_exact_score"`
				PrimaryExactScore int `json:"primary_exact_score"`
				AliasExactScore   int `json:"alias_exact_score"`
				FamilyScore       int `json:"family_score"`
				ShadeOverlapScore int `json:"shade_overlap_score"`
				ConflictPenalty   int `json:"conflict_penalty"`
			} `json:"color"`
		} `json:"attribute_rules"`
		Families map[string][]string `json:"color_family_dictionary"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		tb.Fatalf("parse strategy snapshot: %v", err)
	}

	cfg := DefaultConfig()
	cfg.PriceRangePercent = doc.Price.RangePercent
	cfg.Title = TitleConfig{
		Enabled: doc.Title.Enabled, ExactScore: doc.Title.ExactScore,
		JaccardMax: doc.Title.JaccardMax, ModelMax: doc.Title.ModelMax,
		ModelTokenScore: doc.Title.ModelTokenScore, FeatureMax: doc.Title.FeatureMax,
		FeatureTokenScore: doc.Title.FeatureTokenScore, NoCategoryCap: doc.Title.NoCategoryCap,
		MinTokenLength: doc.Title.MinTokenLength, Stopwords: doc.Title.Stopwords,
		ModelWords: doc.Title.ModelWords, FeatureWords: doc.Title.FeatureWords,
		SynonymTokenScore: doc.Title.SynonymTokenScore, SynonymMax: doc.Title.SynonymMax,
		SynonymGroups:       doc.Title.SynonymGroups,
		IgnoreModifierWords: doc.Title.IgnoreModifierWords,
	}
	cfg.Title.SynonymIndex = BuildSynonymIndex(cfg.Title.SynonymGroups)
	cfg.Rules.ColorFamilies = doc.Families
	for _, r := range doc.Rules {
		rc := AttributeRuleConfig{
			ID: r.ID, Enabled: r.Enabled, Source: r.Source, Key: r.Key,
			MatchType: r.MatchType, Weight: r.Weight, MaxScore: r.MaxScore,
			ScopeSections: r.ScopeSections,
		}
		if r.Color != nil {
			rc.Color = &ColorSimilarityConfig{
				RawExactScore: r.Color.RawExactScore, PrimaryExactScore: r.Color.PrimaryExactScore,
				AliasExactScore: r.Color.AliasExactScore, FamilyScore: r.Color.FamilyScore,
				ShadeOverlapScore: r.Color.ShadeOverlapScore, ConflictPenalty: r.Color.ConflictPenalty,
			}
		}
		cfg.Rules.Rules = append(cfg.Rules.Rules, rc)
	}
	return cfg
}

// benchPoolActive — пул с атрибутами под ключи активных правил.
func benchPoolActive(tb testing.TB, n int, cfg StrategyConfig) []ProductRow {
	rows := make([]ProductRow, n)
	colors := []string{"чёрный", "белый", "синий", "красный", "бежевый меланж"}
	for i := range rows {
		attrs := fmt.Sprintf(`{"color":%q,"Длина рукава":"короткий рукав","Силуэт":"прямой","Тип посадки":"средняя посадка","Застёжка":"пуговицы","Конструктивные элементы":"накладной карман","Возраст":"взрослый","Узор":"графика %d"}`, colors[i%len(colors)], i%7)
		rows[i] = ProductRow{
			SKU:          fmt.Sprintf("BWT%06d-%d", i/2, i%2+1),
			Name:         fmt.Sprintf("Рубашка оверсайз с коротким рукавом и принтом %d", i%37),
			CategoryPath: `["Каталог","Мужчины","Одежда","Рубашки"]`,
			RootSection:  "men",
			Gender:       "мужской",
			Color:        colors[i%len(colors)],
			Price:        float64(999 + (i%40)*100),
			Attributes:   attrs,
			ImageURL:     "https://cdn.example/img.jpg",
		}
		enrichProductRow(&rows[i])
	}
	d := newTitleDicts(cfg.Title)
	for i := range rows {
		precomputeTitle(&rows[i], d, cfg.Title)
		precomputeRuleFeatures(&rows[i], cfg)
	}
	idSpace := newPreparedIDSpace(len(rows) * 3)
	for i := range rows {
		prepareNumericIDs(&rows[i], idSpace)
	}
	return rows
}

func BenchmarkRankSimilarCandidates_ActiveStrategy(b *testing.B) {
	cfg := loadActiveStrategyConfig(b)
	pool := benchPoolActive(b, 1500, cfg)
	base := pool[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RankSimilarCandidates(base, pool, cfg)
	}
}

// TestRuleFeats_EquivalentToSlowPath: прекомпьют-путь правил обязан давать
// РОВНО те же суммы и разбивку, что и медленный путь — на боевом конфиге.
func TestRuleFeats_EquivalentToSlowPath(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	pool := benchPoolActive(t, 60, cfg) // с фичами

	stripped := make([]ProductRow, len(pool))
	copy(stripped, pool)
	for i := range stripped {
		stripped[i].ruleFeats = nil // медленный путь
	}

	for i := 0; i < len(pool); i++ {
		for j := 0; j < len(pool); j += 7 {
			fastTotal, fastBy := scoreAttributeRules(pool[i], pool[j], cfg)
			slowTotal, slowBy := scoreAttributeRules(stripped[i], stripped[j], cfg)
			if fastTotal != slowTotal {
				t.Fatalf("pair (%d,%d): total fast=%d slow=%d (by fast=%v slow=%v)", i, j, fastTotal, slowTotal, fastBy, slowBy)
			}
			if len(fastBy) != len(slowBy) {
				t.Fatalf("pair (%d,%d): byRule len fast=%v slow=%v", i, j, fastBy, slowBy)
			}
			for k, v := range slowBy {
				if fastBy[k] != v {
					t.Fatalf("pair (%d,%d) rule %s: fast=%d slow=%d", i, j, k, fastBy[k], v)
				}
			}
		}
	}
}

func TestRuleFeats_genderTeenPrecomputeMatchesSlowPath(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	girl := ProductRow{
		SKU: "G1", RootSection: "teens", Gender: "Девочки", ImageURL: "http://a",
		LeafCategory: "Брюки для девушек",
		CategoryPath: `["Брюки для девушек"]`,
	}
	uni := ProductRow{
		SKU: "G2", RootSection: "teens", Gender: "Унисекс", ImageURL: "http://b",
		LeafCategory: "Спортивные брюки для девушек",
		CategoryPath: `["Спортивная одежда для девушек","Спортивные брюки для девушек"]`,
	}
	boy := ProductRow{
		SKU: "B1", RootSection: "teens", Gender: "Мальчики", ImageURL: "http://c",
		LeafCategory: "Свободные брюки для мальчиков-подростков",
		CategoryPath: `["Подростки","Мальчики-подростки","Свободные брюки для мальчиков-подростков"]`,
	}
	for _, row := range []*ProductRow{&girl, &uni, &boy} {
		enrichProductRow(row)
		precomputeRuleFeatures(row, cfg)
	}
	strip := func(r ProductRow) ProductRow {
		r.ruleFeats = nil
		return r
	}

	cases := []struct {
		base, cand ProductRow
	}{
		{girl, uni},
		{girl, boy},
		{uni, girl},
	}
	for _, tc := range cases {
		fast, _ := scoreAttributeRules(tc.base, tc.cand, cfg)
		slow, _ := scoreAttributeRules(strip(tc.base), strip(tc.cand), cfg)
		if fast != slow {
			t.Fatalf("%s vs %s: fast=%d slow=%d", tc.base.SKU, tc.cand.SKU, fast, slow)
		}
	}
}
