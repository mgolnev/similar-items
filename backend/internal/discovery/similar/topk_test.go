package similar

import (
	"reflect"
	"testing"
)

func rankTopKIDsReference(base *ProductRow, candidates []ProductRow, limit int, cfg StrategyConfig) []string {
	ranked := RankSimilarCandidates(*base, candidates, cfg)
	exclude := map[string]struct{}{base.SKU: {}, productGroupKey(*base): {}}
	items, _, _ := PickSimilarFromRanked(ranked, exclude, modelKey(*base), limit, cfg)
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	return ids
}

func TestRankTopKIDs_ExactlyMatchesFullSort(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	pool := benchPoolActive(t, 700, cfg)

	for _, limit := range []int{1, 12, 40, 100} {
		for _, baseIndex := range []int{0, 17, 233, 699} {
			base := &pool[baseIndex]
			want := rankTopKIDsReference(base, pool, limit, cfg)
			got := rankTopKIDs(base, pool, limit, &cfg)
			if !reflect.DeepEqual(want, got) {
				t.Fatalf(
					"base=%s limit=%d differs:\nwant=%v\ngot =%v",
					base.SKU,
					limit,
					want,
					got,
				)
			}
		}
	}
}

func TestRankTopKIDs_StringFallbackExactlyMatchesFullSort(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	pool := withoutNumericIDs(benchPoolActive(t, 300, cfg))

	for _, baseIndex := range []int{0, 29, 299} {
		base := &pool[baseIndex]
		want := rankTopKIDsReference(base, pool, 40, cfg)
		got := rankTopKIDs(base, pool, 40, &cfg)
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("base=%s differs:\nwant=%v\ngot =%v", base.SKU, want, got)
		}
	}
}

func TestRankTopKIDs_KeepsCandidateModelColorVariants(t *testing.T) {
	cfg := DefaultConfig()
	base := ProductRow{
		SKU: "base", GroupID: "GSW009030-2", Name: "Базовый джемпер",
		LeafCategory: "Женские джемперы", CategoryPath: `["Женщины","Женские джемперы"]`, ImageURL: "http://base",
	}
	candidates := []ProductRow{
		{SKU: "beige", GroupID: "GSW008762-1", Name: "Бежевый джемпер", LeafCategory: "Женские джемперы", CategoryPath: `["Женщины","Женские джемперы"]`, ImageURL: "http://beige"},
		{SKU: "gray", GroupID: "GSW008762-2", Name: "Серый джемпер", LeafCategory: "Женские джемперы", CategoryPath: `["Женщины","Женские джемперы"]`, ImageURL: "http://gray"},
	}
	space := newPreparedIDSpace(16)
	prepareNumericIDs(&base, space)
	for i := range candidates {
		prepareNumericIDs(&candidates[i], space)
	}

	ids := rankTopKIDs(&base, candidates, 2, &cfg)
	if len(ids) != 2 {
		t.Fatalf("expected both color variants, got %v", ids)
	}
	got := map[string]bool{ids[0]: true, ids[1]: true}
	if !got["GSW008762-1"] || !got["GSW008762-2"] {
		t.Fatalf("expected both GSW008762 variants, got %v", ids)
	}
}

func BenchmarkRankTopKIDs_ActiveStrategy(b *testing.B) {
	cfg := loadActiveStrategyConfig(b)
	pool := benchPoolActive(b, 1500, cfg)
	base := &pool[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rankTopKIDs(base, pool, idCacheTopK, &cfg)
	}
}

func BenchmarkRankTopKIDs_ActiveStrategyFullCatalog(b *testing.B) {
	cfg := loadActiveStrategyConfig(b)
	pool := benchPoolActive(b, 11632, cfg)
	base := &pool[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rankTopKIDs(base, pool, idCacheTopK, &cfg)
	}
}

func BenchmarkRankTopKIDsReference_ActiveStrategyFullCatalog(b *testing.B) {
	cfg := loadActiveStrategyConfig(b)
	pool := benchPoolActive(b, 11632, cfg)
	base := &pool[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rankTopKIDsReference(base, pool, idCacheTopK, cfg)
	}
}
