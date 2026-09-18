export interface OutfitStrategyDocument {
  schema_version: number;
  enabled: boolean;
  attribute_keys: string[];
  max_items: number;
  include_reverse_links: boolean;
  updated_reason?: string;
}

export interface OutfitProductSummary {
  id: string;
  sku: string;
  name: string;
  image_url: string;
}

export interface OutfitReferenceDiagnostic {
  reference: string;
  resolved_id?: string;
  status: 'resolved' | 'not_found' | 'self_reference';
  direction: 'direct' | 'reverse';
}

export interface OutfitPreviewResult {
  base_product: OutfitProductSummary;
  items: OutfitProductSummary[];
  references: OutfitReferenceDiagnostic[];
  strategy: OutfitStrategyDocument;
}

export interface OutfitWarmStatus {
  running: boolean;
  pending: boolean;
  ready: boolean;
  phase: string;
  total: number;
  done: number;
  cache_entries: number;
  linked_cards: number;
  resolved_links: number;
  unresolved_links: number;
  serving_generation: number;
  target_generation: number;
  active_generation: number;
  last_warmed_at_unix: number;
  last_warmed_cards: number;
  last_error?: string;
}
