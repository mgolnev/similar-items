package similar

import "testing"

func hotpathPair(tb testing.TB) (base, cand ProductRow, cfg StrategyConfig) {
	tb.Helper()
	cfg = loadActiveStrategyConfig(tb)
	pool := benchPoolActive(tb, 1500, cfg)
	return pool[0], pool[42], cfg
}

func BenchmarkHotPath_categoryMatchHit(b *testing.B) {
	base, cand, cfg := hotpathPair(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = categoryMatchHit(base, cand, cfg)
	}
}

func BenchmarkHotPath_titlePrecomputed(b *testing.B) {
	base, cand, cfg := hotpathPair(b)
	cat := categoryMatchHit(base, cand, cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cappedTitleScoreSetsWithCategory(
			base.TitleNorm, cand.TitleNorm,
			base.TitleTokens, cand.TitleTokens,
			cat, cfg.Title,
		)
	}
}

func BenchmarkHotPath_attributeRulesPrecomputed(b *testing.B) {
	base, cand, cfg := hotpathPair(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scoreAttributeRules(base, cand, cfg)
	}
}

func BenchmarkHotPath_attributeRulesSlowPath(b *testing.B) {
	base, cand, cfg := hotpathPair(b)
	base.ruleFeats = nil
	cand.ruleFeats = nil
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scoreAttributeRules(base, cand, cfg)
	}
}

func BenchmarkHotPath_buildScoreNotes(b *testing.B) {
	base, cand, cfg := hotpathPair(b)
	cat := categoryMatchHit(base, cand, cfg)
	titleRes := cappedTitleScoreSetsWithCategory(
		base.TitleNorm, cand.TitleNorm,
		base.TitleTokens, cand.TitleTokens,
		cat, cfg.Title,
	)
	_, ruleScores := scoreAttributeRules(base, cand, cfg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildScoreNotes(cat, titleRes, ruleScores, cfg)
	}
}

func BenchmarkHotPath_scorePairDetailed(b *testing.B) {
	base, cand, cfg := hotpathPair(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scorePairDetailed(base, cand, cfg)
	}
}
