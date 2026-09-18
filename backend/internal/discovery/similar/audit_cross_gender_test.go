package similar

import (
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

// Audit: ложные срабатывания teen affinity и gender между полами в teens.
func TestAuditTeenSubsection_noCrossGenderFalsePositives(t *testing.T) {
	cfg := DefaultConfig()
	girl := ProductRow{
		SKU: "GPT011623-1", RootSection: "teens", Gender: "Девочки",
		LeafCategory: "Брюки для девушек",
		CategoryPath: `["Брюки для девушек"]`, ImageURL: "http://a", Name: "Чёрные брюки клёш",
	}
	enrichProductRow(&girl)

	falsePositiveCases := []struct {
		name string
		row  ProductRow
	}{
		{
			name: "boy teen pants",
			row: ProductRow{
				SKU: "BSU000511-1", RootSection: "teens", Gender: "Мальчики",
				LeafCategory: "Свободные брюки для мальчиков-подростков",
				CategoryPath: `["Подростки","Мальчики-подростки","Брюки для мальчиков-подростков","Свободные брюки для мальчиков-подростков"]`,
				ImageURL:     "http://b",
			},
		},
		{
			name: "women pants",
			row: ProductRow{
				SKU: "W1", RootSection: "women", Gender: "Женщины",
				LeafCategory: "Женские брюки",
				CategoryPath: `["Женщины","Женские брюки"]`, ImageURL: "http://c",
			},
		},
		{
			name: "kids girl pants",
			row: ProductRow{
				SKU: "K1", RootSection: "kids", Gender: "Девочки",
				LeafCategory: "Брюки для девочек",
				CategoryPath: `["Дети","Девочки","Брюки для девочек"]`, ImageURL: "http://d",
			},
		},
		{
			name: "teen boy hoodie not pants",
			row: ProductRow{
				SKU: "B1", RootSection: "teens", Gender: "Мальчики",
				LeafCategory: "Толстовки для мальчиков-подростков",
				CategoryPath: `["Подростки","Мальчики-подростки","Толстовки для мальчиков-подростков"]`,
				ImageURL:     "http://e",
			},
		},
	}

	for _, tc := range falsePositiveCases {
		t.Run(tc.name, func(t *testing.T) {
			enrichProductRow(&tc.row)
			b := ScorePair(girl, tc.row, cfg)
			if b.CategoryKind == "teen_subsection" || b.CategoryKind == "teen_subsection_class" || b.CategoryKind == "teen_girl_pants" {
				t.Fatalf("unexpected teen affinity for %s: %+v", tc.name, b.CategoryKind)
			}
		})
	}
}

func TestAuditGenderCompatible_noBoyGirlMatch(t *testing.T) {
	girl := ProductRow{RootSection: "teens", Gender: "Девочки", Attributes: `{"Пол":"Девочки"}`}
	boy := ProductRow{
		RootSection: "teens", Gender: "Мальчики",
		LeafCategory: "Свободные брюки для мальчиков-подростков",
		CategoryPath: `["Подростки","Мальчики-подростки","Свободные брюки для мальчиков-подростков"]`,
		Attributes:   `{"Пол":"Мальчики"}`,
	}
	rule := AttributeRuleConfig{Key: "gender", MatchType: "exact", Weight: 20}
	if genderCompatibleForSimilar(girl, boy, rule, "девочки", "мальчики") {
		t.Fatal("boy vs girl must not be gender-compatible")
	}
	if genderCompatibleForSimilar(girl, boy, rule, "девочки", "унисекс") {
		t.Fatal("boy-coded product must not match via unisex shortcut")
	}
}

func TestSimilar_sameLeafLabelDifferentCatalogPathAndGenderIsExcluded(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "GAC026313-2", GroupID: "GAC026313-2", Name: "Чёрные спортивные джоггеры",
		Gender: "Девочки", RootSection: "teens", ImageURL: "http://girls",
		CategoryPath: "[\"Подростки\",\"Девочки-подростки\",\"Школьная одежда для девочек-подростков\",\"Чёрная форма\"]",
		LeafCategory: "Чёрная форма",
	}
	boy := ProductRow{
		SKU: "BSU000463-2", GroupID: "BSU000463-2", Name: "Чёрные школьные брюки джоггеры",
		Gender: "Мальчики", RootSection: "teens", ImageURL: "http://boys",
		CategoryPath: "[\"Подростки\",\"Мальчики-подростки\",\"Школьная одежда для мальчиков-подростков\",\"Чёрная форма\"]",
		LeafCategory: "Чёрная форма",
	}
	girl := ProductRow{
		SKU: "GAC026340-1", GroupID: "GAC026340-1", Name: "Чёрные спортивные брюки клёш",
		Gender: "Девочки", RootSection: "teens", ImageURL: "http://girls-2",
		CategoryPath: "[\"Подростки\",\"Девочки-подростки\",\"Школьная одежда для девочек-подростков\",\"Чёрная форма\"]",
		LeafCategory: "Чёрная форма",
	}
	enrichProductRow(&base)
	enrichProductRow(&boy)
	enrichProductRow(&girl)

	boyBreakdown := ScorePair(base, boy, cfg)
	if boyBreakdown.CategoryKind == "leaf" || boyBreakdown.Category == cfg.Weights.CategoryLeaf {
		t.Fatalf("different catalog paths must not match as leaf: %+v", boyBreakdown.ScoreBreakdown)
	}

	ranked := RankSimilarCandidates(base, []ProductRow{boy, girl}, cfg)
	if len(ranked) != 1 || ranked[0].row.SKU != girl.SKU {
		t.Fatalf("opposite-gender candidate must be excluded, ranked=%v", ranked)
	}
}

func TestExplicitGenderConflict_unknownAndUnisexDoNotBlock(t *testing.T) {
	if !explicitGenderConflict("Девочки", "Мальчики") {
		t.Fatal("girls vs boys must conflict")
	}
	if !explicitGenderConflict("Женщины", "Мужчины") {
		t.Fatal("women vs men must conflict")
	}
	if explicitGenderConflict("Девочки", "Унисекс") || explicitGenderConflict("Девочки", "") {
		t.Fatal("unisex or missing gender must not block")
	}
}

func TestAuditIsTeenGirlCatalogLabel_matrix(t *testing.T) {
	wantTrue := []string{
		"Брюки для девушек",
		"Спортивные брюки для девочек-подростков",
		"Ready! Steady! Go! для девушек", // marketing name, but label check used on catalog segments
	}
	wantFalse := []string{
		"Свободные брюки для мальчиков-подростков",
		"Брюки для мальчиков-подростков",
		"Подростки",
		"Мальчики-подростки",
		"Женские брюки",
	}
	for _, l := range wantTrue {
		if !catalog.IsTeenGirlCatalogLabel(l) {
			t.Errorf("expected true for %q", l)
		}
	}
	for _, l := range wantFalse {
		if catalog.IsTeenGirlCatalogLabel(l) {
			t.Errorf("expected false for %q", l)
		}
	}
	// «Девочки-подростки» — узел навигации, допустимо true для root-section inference
	if !catalog.IsTeenGirlCatalogLabel("Девочки-подростки") {
		t.Error("expected true for subsection node Девочки-подростки")
	}
}

func TestAuditResolveRootSection_pathWithoutPodrostkiRoot(t *testing.T) {
	// RSG-only path (как GPT011623-1 после strip marketing)
	if got := catalog.ResolveRootSection("", []string{"Брюки для девушек"}, "Девочки"); got != catalog.SectionTeens {
		t.Fatalf("RSG pants: got %q want teens", got)
	}
	// Путь без «Подростки» в корне — не должен падать в kids из-за «девоч» в сегменте
	if got := catalog.ResolveRootSection("", []string{"Девочки-подростки", "Брюки для девочек-подростков"}, "Девочки"); got != catalog.SectionTeens {
		t.Fatalf("teen girl path without Podrostki root: got %q want teens", got)
	}
	if got := catalog.ResolveRootSection("", []string{"Дети", "Девочки", "Брюки для девочек"}, "Девочки"); got != catalog.SectionKids {
		t.Fatalf("kids path: got %q want kids", got)
	}
}
