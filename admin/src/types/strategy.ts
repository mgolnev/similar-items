import type { AttributeRule, ColorSimilarityRule } from './strategyRules';

export type { AttributeRule, ColorSimilarityRule };

export interface TitleSynonymGroup {
  id: string;
  words: string[];
}

export interface SimilarStrategyDocument {
  schema_version: number;
  enabled: boolean;
  weights: {
    // Категория — каскад «первое совпадение побеждает» (срабатывает один уровень).
    category_leaf: number;
    category_shared_leaf: number;
    category_parent_deep: number;
    category_parent: number;
    category_parent_shallow: number;
    category_branch: number;
    // Аддитивные сигналы (складываются при совпадении).
    gender_or_age_group: number;
    article_prefix: number;
    price: number;
  };
  price: { range_percent: number };
  title: {
    enabled: boolean;
    max_score: number;
    exact_score: number;
    jaccard_max_score: number;
    model_token_score: number;
    model_max_score: number;
    feature_token_score: number;
    feature_max_score: number;
    synonym_token_score: number;
    synonym_max_score: number;
    no_category_cap: number;
    ignore_category_mismatch: boolean;
    min_token_length: number;
    stopwords: string[];
    model_words: string[];
    feature_words: string[];
    synonym_groups: TitleSynonymGroup[];
    ignore_modifier_words_enabled: boolean;
  };
  thresholds: {
    strict: { score_min: number; base_without_title_min: number; strategy: string };
    parent: { score_min: number; base_without_title_min: number; strategy: string };
    branch: { score_min: number; strategy: string };
    popular: { enabled: boolean; strategy: string };
  };
  attribute_rules: AttributeRule[];
  color_family_dictionary: Record<string, string[]>;
  color_family_palette: Record<string, string>;
  updated_reason: string;
}

export interface SearchStrategyDocument {
  schema_version: number;
  enabled: boolean;
  search_synonyms: Record<string, string[]>;
  color_family_dictionary: Record<string, string[]>;
  search: SearchConfigDocument;
  updated_reason: string;
}

export interface SearchPin {
  query: string;
  sku: string;
  position: number;
}

export interface SearchMatch {
  leaf_category?: string;
  product_type?: string;
  normalized_color?: string;
  attr_key?: string;
  attr_value?: string;
  sku?: string;
}

export interface SearchBoost {
  when: SearchMatch;
  multiplier: number;
  scope?: string;
}

export interface SearchConfigDocument {
  popularity_weight: number;
  pins: SearchPin[];
  boosts: SearchBoost[];
  product_type_dictionary: string[];
}

export interface SimilarScoreBreakdown {
  total: number;
  base_without_title: number;
  category: number;
  category_kind: string;
  category_common?: string;
  category_depth?: number;
  color: number;
  product_type: number;
  gender: number;
  fit_or_silhouette: number;
  season: number;
  article_prefix: number;
  price: number;
  popularity: number;
  title: number;
  title_raw?: number;
  title_gate?: string;
  score_notes?: string[];
  rule_scores?: Record<string, number>;
}

export interface SimilarRuleScoreDetail {
  score?: number;
  base_value?: string;
  cand_value?: string;
  status: string;
}

export interface SimilarExplainItem {
  id: string;
  sku: string;
  name: string;
  image_url: string;
  score: number;
  breakdown: SimilarScoreBreakdown;
  rule_details?: Record<string, SimilarRuleScoreDetail>;
}

export interface SimilarPreviewResult {
  base_product: {
    id: string;
    sku: string;
    name: string;
    leaf_category: string;
    image_url: string;
    rule_details?: Record<string, SimilarRuleScoreDetail>;
  };
  items: SimilarExplainItem[];
  ranked_candidates: SimilarExplainItem[];
  strategy: string;
  fallback_used: boolean;
}

export interface ZeroResultRow {
  id: number;
  query: string;
  section: string;
  created_at: string;
}
