package similar

import "testing"

func TestColorSimilarity_sameFamily(t *testing.T) {
	families := map[string][]string{
		"blue": {"синий", "голубой", "деним", "айс"},
	}
	cfg := StrategyConfig{
		Rules: RulesConfig{
			ColorFamilies: families,
			Rules: []AttributeRuleConfig{{
				ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
				MatchType: "color_similarity", MaxScore: 30,
				Color: &ColorSimilarityConfig{
					RawExactScore: 30, FamilyScore: 18, ShadeOverlapScore: 6,
				},
			}},
		},
	}
	base := ProductRow{NormalizedColor: "синий /лайт айс/ (синий)"}
	cand := ProductRow{NormalizedColor: "синий /медиум айс овердайт/ (синий)"}
	_, scores := scoreAttributeRules(base, cand, cfg)
	if scores["color_similarity"] <= 0 {
		t.Fatalf("expected color family score, got %v", scores)
	}
}

func TestColorSimilarity_MultiColorDoesNotMatchSingleColor(t *testing.T) {
	families := map[string][]string{
		"brown": {"коричневый", "шоколадный"},
		"white": {"молочный", "белый"},
	}
	rule := AttributeRuleConfig{
		ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
		MatchType: "color_similarity", MaxScore: 30,
		Color: &ColorSimilarityConfig{
			RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
			FamilyScore: 18, ShadeOverlapScore: 6,
		},
	}

	base := ProductRow{NormalizedColor: "коричневый/молочный"}
	cand := ProductRow{NormalizedColor: "коричневый"}

	if got := scoreColorSimilarityRule(base, cand, rule, families); got != 0 {
		t.Fatalf("multi-color must not match single color, got score %d", got)
	}
}

func TestColorSimilarity_MultiColorMatchesSimilarMultiColor(t *testing.T) {
	families := map[string][]string{
		"brown": {"коричневый", "шоколадный"},
		"white": {"молочный", "белый"},
		"black": {"черный"},
	}
	rule := AttributeRuleConfig{
		ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
		MatchType: "color_similarity", MaxScore: 30,
		Color: &ColorSimilarityConfig{
			RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
			FamilyScore: 18, ShadeOverlapScore: 6,
		},
	}

	base := ProductRow{NormalizedColor: "коричневый/молочный"}
	sameSet := ProductRow{NormalizedColor: "коричневый/белый"}
	partialSet := ProductRow{NormalizedColor: "коричневый/черный"}

	if got := scoreColorSimilarityRule(base, sameSet, rule, families); got < 18 {
		t.Fatalf("same multi-color family set should get family score, got %d", got)
	}
	if got := scoreColorSimilarityRule(base, partialSet, rule, families); got <= 0 || got >= 18 {
		t.Fatalf("partial multi-color overlap should get reduced score, got %d", got)
	}
}

func TestColorSimilarity_AmbiguousAliasIsSingleColor(t *testing.T) {
	families := map[string][]string{
		"beige": {"бежевый", "молочный"},
		"white": {"белый", "молочный"},
		"brown": {"коричневый"},
	}
	rule := AttributeRuleConfig{
		ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
		MatchType: "color_similarity", MaxScore: 30,
		Color: &ColorSimilarityConfig{
			RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
			FamilyScore: 18, ShadeOverlapScore: 6,
		},
	}

	milkFamilies, milkMulti := colorFamilyInfo("молочный", families)
	if milkMulti {
		t.Fatal("one alias shared by several families must remain a single color")
	}
	if len(milkFamilies) != 2 {
		t.Fatalf("expected both possible families for milk, got %v", milkFamilies)
	}

	base := ProductRow{NormalizedColor: "бежевый"}
	milk := ProductRow{NormalizedColor: "молочный"}
	if got := scoreColorSimilarityRule(base, milk, rule, families); got != 18 {
		t.Fatalf("slow score: expected beige family score 18, got %d", got)
	}
	bf := computeRuleFeat(base, rule, families)
	mf := computeRuleFeat(milk, rule, families)
	if got := scoreColorSimilarityFeats(&bf, &mf, &rule); got != 18 {
		t.Fatalf("fast score: expected beige family score 18, got %d", got)
	}

	if _, multi := colorFamilyInfo("бежевый/молочный", families); multi {
		t.Fatal("components with a shared possible family must remain a single color")
	}
	if _, multi := colorFamilyInfo("коричневый/молочный", families); !multi {
		t.Fatal("components without a shared possible family must be multi-color")
	}
}

func TestColorSimilarity_ConflictPenaltyContributesToTotals(t *testing.T) {
	families := map[string][]string{
		"black":  {"черный"},
		"white":  {"белый"},
		"beige":  {"бежевый"},
		"orange": {"оранжевый"},
		"yellow": {"оранжевый", "желтый"},
	}
	rule := AttributeRuleConfig{
		ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
		MatchType: "color_similarity", MaxScore: 35,
		Color: &ColorSimilarityConfig{
			RawExactScore: 35, PrimaryExactScore: 25, AliasExactScore: 22,
			FamilyScore: 20, ShadeOverlapScore: 5, ConflictPenalty: -20,
		},
	}
	cfg := StrategyConfig{Rules: RulesConfig{
		ColorFamilies: families,
		Rules:         []AttributeRuleConfig{rule},
	}}
	base := ProductRow{NormalizedColor: "черный"}
	cand := ProductRow{NormalizedColor: "белый"}

	if got := scoreColorSimilarityRule(base, cand, rule, families); got != -20 {
		t.Fatalf("color rule: expected conflict penalty -20, got %d", got)
	}
	if total, scores := scoreAttributeRules(base, cand, cfg); total != -20 || scores[rule.ID] != -20 {
		t.Fatalf("detailed aggregation: expected total/rule -20, got total=%d scores=%v", total, scores)
	}
	if got := scoreAttributeRulesTotal(&base, &cand, &cfg); got != -20 {
		t.Fatalf("slow production aggregation: expected -20, got %d", got)
	}

	precomputeRuleFeatures(&base, cfg)
	precomputeRuleFeatures(&cand, cfg)
	if got := scoreAttributeRulesTotal(&base, &cand, &cfg); got != -20 {
		t.Fatalf("fast production aggregation: expected -20, got %d", got)
	}
	if details := explainAttributeRules(base, cand, cfg); details[rule.ID].Status != "conflict" {
		t.Fatalf("expected conflict detail, got %+v", details[rule.ID])
	}

	beige := ProductRow{NormalizedColor: "бежевый"}
	orangeBlack := ProductRow{NormalizedColor: "оранжевый/черный"}
	if got := scoreColorSimilarityRule(beige, orangeBlack, rule, families); got != -20 {
		t.Fatalf("single vs disjoint multi-color: expected -20, got %d", got)
	}
	beigeFeat := computeRuleFeat(beige, rule, families)
	orangeBlackFeat := computeRuleFeat(orangeBlack, rule, families)
	if got := scoreColorSimilarityFeats(&beigeFeat, &orangeBlackFeat, &rule); got != -20 {
		t.Fatalf("fast single vs disjoint multi-color: expected -20, got %d", got)
	}
	unknown := ProductRow{NormalizedColor: "неизвестный цвет"}
	if got := scoreColorSimilarityRule(beige, unknown, rule, families); got != 0 {
		t.Fatalf("unknown family must not be penalized, got %d", got)
	}
}

func TestColorFamilies_PrefersFullPhraseAndFirstColorToken(t *testing.T) {
	families := map[string][]string{
		"beige": {"бежевый", "бежевый меланж", "темно-бежевый"},
		"brown": {"коричневый"},
		"gray":  {"серый", "меланж"},
	}

	tests := []struct {
		color string
		want  string
	}{
		{color: "бежевый меланж", want: "beige"},
		{color: "серый меланж", want: "gray"},
		{color: "тёмно-бежевый", want: "beige"},
		{color: "коричневый меланж", want: "brown"},
	}
	for _, tt := range tests {
		got := colorFamiliesForString(tt.color, families)
		if len(got) != 1 {
			t.Fatalf("%q: expected one family %q, got %v", tt.color, tt.want, got)
		}
		if _, ok := got[tt.want]; !ok {
			t.Fatalf("%q: expected family %q, got %v", tt.color, tt.want, got)
		}
	}
}

func TestColorSimilarity_BeigeMelangeMatchesDarkBeigeButNotGrayMelange(t *testing.T) {
	families := map[string][]string{
		"beige": {"бежевый", "бежевый меланж", "темно-бежевый"},
		"gray":  {"серый", "меланж"},
	}
	rule := AttributeRuleConfig{
		ID: "color_similarity", Enabled: true, Source: "column", Key: "color",
		MatchType: "color_similarity", MaxScore: 30,
		Color: &ColorSimilarityConfig{
			RawExactScore: 30, PrimaryExactScore: 24, AliasExactScore: 22,
			FamilyScore: 18, ShadeOverlapScore: 6, ConflictPenalty: -20,
		},
	}
	base := ProductRow{NormalizedColor: "бежевый меланж"}
	darkBeige := ProductRow{NormalizedColor: "тёмно-бежевый"}
	gray := ProductRow{NormalizedColor: "серый меланж"}

	if got := scoreColorSimilarityRule(base, darkBeige, rule, families); got != 18 {
		t.Fatalf("dark beige: expected family score 18, got %d", got)
	}
	if got := scoreColorSimilarityRule(base, gray, rule, families); got != -20 {
		t.Fatalf("gray melange: expected conflict penalty -20, got %d", got)
	}

	bf := computeRuleFeat(base, rule, families)
	darkBeigeFeat := computeRuleFeat(darkBeige, rule, families)
	grayFeat := computeRuleFeat(gray, rule, families)
	if got := scoreColorSimilarityFeats(&bf, &darkBeigeFeat, &rule); got != 18 {
		t.Fatalf("fast dark beige: expected family score 18, got %d", got)
	}
	if got := scoreColorSimilarityFeats(&bf, &grayFeat, &rule); got != -20 {
		t.Fatalf("fast gray melange: expected conflict penalty -20, got %d", got)
	}
}
