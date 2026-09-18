package similar

import (
	"testing"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

func TestResolveAttributeValue_prefersExactKey(t *testing.T) {
	t.Parallel()

	row := ProductRow{Attributes: `{"Тип рукава":"стандартный","Тип кармана":"без карманов","Узор":"однотонный"}`}
	if got := resolveAttributeValue(row, "Тип"); got != "" {
		t.Fatalf("short key must not fuzzy-match, got %q", got)
	}
	if got := resolveAttributeValue(row, "Узор"); got != "однотонный" {
		t.Fatalf("exact key: got %q", got)
	}
}

func TestScoreSetOverlap_decorSlots(t *testing.T) {
	t.Parallel()

	rule := AttributeRuleConfig{
		ID: "decor", Enabled: true, Source: "attributes",
		Key: catalog.DecorSetAttributeKey, MatchType: "set_overlap",
		Weight: 12, MaxScore: 12,
	}
	base := ProductRow{Attributes: `{"Декор":"планка","Декор2":"принт","Декор3":"сетка"}`}
	cand := ProductRow{Attributes: `{"Декор":"принт","Декор2":"планка"}`}
	base.Attributes = catalog.RebuildAttributeSlotSetsInAttributesJSON(base.Attributes)
	cand.Attributes = catalog.RebuildAttributeSlotSetsInAttributesJSON(cand.Attributes)

	got := scoreSetOverlapRule(base, cand, rule)
	if got <= 0 {
		t.Fatalf("expected overlap score, got %d", got)
	}
	full := ProductRow{Attributes: base.Attributes}
	if scoreSetOverlapRule(base, full, rule) != 12 {
		t.Fatalf("full match should hit max_score")
	}
}

func TestExplainAttributeRules_missingBoth(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Rules.Rules = append(cfg.Rules.Rules, AttributeRuleConfig{
		ID: "sil", Enabled: true, Source: "attributes", Key: "Силуэт", MatchType: "exact", Weight: 10,
	})
	base := ProductRow{Attributes: `{"Узор":"однотонный"}`}
	cand := ProductRow{Attributes: `{"Узор":"однотонный"}`}
	details := explainAttributeRules(base, cand, cfg)
	if details["sil"].Status != "missing_both" {
		t.Fatalf("status=%q", details["sil"].Status)
	}
}
