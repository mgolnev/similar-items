import { OutfitStrategyDocument } from '../types/outfit';

export const DEFAULT_OUTFIT_STRATEGY: OutfitStrategyDocument = {
  schema_version: 1,
  enabled: true,
  attribute_keys: ['Сочетается с товаром', 'Сочетается с'],
  max_items: 12,
  include_reverse_links: false,
  updated_reason: '',
};

export function normalizeOutfitStrategy(raw?: Partial<OutfitStrategyDocument> | null): OutfitStrategyDocument {
  const keys = Array.from(
    new Set(
      (raw?.attribute_keys ?? DEFAULT_OUTFIT_STRATEGY.attribute_keys)
        .map((key) => String(key).trim())
        .filter(Boolean)
    )
  );
  const maxItems = Number(raw?.max_items ?? DEFAULT_OUTFIT_STRATEGY.max_items);
  return {
    schema_version: 1,
    enabled: raw?.enabled ?? true,
    attribute_keys: keys.length ? keys : [...DEFAULT_OUTFIT_STRATEGY.attribute_keys],
    max_items: Math.max(1, Math.min(40, Number.isFinite(maxItems) ? maxItems : 12)),
    include_reverse_links: Boolean(raw?.include_reverse_links),
    updated_reason: String(raw?.updated_reason ?? ''),
  };
}
