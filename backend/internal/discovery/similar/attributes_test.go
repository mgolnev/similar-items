package similar

import (
	"strings"
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

func TestExtractColorFromGroupID(t *testing.T) {
	got := catalog.ExtractColorFromGroupID("GSH012871-1 светло-голубой /супер арктик/")
	if !strings.Contains(got, "голубой") {
		t.Fatalf("got %q", got)
	}
}

func TestEnrichProductRow_fromGroupID(t *testing.T) {
	row := ProductRow{GroupID: "GSH012871-1 светло-голубой /супер арктик/"}
	enrichProductRow(&row)
	if row.NormalizedColor == "" {
		t.Fatal("expected normalized color")
	}
}

func TestEnrichDiscoveryCategory_prefersCatalogBranch(t *testing.T) {
	feed := "2:Коллекции | 3456:Content: ручные подборки | 3236:Базовая коллекция | " +
		"2417:Базовая коллекция одежды для женщин | 5:Каталог | 9:Женщины | " +
		"28:Женские футболки и лонгсливы | 3365:Женские укороченные футболки"
	row := ProductRow{
		CategoriesFeed: feed,
		CategoryPath:   `["Content: ручные подборки","Базовая коллекция","Базовая коллекция одежды для женщин"]`,
		LeafCategory:   "Базовая коллекция одежды для женщин",
		Gender:         "Женщины",
	}
	enrichProductRow(&row)
	if row.LeafCategory != "Женские укороченные футболки" {
		t.Fatalf("leaf: %q path: %s", row.LeafCategory, row.CategoryPath)
	}

	socksFeed := feed + " | 5:Каталог | 9:Женщины | 666:Женские носки и колготки | 756:Женские носки"
	socks := ProductRow{
		CategoriesFeed: socksFeed,
		CategoryPath:   row.CategoryPath,
		LeafCategory:   row.LeafCategory,
		Gender:         "Женщины",
	}
	enrichProductRow(&socks)
	if socks.LeafCategory == row.LeafCategory {
		t.Fatalf("socks must differ from top: %q", socks.LeafCategory)
	}
}

func TestScorePair_twoBlueFromFeedGroupIDs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rules.ColorFamilies = map[string][]string{
		"blue": {"синий", "голубой", "лайт", "айс", "арктик", "овердайт"},
	}
	cfg.Rules.Rules = []AttributeRuleConfig{{
		ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
		MatchType: "color_similarity", MaxScore: 30,
		Color: &ColorSimilarityConfig{FamilyScore: 18, AliasExactScore: 22},
	}}
	base := ProductRow{GroupID: "GJN032711-2 синий /лайт айс/ (светло-голубой)"}
	cand := ProductRow{GroupID: "GJN037254-1 синий /лайт айс/ (голубой)"}
	enrichProductRow(&base)
	enrichProductRow(&cand)
	b := scorePairDetailed(base, cand, cfg)
	if b.Color <= 0 {
		t.Fatalf("color=%d rules=%v normB=%q normC=%q", b.Color, b.RuleScores, base.NormalizedColor, cand.NormalizedColor)
	}
}

func TestScorePair_blueVariantsFromGroupID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rules = RulesConfig{
		ColorFamilies: map[string][]string{"blue": {"синий", "голубой", "айс", "арктик"}},
		Rules: []AttributeRuleConfig{{
			ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
			MatchType: "color_similarity", MaxScore: 30,
			Color: &ColorSimilarityConfig{FamilyScore: 18},
		}},
	}
	base := ProductRow{GroupID: "GSH012871-1 светло-голубой /супер арктик/"}
	cand := ProductRow{GroupID: "GSH012871-2 синий /медиум-лайт айс/"}
	enrichProductRow(&base)
	enrichProductRow(&cand)
	b := scorePairDetailed(base, cand, cfg)
	if b.Color <= 0 {
		t.Fatalf("expected color score, got %d rules=%v", b.Color, b.RuleScores)
	}
}
