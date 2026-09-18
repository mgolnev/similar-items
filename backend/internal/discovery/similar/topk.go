package similar

import (
	"sort"
)

// scoredPairBetter задаёт тот же полный порядок, что sortScoredPairs:
// больший score выше, при равенстве меньший SKU выше.
func scoredPairBetter(a, b scoredPair) bool {
	if a.score != b.score {
		return a.score > b.score
	}
	return a.row.SKU < b.row.SKU
}

// worstFirstScoredHeap держит худший элемент выбранного top-K в корне.
// Благодаря этому каждый новый кандидат сравнивается только с границей top-K.
type worstFirstScoredHeap []scoredPair

func scoredPairWorse(a, b scoredPair) bool {
	return scoredPairBetter(b, a)
}

func heapUpWorstFirst(h worstFirstScoredHeap, index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if !scoredPairWorse(h[index], h[parent]) {
			return
		}
		h[index], h[parent] = h[parent], h[index]
		index = parent
	}
}

func heapDownWorstFirst(h worstFirstScoredHeap, index int) {
	for {
		left := index*2 + 1
		if left >= len(h) {
			return
		}
		worst := left
		right := left + 1
		if right < len(h) && scoredPairWorse(h[right], h[left]) {
			worst = right
		}
		if !scoredPairWorse(h[worst], h[index]) {
			return
		}
		h[index], h[worst] = h[worst], h[index]
		index = worst
	}
}

func addTopK(h *worstFirstScoredHeap, pair scoredPair, limit int) {
	if limit <= 0 {
		return
	}
	if len(*h) < limit {
		*h = append(*h, pair)
		heapUpWorstFirst(*h, len(*h)-1)
		return
	}
	if scoredPairBetter(pair, (*h)[0]) {
		(*h)[0] = pair
		heapDownWorstFirst(*h, 0)
	}
}

type cascadeTopK struct {
	limit   int
	cfg     *StrategyConfig
	strict  worstFirstScoredHeap
	parent  worstFirstScoredHeap
	branch  worstFirstScoredHeap
	popular worstFirstScoredHeap
}

func newCascadeTopK(limit int, cfg *StrategyConfig) *cascadeTopK {
	capacity := limit
	if capacity < 0 {
		capacity = 0
	}
	return &cascadeTopK{
		limit:   limit,
		cfg:     cfg,
		strict:  make(worstFirstScoredHeap, 0, capacity),
		parent:  make(worstFirstScoredHeap, 0, capacity),
		branch:  make(worstFirstScoredHeap, 0, capacity),
		popular: make(worstFirstScoredHeap, 0, capacity),
	}
}

func (s *cascadeTopK) add(pair scoredPair) {
	th := s.cfg.Thresholds
	if pair.score >= th.StrictScoreMin &&
		pair.baseScoreWithoutTitle >= float64(th.StrictBaseWithoutTitle) {
		addTopK(&s.strict, pair, s.limit)
	}
	if pair.score >= th.ParentScoreMin &&
		pair.baseScoreWithoutTitle >= float64(th.ParentBaseWithoutTitle) {
		addTopK(&s.parent, pair, s.limit)
	}
	if pair.score >= th.BranchScoreMin {
		addTopK(&s.branch, pair, s.limit)
	}
	addTopK(&s.popular, pair, s.limit)
}

// pick зеркалит pickFromScored: берём первый уровень каскада, который способен
// наполнить limit, иначе следующий. Каждый heap уже содержит точный top-K
// своего уровня; перед выдачей восстанавливаем production-порядок.
func (s *cascadeTopK) pick() []scoredPair {
	selected := s.strict
	if len(selected) < s.limit {
		selected = s.parent
	}
	if len(selected) < s.limit {
		selected = s.branch
	}
	if len(selected) < s.limit {
		selected = s.popular
	}
	sort.Slice(selected, func(i, j int) bool {
		return scoredPairBetter(selected[i], selected[j])
	})
	return selected
}

func canUseNumericModelIDs(base *ProductRow, candidates []ProductRow) bool {
	if base.preparedIDs == nil || base.modelID == 0 {
		return false
	}
	for i := range candidates {
		if !sharePreparedIDSpace(base, &candidates[i]) || candidates[i].modelID == 0 {
			return false
		}
	}
	return true
}

// rankTopKIDs выполняет production warm-путь без полной сортировки:
//  1. считает score каждой допустимой пары;
//  2. исключает цветомодификации запрошенной модели;
//  3. поддерживает точный top-K отдельно для каждого уровня каскада, не
//     схлопывая цветомодификации остальных моделей.
//
// Семантика эквивалентна RankSimilarCandidates + PickSimilarFromRanked, но
// сложность сортировки уменьшается с O(N log N) до O(N log K), где K=100.
func rankTopKIDs(base *ProductRow, candidates []ProductRow, limit int, cfg *StrategyConfig) []string {
	if limit <= 0 || len(candidates) == 0 {
		return nil
	}

	selector := newCascadeTopK(limit, cfg)
	if canUseNumericModelIDs(base, candidates) {
		for i := range candidates {
			cand := &candidates[i]
			if skipSimilarCandidateRows(base, cand, nil) || cand.modelID == base.modelID {
				continue
			}
			fast := scorePairFast(base, cand, cfg)
			selector.add(scoredPair{
				row:                   cand,
				score:                 fast.score,
				baseScoreWithoutTitle: fast.baseScoreWithoutTitle,
			})
		}
	} else {
		baseModel := modelKey(*base)
		for i := range candidates {
			cand := &candidates[i]
			if skipSimilarCandidateRows(base, cand, nil) {
				continue
			}
			model := modelKey(*cand)
			if model == baseModel {
				continue
			}
			fast := scorePairFast(base, cand, cfg)
			selector.add(scoredPair{
				row:                   cand,
				score:                 fast.score,
				baseScoreWithoutTitle: fast.baseScoreWithoutTitle,
			})
		}
	}

	picked := selector.pick()
	ids := make([]string, len(picked))
	for i := range picked {
		ids[i] = productGroupKeyPtr(picked[i].row)
	}
	return ids
}
