import { RecommendationCompositionConfig } from '../types/recommendationComposition';

export const DEFAULT_RECOMMENDATION_COMPOSITION: RecommendationCompositionConfig = {
  schema_version: 1,
  show_outfit_first: false,
  serve_composed_via_similar: false,
  updated_reason: '',
};

export function normalizeRecommendationComposition(
  raw?: Partial<RecommendationCompositionConfig> | null
): RecommendationCompositionConfig {
  return {
    schema_version: 1,
    show_outfit_first: Boolean(raw?.show_outfit_first),
    serve_composed_via_similar: Boolean(raw?.serve_composed_via_similar),
    updated_reason: String(raw?.updated_reason ?? ''),
  };
}
