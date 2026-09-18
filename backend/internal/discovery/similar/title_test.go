package similar

import "testing"

func TestCappedTitleScore_ignoreOneSidedExtraTokens(t *testing.T) {
	base := "Белая базовая футболка оверсайз из джерси"
	extraWordCases := []string{
		"Белая базовая футболка оверсайз из тонкого джерси",
		"Белая базовая футболка оверсайз из легкого джерси",
		"Белая базовая укороченная футболка оверсайз из джерси",
	}

	cfgOff := DefaultConfig().Title
	cfgOff.IgnoreModifierWords = false
	cfgOn := DefaultConfig().Title
	cfgOn.IgnoreModifierWords = true

	for _, cand := range extraWordCases {
		off := CappedTitleScore(base, cand, true, cfgOff)
		on := CappedTitleScore(base, cand, true, cfgOn)
		if on <= off {
			t.Fatalf("expected higher score for %q: off=%.1f on=%.1f", cand, off, on)
		}
		same := CappedTitleScore(base, base, true, cfgOn)
		if on+10 != same {
			t.Fatalf("expected on+10 == identical for %q: on=%.1f same=%.1f", cand, on, same)
		}
	}
}

func TestTitleTokenOverlap_missingBaseTokenStillPenalizes(t *testing.T) {
	cfg := DefaultConfig().Title
	base := "Белая базовая футболка из джерси"
	cand := "Базовая футболка из джерси"
	on := CappedTitleScore(base, cand, true, cfg)
	same := CappedTitleScore(base, base, true, cfg)
	if on >= same {
		t.Fatalf("missing base token should score lower: on=%.1f same=%.1f", on, same)
	}
}

func TestCappedTitleScore_featureWords(t *testing.T) {
	cfg := DefaultConfig().Title
	base := "Молочная спортивная футболка в рубчик с завязками"
	withPrint := "Молочная спортивная футболка оверсайз с принтом и лампасами"
	withSame := "Молочная спортивная футболка с завязками в рубчик"

	sameFeat := CappedTitleScore(base, withSame, true, cfg)
	diffFeat := CappedTitleScore(base, withPrint, true, cfg)
	if sameFeat <= diffFeat {
		t.Fatalf("expected higher score for matching features: same=%.1f diff=%.1f", sameFeat, diffFeat)
	}
}

func TestCappedTitleScore_modifierWordsDisabled(t *testing.T) {
	cfg := DefaultConfig().Title
	cfg.IgnoreModifierWords = false
	base := "Белая базовая футболка оверсайз из джерси"
	withThin := "Белая базовая футболка оверсайз из тонкого джерси"
	if CappedTitleScore(base, withThin, true, cfg) >= CappedTitleScore(base, base, true, cfg) {
		t.Fatal("without ignore flag thin jersey should score lower than identical")
	}
}

func TestCappedTitleScore_synonymsPureMatch(t *testing.T) {
	cfg := DefaultConfig().Title
	cfg.SynonymGroups = DefaultTitleSynonymGroups()
	cfg.SynonymIndex = BuildSynonymIndex(cfg.SynonymGroups)
	cfg.SynonymTokenScore = 5
	cfg.SynonymMax = 15

	syn := CappedTitleScore("Ветровка оверсайз", "Куртка оверсайз", true, cfg)
	same := CappedTitleScore("Куртка оверсайз", "Куртка оверсайз", true, cfg)
	unrelated := CappedTitleScore("Ветровка оверсайз", "Брюки оверсайз", true, cfg)
	if syn <= unrelated {
		t.Fatalf("synonym pair should score higher: syn=%.1f unrelated=%.1f", syn, unrelated)
	}
	// exact surface match already covered by jaccard/exact — synonym must not double-count
	exactSurface := CappedTitleScore("Куртка синяя", "Куртка красная", true, cfg)
	pureSyn := CappedTitleScore("Ветровка синяя", "Куртка красная", true, cfg)
	if pureSyn <= exactSurface {
		// pure synonym should still help vs unrelated colors case; at least equal or higher than no-synonym baseline
		t.Logf("exactSurface=%.1f pureSyn=%.1f same=%.1f", exactSurface, pureSyn, same)
	}
	noIdx := cfg
	noIdx.SynonymIndex = nil
	without := CappedTitleScore("Ветровка оверсайз", "Куртка оверсайз", true, noIdx)
	if syn <= without {
		t.Fatalf("synonym index must add score: with=%.1f without=%.1f", syn, without)
	}
}

func TestCappedTitleScore_synonymNoDoubleCountExactSurface(t *testing.T) {
	cfg := TitleConfig{
		Enabled: true, JaccardMax: 20, MinTokenLength: 3,
		SynonymTokenScore: 5, SynonymMax: 15,
		SynonymGroups: DefaultTitleSynonymGroups(),
	}
	cfg.SynonymIndex = BuildSynonymIndex(cfg.SynonymGroups)
	// одинаковая словоформа «куртка» — синонимный балл не начисляется
	withIdx := matchedSynonymScore(
		tokenSets("куртка синяя", cfg),
		tokenSets("куртка красная", cfg),
		cfg.SynonymIndex, cfg.SynonymTokenScore, cfg.SynonymMax,
	)
	if withIdx != 0 {
		t.Fatalf("exact surface in same group must not get synonym score, got %.1f", withIdx)
	}
	pure := matchedSynonymScore(
		tokenSets("ветровка синяя", cfg),
		tokenSets("куртка красная", cfg),
		cfg.SynonymIndex, cfg.SynonymTokenScore, cfg.SynonymMax,
	)
	if pure != 5 {
		t.Fatalf("pure synonym expected 5, got %.1f", pure)
	}
}

func TestCappedTitleScore_emptySynonymsFastPath(t *testing.T) {
	cfg := DefaultConfig().Title
	cfg.SynonymGroups = nil
	cfg.SynonymIndex = nil
	cfg.SynonymMax = 0
	a := CappedTitleScore("Футболка базовая", "Футболка оверсайз", true, cfg)
	cfg.SynonymMax = 15 // max без индекса всё равно 0
	b := CappedTitleScore("Футболка базовая", "Футболка оверсайз", true, cfg)
	if a != b {
		t.Fatalf("empty index must not change score: a=%.1f b=%.1f", a, b)
	}
}

func BenchmarkCappedTitleScore_noSynonyms(b *testing.B) {
	cfg := DefaultConfig().Title
	cfg.SynonymGroups = nil
	cfg.SynonymIndex = nil
	cfg.SynonymMax = 0
	base := "Белая базовая футболка оверсайз из джерси"
	cand := "Белая базовая футболка оверсайз из тонкого джерси"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CappedTitleScore(base, cand, true, cfg)
	}
}

func BenchmarkCappedTitleScore_withSynonyms(b *testing.B) {
	cfg := DefaultConfig().Title
	base := "Ветровка оверсайз синяя"
	cand := "Куртка оверсайз голубая"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CappedTitleScore(base, cand, true, cfg)
	}
}

func TestCappedTitleScore_featureGate_weakCategory(t *testing.T) {
	cfg := DefaultConfig().Title
	cfg.FeatureTokenScore = 5
	cfg.FeatureMax = 15
	base := "Ветровка принт"
	cand := "Юбка принт"

	leaf := categoryHit{OK: true, Kind: "leaf", Depth: 4}
	shallow := categoryHit{OK: true, Kind: "parent_shallow", Depth: 1, Common: "Девочки"}

	leafRes := CappedTitleScoreWithCategory(base, cand, leaf, cfg)
	shallowRes := CappedTitleScoreWithCategory(base, cand, shallow, cfg)
	if shallowRes.Score >= leafRes.Score {
		t.Fatalf("weak category should reduce title score: leaf=%d shallow=%d gate=%q",
			leafRes.Score, shallowRes.Score, shallowRes.Gate)
	}
	if shallowRes.Gate != "feature_blocked_weak_category" {
		t.Fatalf("gate=%q", shallowRes.Gate)
	}
}

func TestCappedTitleScore_canIgnoreCategoryMismatch(t *testing.T) {
	cfg := DefaultConfig().Title
	cfg.FeatureTokenScore = 5
	cfg.FeatureMax = 15
	base := "Молочный свитшот оверсайз с вышивкой"
	cand := "Бежевый свитшот оверсайз с вышивкой и принтом"
	weak := categoryHit{OK: true, Kind: "teen_subsection_class", Depth: 1}

	gated := CappedTitleScoreWithCategory(base, cand, weak, cfg)
	if gated.Gate != "feature_blocked_weak_category" {
		t.Fatalf("expected feature gate for weak category, got %+v", gated)
	}

	cfg.IgnoreCategoryMismatch = true
	ignored := CappedTitleScoreWithCategory(base, cand, weak, cfg)
	if ignored.Gate != "" {
		t.Fatalf("category gate must be disabled, got %+v", ignored)
	}
	if ignored.Score != gated.Score+cfg.FeatureTokenScore {
		t.Fatalf("expected restored feature score %d, gated=%d ignored=%d",
			cfg.FeatureTokenScore, gated.Score, ignored.Score)
	}

	noCategory := categoryHit{}
	uncapped := CappedTitleScoreWithCategory(base, cand, noCategory, cfg)
	if uncapped.Score <= cfg.NoCategoryCap {
		t.Fatalf("disabled category adjustment must bypass no_category_cap: %+v", uncapped)
	}
}
