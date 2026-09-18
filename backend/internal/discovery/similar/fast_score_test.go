package similar

import "testing"

func withoutNumericIDs(rows []ProductRow) []ProductRow {
	out := make([]ProductRow, len(rows))
	copy(out, rows)
	for i := range out {
		out[i].preparedIDs = nil
		out[i].groupKeyID = 0
		out[i].modelID = 0
		out[i].articleFamilyID = 0
		out[i].articlePrefixID = 0
		out[i].leafCategoryID = 0
		out[i].parentCategoryID = 0
		out[i].topBranchID = 0
		out[i].teenSubsectionID = 0
		out[i].teenProductClassID = 0
		out[i].categoryPartIDs = nil
		out[i].catalogLeafIDs = nil
	}
	return out
}

func assertSameRanking(t *testing.T, baseSKU string, want, got []scoredPair) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("base %s: length differs: want=%d got=%d", baseSKU, len(want), len(got))
	}
	for i := range want {
		if want[i].row.SKU != got[i].row.SKU ||
			want[i].score != got[i].score ||
			want[i].baseScoreWithoutTitle != got[i].baseScoreWithoutTitle {
			t.Fatalf(
				"base %s rank %d differs: want=%s/%d/%v got=%s/%d/%v",
				baseSKU,
				i,
				want[i].row.SKU,
				want[i].score,
				want[i].baseScoreWithoutTitle,
				got[i].row.SKU,
				got[i].score,
				got[i].baseScoreWithoutTitle,
			)
		}
	}
}

func TestFastPairScoreEquivalentToDetailedActiveStrategy(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	pool := benchPoolActive(t, 240, cfg)

	for i := 0; i < len(pool); i++ {
		for j := 0; j < len(pool); j += 7 {
			if i == j {
				continue
			}
			fast := scorePairFast(&pool[i], &pool[j], &cfg)
			detailed := scorePairDetailed(pool[i], pool[j], cfg)
			if fast.score != detailed.Score ||
				fast.baseScoreWithoutTitle != detailed.BaseScoreWithoutTitle {
				t.Fatalf(
					"pair %s→%s differs: fast=(%d,%v) detailed=(%d,%v)",
					pool[i].SKU,
					pool[j].SKU,
					fast.score,
					fast.baseScoreWithoutTitle,
					detailed.Score,
					detailed.BaseScoreWithoutTitle,
				)
			}
		}
	}
}

func TestFastRankingEquivalentToDetailedActiveStrategy(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	pool := benchPoolActive(t, 500, cfg)

	for _, baseIndex := range []int{0, 17, 233, 499} {
		fast := RankSimilarCandidates(pool[baseIndex], pool, cfg)
		detailed := rankSimilarCandidatesDetailed(pool[baseIndex], pool, cfg)
		assertSameRanking(t, pool[baseIndex].SKU, detailed, fast)
	}
}

func TestNumericPreparedRankingEquivalentToStringPath(t *testing.T) {
	cfg := loadActiveStrategyConfig(t)
	numericPool := benchPoolActive(t, 500, cfg)
	stringPool := withoutNumericIDs(numericPool)

	for _, baseIndex := range []int{0, 17, 233, 499} {
		numeric := RankSimilarCandidates(numericPool[baseIndex], numericPool, cfg)
		stringsOnly := RankSimilarCandidates(stringPool[baseIndex], stringPool, cfg)
		assertSameRanking(t, numericPool[baseIndex].SKU, stringsOnly, numeric)
	}
}

func BenchmarkRankSimilarCandidatesActiveStrategyFullCatalog(b *testing.B) {
	cfg := loadActiveStrategyConfig(b)
	pool := benchPoolActive(b, 11632, cfg)
	base := pool[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RankSimilarCandidates(base, pool, cfg)
	}
}
