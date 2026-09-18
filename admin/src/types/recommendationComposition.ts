export interface RecommendationCompositionConfig {
  schema_version: number;
  show_outfit_first: boolean;
  serve_composed_via_similar: boolean;
  updated_reason?: string;
}

export type RecommendationSource = 'outfit' | 'similar';

export interface RecommendationCompositionItem {
  id: string;
  source: RecommendationSource;
}

export interface RecommendationCompositionPreview {
  items: RecommendationCompositionItem[];
  warning?: string;
}
