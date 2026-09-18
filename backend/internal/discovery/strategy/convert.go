package strategy

import "github.com/mgolnev/similar-items/backend/internal/discovery/similar"

// ToSimilarConfig преобразует документ в runtime-конфиг similar.
func (d Document) ToSimilarConfig() similar.StrategyConfig {
	cfg := similar.StrategyConfig{
		PriceRangePercent: d.Price.RangePercent,
	}
	// Пул кандидатов — весь каталог (PoolLimit=0 → полный каталог в similar).
	if cfg.PriceRangePercent <= 0 {
		cfg.PriceRangePercent = 25
	}

	cfg.Weights.CategoryLeaf = d.Weights.CategoryLeaf
	cfg.Weights.CategorySharedLeaf = d.Weights.CategorySharedLeaf
	if cfg.Weights.CategorySharedLeaf <= 0 {
		cfg.Weights.CategorySharedLeaf = 80
	}
	cfg.Weights.CategoryParent = d.Weights.CategoryParent
	cfg.Weights.CategoryParentDeep = d.Weights.CategoryParentDeep
	cfg.Weights.CategoryParentShallow = d.Weights.CategoryParentShallow
	if cfg.Weights.CategoryParentDeep <= 0 && cfg.Weights.CategoryParent > 0 {
		cfg.Weights.CategoryParentDeep = cfg.Weights.CategoryParent + 20
		if cfg.Weights.CategoryLeaf > 0 && cfg.Weights.CategoryParentDeep >= cfg.Weights.CategoryLeaf {
			cfg.Weights.CategoryParentDeep = cfg.Weights.CategoryLeaf - 1
		}
	}
	if cfg.Weights.CategoryParentShallow <= 0 && cfg.Weights.CategoryParent > 0 {
		cfg.Weights.CategoryParentShallow = cfg.Weights.CategoryParent / 2
		if cfg.Weights.CategoryBranch > 0 && cfg.Weights.CategoryParentShallow < cfg.Weights.CategoryBranch {
			cfg.Weights.CategoryParentShallow = cfg.Weights.CategoryBranch
		}
	}
	cfg.Weights.CategoryBranch = d.Weights.CategoryBranch
	cfg.Weights.Color = d.Weights.Color
	cfg.Weights.ProductType = d.Weights.ProductType
	cfg.Weights.GenderOrAgeGroup = d.Weights.GenderOrAgeGroup
	cfg.Weights.FitOrSilhouette = d.Weights.FitOrSilhouette
	cfg.Weights.SeasonOrCollection = d.Weights.SeasonOrCollection
	cfg.Weights.ArticlePrefix = d.Weights.ArticlePrefix
	cfg.Weights.Price = d.Weights.Price
	cfg.Weights.PopularityMax = d.Weights.PopularityMax

	cfg.Title = similar.TitleConfig{
		Enabled:                d.Title.Enabled,
		ExactScore:             d.Title.ExactScore,
		JaccardMax:             d.Title.JaccardMaxScore,
		ModelMax:               d.Title.ModelMaxScore,
		ModelTokenScore:        d.Title.ModelTokenScore,
		FeatureMax:             d.Title.FeatureMaxScore,
		FeatureTokenScore:      d.Title.FeatureTokenScore,
		SynonymMax:             d.Title.SynonymMaxScore,
		SynonymTokenScore:      d.Title.SynonymTokenScore,
		NoCategoryCap:          d.Title.NoCategoryCap,
		IgnoreCategoryMismatch: d.Title.IgnoreCategoryMismatch,
		MinTokenLength:         d.Title.MinTokenLength,
		IgnoreModifierWords:    d.Title.IgnoreModifierWordsEnabled,
		Stopwords:              append([]string(nil), d.Title.Stopwords...),
		ModelWords:             append([]string(nil), d.Title.ModelWords...),
		FeatureWords:           append([]string(nil), d.Title.FeatureWords...),
		SynonymGroups:          copySynonymGroups(d.Title.SynonymGroups),
	}
	if cfg.Title.ModelTokenScore <= 0 {
		cfg.Title.ModelTokenScore = 5
	}
	if cfg.Title.FeatureTokenScore <= 0 {
		cfg.Title.FeatureTokenScore = 5
	}
	if cfg.Title.SynonymTokenScore <= 0 {
		cfg.Title.SynonymTokenScore = 5
	}
	if cfg.Title.SynonymMax > 0 && len(cfg.Title.SynonymGroups) > 0 {
		cfg.Title.SynonymIndex = similar.BuildSynonymIndex(cfg.Title.SynonymGroups)
	}

	cfg.Thresholds.StrictScoreMin = d.Thresholds.Strict.ScoreMin
	cfg.Thresholds.StrictBaseWithoutTitle = d.Thresholds.Strict.BaseWithoutTitleMin
	cfg.Thresholds.StrictStrategy = d.Thresholds.Strict.Strategy
	cfg.Thresholds.ParentScoreMin = d.Thresholds.Parent.ScoreMin
	cfg.Thresholds.ParentBaseWithoutTitle = d.Thresholds.Parent.BaseWithoutTitleMin
	cfg.Thresholds.ParentStrategy = d.Thresholds.Parent.Strategy
	cfg.Thresholds.BranchScoreMin = d.Thresholds.Branch.ScoreMin
	cfg.Thresholds.BranchStrategy = d.Thresholds.Branch.Strategy
	cfg.Thresholds.PopularStrategy = d.Thresholds.Popular.Strategy

	cfg.Rules = d.toRulesConfig()

	cfg.Blocking = similar.BlockingConfig{Enabled: false, MinCandidates: 60}
	if d.Blocking != nil {
		cfg.Blocking.Enabled = d.Blocking.Enabled
		if d.Blocking.MinCandidates > 0 {
			cfg.Blocking.MinCandidates = d.Blocking.MinCandidates
		}
	}
	return cfg
}

func copySynonymGroups(in []SynonymGroup) []similar.SynonymGroup {
	if len(in) == 0 {
		return nil
	}
	out := make([]similar.SynonymGroup, 0, len(in))
	for _, g := range in {
		out = append(out, similar.SynonymGroup{
			ID:    g.ID,
			Words: append([]string(nil), g.Words...),
		})
	}
	return out
}

func (d Document) toRulesConfig() similar.RulesConfig {
	out := similar.RulesConfig{
		ColorFamilies: d.ColorFamilyDictionary,
	}
	if len(out.ColorFamilies) == 0 {
		out.ColorFamilies = DefaultColorFamilyDictionary()
	}
	for _, r := range d.AttributeRules {
		if !r.Enabled {
			continue
		}
		rc := similar.AttributeRuleConfig{
			ID: r.ID, Enabled: r.Enabled, LabelRu: r.LabelRu,
			Source: r.Source, Key: r.Key, MatchType: r.MatchType,
			Weight: r.Weight, MaxScore: r.MaxScore, MissingPolicy: r.MissingPolicy,
		}
		if r.Scope != nil {
			rc.ScopeSections = append([]string(nil), r.Scope.RootSections...)
		}
		if r.Color != nil {
			rc.Color = &similar.ColorSimilarityConfig{
				RawExactScore: r.Color.RawExactScore, PrimaryExactScore: r.Color.PrimaryExactScore,
				AliasExactScore: r.Color.AliasExactScore, FamilyScore: r.Color.FamilyScore,
				ShadeOverlapScore: r.Color.ShadeOverlapScore, ConflictPenalty: r.Color.ConflictPenalty,
			}
		}
		out.Rules = append(out.Rules, rc)
	}
	return out
}
