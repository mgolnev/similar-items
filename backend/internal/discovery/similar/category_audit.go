package similar

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const (
	defaultCategoryAuditTopN      = 30
	defaultCategoryAuditMinSample = 5
)

// CategoryAuditParams — параметры массового аудита категорий для контент-менеджеров.
type CategoryAuditParams struct {
	Section   string
	TopN      int     // размер выборки кандидатов (как в preview ranked_candidates)
	MaxAvg    float64 // 0 = без фильтра; иначе только avg <= MaxAvg
	MinSample int     // минимум кандидатов в выборке
}

// CategoryAuditRow — одна строка отчёта.
type CategoryAuditRow struct {
	Article      string  `json:"article"`
	Name         string  `json:"name"`
	LeafCategory string  `json:"leaf_category"`
	AvgCategory  float64 `json:"avg_category"`
	FiredPct     float64 `json:"fired_pct"`
	SampleSize   int     `json:"sample_size"`
	DominantKind string  `json:"dominant_kind"`
}

type categoryScored struct {
	score int
	kind  string
}

// AuditCategoryHealth считает для каждого товара пула средний балл категории
// по топ‑N кандидатам, отсортированным по категорийному каскаду
// (leaf → shared_leaf → parent → branch). Без title/атрибутов — быстро и прозрачно
// для контент-менеджеров.
//
// section=all — как в preview: каждый товар считается в пуле своей root-секции
// (women/men/kids/teens), а не против всего каталога.
func (s *Service) AuditCategoryHealth(ctx context.Context, params CategoryAuditParams) ([]CategoryAuditRow, error) {
	section := strings.TrimSpace(strings.ToLower(params.Section))
	if section == "" {
		section = "all"
	}
	topN := params.TopN
	if topN <= 0 {
		topN = defaultCategoryAuditTopN
	}
	minSample := params.MinSample
	if minSample <= 0 {
		minSample = defaultCategoryAuditMinSample
	}

	sections := []string{section}
	if section == "all" {
		sections = []string{"women", "men", "kids", "teens"}
	}

	var out []CategoryAuditRow
	seenArticle := make(map[string]struct{})
	for _, sec := range sections {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		rows, err := s.auditCategorySection(ctx, sec, topN, minSample)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if params.MaxAvg > 0 && row.AvgCategory > params.MaxAvg {
				continue
			}
			if _, dup := seenArticle[row.Article]; dup {
				continue
			}
			seenArticle[row.Article] = struct{}{}
			out = append(out, row)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].AvgCategory != out[j].AvgCategory {
			return out[i].AvgCategory < out[j].AvgCategory
		}
		return out[i].Article < out[j].Article
	})
	return out, nil
}

func (s *Service) auditCategorySection(
	ctx context.Context,
	section string,
	topN, minSample int,
) ([]CategoryAuditRow, error) {
	cfg := s.activeConfig()
	pool, err := s.preparedPool(ctx, section, normalizedPoolLimit(cfg.PoolLimit), cfg)
	if err != nil {
		return nil, fmt.Errorf("load similar pool %s: %w", section, err)
	}
	if len(pool) == 0 {
		return nil, nil
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}

	type job struct {
		idx int
		row ProductRow
	}
	jobs := make(chan job, workers*4)
	results := make([]CategoryAuditRow, len(pool))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			top := make([]categoryScored, 0, topN)
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				row, ok := summarizeCategoryAudit(j.row, pool, cfg, topN, minSample, top[:0])
				if ok {
					results[j.idx] = row
				}
			}
		}()
	}

	for i := range pool {
		if ctx.Err() != nil {
			break
		}
		jobs <- job{idx: i, row: pool[i]}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out := make([]CategoryAuditRow, 0, len(pool))
	for _, row := range results {
		if row.Article != "" {
			out = append(out, row)
		}
	}
	return out, nil
}

// summarizeCategoryAudit держит только топ‑N кандидатов (без полной сортировки пула).
func summarizeCategoryAudit(
	base ProductRow,
	pool []ProductRow,
	cfg StrategyConfig,
	topN, minSample int,
	top []categoryScored,
) (CategoryAuditRow, bool) {
	seenGroups := make(map[string]struct{}, 64)
	seenGroups[productGroupKey(base)] = struct{}{}
	candidates := 0
	minKept := 0

	// Предрасчёт базы — иначе categoryMatch/sharedCatalogLeaf аллоцируют на каждую пару.
	baseParts := categoryParts(base)
	baseLeaf := strings.TrimSpace(base.LeafCategory)
	baseParent := rowParentCategory(base, baseParts)
	baseBranch := rowTopBranch(base, baseParts)
	baseCatalogKeys := catalogLeafKeys(base)

	for i := range pool {
		cand := pool[i]
		if skipSimilarCandidate(base, cand, nil) {
			continue
		}
		gkey := productGroupKey(cand)
		if _, dup := seenGroups[gkey]; dup {
			continue
		}
		seenGroups[gkey] = struct{}{}
		candidates++

		score, kind := categoryMatchPrepared(baseLeaf, baseParent, baseBranch, baseParts, baseCatalogKeys, cand, cfg)
		item := categoryScored{score: score, kind: kind}

		if len(top) < topN {
			top = append(top, item)
			if len(top) == topN {
				sort.Slice(top, func(a, b int) bool { return top[a].score > top[b].score })
				minKept = top[topN-1].score
			}
			continue
		}
		if score <= minKept {
			continue
		}
		// заменить худший и поддерживать порядок по убыванию score
		top[topN-1] = item
		sort.Slice(top, func(a, b int) bool { return top[a].score > top[b].score })
		minKept = top[topN-1].score
	}

	if candidates < minSample || len(top) < minSample {
		return CategoryAuditRow{}, false
	}

	n := len(top)
	sum := 0
	fired := 0
	kindCounts := map[string]int{}
	for _, p := range top {
		sum += p.score
		if p.score > 0 {
			fired++
		}
		if p.kind != "" {
			kindCounts[p.kind]++
		}
	}

	return CategoryAuditRow{
		Article:      displayArticle(base),
		Name:         base.Name,
		LeafCategory: base.LeafCategory,
		AvgCategory:  float64(sum) / float64(n),
		FiredPct:     100 * float64(fired) / float64(n),
		SampleSize:   n,
		DominantKind: dominantCategoryKind(kindCounts),
	}, true
}

func categoryMatchPrepared(
	baseLeaf, baseParent, baseBranch string,
	baseParts []string,
	baseCatalogKeys map[string]struct{},
	cand ProductRow,
	cfg StrategyConfig,
) (score int, kind string) {
	base := ProductRow{
		LeafCategory:   baseLeaf,
		ParentCategory: baseParent,
		TopBranch:      baseBranch,
		CategoryParts:  baseParts,
	}
	if len(baseCatalogKeys) > 0 {
		labels := make([]string, 0, len(baseCatalogKeys))
		for k := range baseCatalogKeys {
			labels = append(labels, k)
		}
		base.CatalogLeafLabels = labels
	}
	hit := categoryMatchHit(base, cand, cfg)
	return hit.Score, hit.Kind
}

func dominantCategoryKind(counts map[string]int) string {
	bestKind := ""
	bestN := 0
	order := []string{"leaf", "shared_leaf", "parent_deep", "parent", "parent_shallow", "branch"}
	for _, k := range order {
		if n := counts[k]; n > bestN {
			bestN = n
			bestKind = k
		}
	}
	for k, n := range counts {
		if n > bestN {
			bestN = n
			bestKind = k
		}
	}
	return bestKind
}

var categoryKindLabelRU = map[string]string{
	"leaf":            "лист",
	"shared_leaf":     "общий лист",
	"parent_deep":     "родитель deep",
	"parent":          "родитель mid",
	"parent_shallow":  "родитель мелкий",
	"branch":          "ветка",
}

// WriteCategoryAuditCSV пишет отчёт в CSV (UTF-8 с BOM для Excel).
func WriteCategoryAuditCSV(w io.Writer, rows []CategoryAuditRow) error {
	if _, err := io.WriteString(w, "\ufeff"); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	if err := cw.Write([]string{
		"Артикул",
		"Название",
		"Категория",
		"Ср. балл категории",
		"Сработало %",
		"Выборка",
		"Доминирующий тип",
	}); err != nil {
		return err
	}
	for _, row := range rows {
		kind := row.DominantKind
		if ru, ok := categoryKindLabelRU[kind]; ok {
			kind = ru
		}
		if err := cw.Write([]string{
			row.Article,
			row.Name,
			row.LeafCategory,
			fmt.Sprintf("%.1f", row.AvgCategory),
			fmt.Sprintf("%.0f", row.FiredPct),
			fmt.Sprintf("%d", row.SampleSize),
			kind,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
