package recommendations

import (
	"context"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

const maxPublicLimit = 40

type SimilarFinder interface {
	FindSimilar(context.Context, similar.Params) (*similar.Result, error)
	FindSimilarBatch(context.Context, similar.BatchParams) (*similar.BatchResult, error)
}

type OutfitFinder interface {
	IsReady() bool
	FindCached(outfits.Params) *outfits.Result
	FindCachedBatch(outfits.BatchParams) *outfits.BatchResult
}

type ConfigSource interface {
	RecommendationConfig() Config
}

// Service composes independent recommendation sources at read time. Neither
// source cache nor its ranking is modified by this layer.
type Service struct {
	similar SimilarFinder
	outfits OutfitFinder
	config  ConfigSource
}

func NewService(similarFinder SimilarFinder, outfitFinder OutfitFinder, config ConfigSource) *Service {
	return &Service{similar: similarFinder, outfits: outfitFinder, config: config}
}

func (s *Service) Find(ctx context.Context, params Params) (*Result, error) {
	return s.find(ctx, params, nil)
}

// FindWithConfig is used by admin preview so an unsaved composition draft can
// be verified without changing the public behavior.
func (s *Service) FindWithConfig(ctx context.Context, params Params, cfg Config) (*Result, error) {
	cfg = NormalizeConfig(cfg)
	return s.find(ctx, params, &cfg)
}

func (s *Service) find(ctx context.Context, params Params, override *Config) (*Result, error) {
	limit := normalizeLimit(params.Limit)
	similarResult, err := s.similar.FindSimilar(ctx, similar.Params{
		ProductID:         params.ProductID,
		Section:           params.Section,
		Limit:             limit,
		ExcludeProductIDs: params.ExcludeProductIDs,
	})
	if err != nil {
		return nil, err
	}

	var outfitIDs []string
	warning := ""
	if s.showOutfitFirst(override) && s.outfits != nil {
		if !s.outfits.IsReady() {
			warning = "outfit source unavailable"
		} else {
			outfitResult := s.outfits.FindCached(outfits.Params{
				ProductID:         params.ProductID,
				Limit:             limit,
				ExcludeProductIDs: params.ExcludeProductIDs,
			})
			outfitIDs = outfitItemIDs(outfitResult.Items)
		}
	}

	return &Result{
		Items:   merge(outfitIDs, similarItemIDs(similarResult.Items), params.ProductID, params.ExcludeProductIDs, limit),
		Warning: warning,
	}, nil
}

func (s *Service) FindBatch(ctx context.Context, params BatchParams) (*BatchResult, error) {
	articles := normalizeIDs(params.ProductIDs)
	limit := normalizeLimit(params.Limit)
	similarResult, err := s.similar.FindSimilarBatch(ctx, similar.BatchParams{
		ProductIDs:        articles,
		Section:           params.Section,
		Limit:             limit,
		ExcludeProductIDs: params.ExcludeProductIDs,
	})
	if err != nil {
		return nil, err
	}

	outfitsByArticle := map[string]outfits.BatchResultItem{}
	batchWarning := ""
	if s.showOutfitFirst(nil) && s.outfits != nil {
		if !s.outfits.IsReady() {
			batchWarning = "outfit source unavailable"
		} else {
			outfitResult := s.outfits.FindCachedBatch(outfits.BatchParams{
				ProductIDs:        articles,
				Limit:             limit,
				ExcludeProductIDs: params.ExcludeProductIDs,
			})
			for _, row := range outfitResult.Results {
				outfitsByArticle[row.Article] = row
			}
		}
	}

	results := make([]BatchResultItem, 0, len(similarResult.Results))
	for _, row := range similarResult.Results {
		outfitRow := outfitsByArticle[row.Article]
		warning := batchWarning
		if outfitRow.Error != "" {
			warning = "outfit source unavailable"
		}
		results = append(results, BatchResultItem{
			Article: row.Article,
			Items: merge(
				outfitItemIDs(outfitRow.Items),
				similarItemIDs(row.Items),
				row.Article,
				params.ExcludeProductIDs,
				limit,
			),
			Error:   row.Error,
			Warning: warning,
		})
	}
	return &BatchResult{Results: results}, nil
}

func (s *Service) showOutfitFirst(override *Config) bool {
	if override != nil {
		return NormalizeConfig(*override).ShowOutfitFirst
	}
	return s.config != nil && NormalizeConfig(s.config.RecommendationConfig()).ShowOutfitFirst
}

func merge(outfitIDs, similarIDs []string, productID string, excluded []string, limit int) []Item {
	blocked := make(map[string]struct{}, len(excluded)+1)
	blocked[strings.TrimSpace(productID)] = struct{}{}
	for _, id := range excluded {
		blocked[strings.TrimSpace(id)] = struct{}{}
	}
	seen := make(map[string]struct{}, limit)
	items := make([]Item, 0, limit)
	appendIDs := func(ids []string, source string) {
		for _, raw := range ids {
			id := strings.TrimSpace(raw)
			if id == "" {
				continue
			}
			if _, skip := blocked[id]; skip {
				continue
			}
			if _, duplicate := seen[id]; duplicate {
				continue
			}
			seen[id] = struct{}{}
			items = append(items, Item{ID: id, Source: source})
			if len(items) >= limit {
				return
			}
		}
	}
	appendIDs(outfitIDs, SourceOutfit)
	if len(items) < limit {
		appendIDs(similarIDs, SourceSimilar)
	}
	return items
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return maxPublicLimit
	}
	if limit > maxPublicLimit {
		return maxPublicLimit
	}
	return limit
}

func normalizeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func similarItemIDs(items []similar.Item) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func outfitItemIDs(items []outfits.Item) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
