package strategy

import "testing"

func TestDefaultDocument_hasColorFamilies(t *testing.T) {
	d := DefaultDocument()
	if len(d.ColorFamilyDictionary) < 10 {
		t.Fatalf("expected color families, got %d", len(d.ColorFamilyDictionary))
	}
	if _, ok := d.ColorFamilyDictionary["blue"]; !ok {
		t.Fatal("missing blue family")
	}
	if d.ColorFamilyPalette["blue"] == "" {
		t.Fatal("missing blue palette color")
	}
}

func TestDefaultColorSimilarityRule(t *testing.T) {
	c := DefaultColorSimilarityRule()
	if c.RawExactScore != 30 || c.FamilyScore != 18 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestDefaultDocument_hasTitleSynonyms(t *testing.T) {
	d := DefaultDocument()
	if d.Title.SynonymMaxScore <= 0 {
		t.Fatal("expected synonym_max_score > 0 in default document")
	}
	if len(d.Title.SynonymGroups) < 3 {
		t.Fatalf("expected synonym groups in default, got %d", len(d.Title.SynonymGroups))
	}
	cfg := d.ToSimilarConfig()
	if len(cfg.Title.SynonymIndex) == 0 {
		t.Fatal("expected prebuilt SynonymIndex in runtime config")
	}
	empty := Document{}
	empty.Title.SynonymMaxScore = 0
	emptyCfg := empty.ToSimilarConfig()
	if emptyCfg.Title.SynonymIndex != nil {
		t.Fatal("empty synonyms must keep SynonymIndex nil (fast-path)")
	}
}

func TestDefaultDocument_uses0206Fixture(t *testing.T) {
	d := DefaultDocument()
	if d.Weights.CategoryLeaf != 90 {
		t.Fatalf("expected 0206 category_leaf=90, got %d", d.Weights.CategoryLeaf)
	}
	if len(d.AttributeRules) < 20 {
		t.Fatalf("expected 0206 attribute rules (>=20), got %d", len(d.AttributeRules))
	}
	var hasDecor, hasPattern bool
	for _, r := range d.AttributeRules {
		if r.ID == "decor" && r.MatchType == "set_overlap" {
			hasDecor = true
		}
		if r.ID == "pattern" && r.MatchType == "set_overlap" {
			hasPattern = true
		}
	}
	if !hasDecor || !hasPattern {
		t.Fatalf("expected slot-set rules in 0206 default: decor=%v pattern=%v", hasDecor, hasPattern)
	}
}
