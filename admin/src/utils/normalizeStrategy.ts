import { SearchStrategyDocument, SimilarStrategyDocument } from '../types/strategy';
import {
  DEFAULT_COLOR_FAMILIES,
  DEFAULT_COLOR_FAMILY_PALETTE,
  SLOT_SET_RULE_PRESETS,
  createSlotSetPreset,
  isSlotSetParamKey,
} from '../types/strategyRules';
import { DEFAULT_FEATURE_WORDS } from './titleMatch';
import type { AttributeRule } from '../types/strategyRules';

function hasGenderAttributeRule(rules: AttributeRule[]): boolean {
  return rules.some((r) => {
    const id = r.id?.toLowerCase().trim();
    if (id === 'gender' || id === 'gender_or_age_group') return true;
    const key = r.key?.toLowerCase().trim();
    if (key !== 'gender' && key !== 'gender_or_age_group') return false;
    const src = (r.source || '').toLowerCase().trim();
    return src === '' || src === 'column';
  });
}

function hasSlotSetRule(rules: AttributeRule[], preset: (typeof SLOT_SET_RULE_PRESETS)[number]): boolean {
  return rules.some((r) => {
    const id = r.id?.toLowerCase().trim();
    if (id === preset.id) return true;
    if (r.key?.trim() === preset.canonKey) return true;
    return (
      r.key?.toLowerCase().trim() === preset.baseKey &&
      r.match_type?.toLowerCase().trim() === 'set_overlap'
    );
  });
}

function isLegacySlotSetRule(r: AttributeRule): boolean {
  const preset = SLOT_SET_RULE_PRESETS.find((p) => isSlotSetParamKey(r.key || '', p.baseKey));
  if (!preset) return false;
  const src = (r.source || '').toLowerCase().trim();
  if (src && src !== 'attributes') return false;
  const mt = (r.match_type || '').toLowerCase().trim();
  return mt === '' || mt === 'exact' || mt === 'token_overlap';
}

function migrateSlotSetRules(rules: AttributeRule[]): AttributeRule[] {
  let next = rules.filter((r) => !isLegacySlotSetRule(r));
  for (const preset of SLOT_SET_RULE_PRESETS) {
    if (!hasSlotSetRule(next, preset)) {
      next = [...next, createSlotSetPreset(preset)];
    }
  }
  return next;
}

function migrateGenderWeight(doc: SimilarStrategyDocument): SimilarStrategyDocument {
  const w = doc.weights.gender_or_age_group;
  if (w <= 0) return doc;
  if (hasGenderAttributeRule(doc.attribute_rules)) {
    return { ...doc, weights: { ...doc.weights, gender_or_age_group: 0 } };
  }
  return {
    ...doc,
    weights: { ...doc.weights, gender_or_age_group: 0 },
    attribute_rules: [
      ...doc.attribute_rules,
      {
        id: 'gender',
        enabled: true,
        label_ru: 'Пол / аудитория',
        source: 'column',
        key: 'gender',
        match_type: 'exact',
        weight: w,
        max_score: w,
        missing_policy: 'no_score',
      },
    ],
  };
}

function migrateSlotSetAttributeRules(doc: SimilarStrategyDocument): SimilarStrategyDocument {
  return {
    ...doc,
    attribute_rules: migrateSlotSetRules(doc.attribute_rules || []),
  };
}

/** Нормализует ответ API (null-поля, пустые синонимы). */
export function normalizeStrategy(raw: Partial<SimilarStrategyDocument> | null): SimilarStrategyDocument {
  const d = raw ?? ({} as Partial<SimilarStrategyDocument>);
  const base = migrateGenderWeight({
    schema_version: d.schema_version ?? 1,
    enabled: d.enabled ?? true,
    weights: {
      category_leaf: d.weights?.category_leaf ?? 100,
      category_shared_leaf: d.weights?.category_shared_leaf ?? 80,
      category_parent_deep: d.weights?.category_parent_deep ?? (d.weights?.category_parent ?? 50) + 20,
      category_parent: d.weights?.category_parent ?? 50,
      category_parent_shallow:
        d.weights?.category_parent_shallow ?? Math.max(25, Math.floor((d.weights?.category_parent ?? 50) / 2)),
      category_branch: d.weights?.category_branch ?? 25,
      gender_or_age_group: d.weights?.gender_or_age_group ?? 0,
      article_prefix: d.weights?.article_prefix ?? 12,
      price: d.weights?.price ?? 10,
    },
    price: { range_percent: d.price?.range_percent ?? 25 },
    title: {
      enabled: d.title?.enabled ?? true,
      max_score: d.title?.max_score ?? 30,
      exact_score: d.title?.exact_score ?? 10,
      jaccard_max_score: d.title?.jaccard_max_score ?? 20,
      model_token_score: d.title?.model_token_score ?? 5,
      model_max_score: d.title?.model_max_score ?? 15,
      feature_token_score: d.title?.feature_token_score ?? 5,
      feature_max_score: d.title?.feature_max_score ?? 15,
      synonym_token_score: d.title?.synonym_token_score ?? 5,
      // 0 = выключено: не меняем cap и скоринг существующих пресетов без синонимов
      synonym_max_score: d.title?.synonym_max_score ?? 0,
      no_category_cap: d.title?.no_category_cap ?? 10,
      ignore_category_mismatch: d.title?.ignore_category_mismatch ?? false,
      min_token_length: d.title?.min_token_length ?? 3,
      stopwords: d.title?.stopwords ?? [],
      model_words: d.title?.model_words ?? [],
      feature_words:
        d.title?.feature_words && d.title.feature_words.length > 0
          ? d.title.feature_words
          : [...DEFAULT_FEATURE_WORDS],
      synonym_groups: Array.isArray(d.title?.synonym_groups) ? d.title!.synonym_groups! : [],
      ignore_modifier_words_enabled: d.title?.ignore_modifier_words_enabled ?? true,
    },
    thresholds: {
      strict: {
        score_min: d.thresholds?.strict?.score_min ?? 80,
        base_without_title_min: d.thresholds?.strict?.base_without_title_min ?? 50,
        strategy: d.thresholds?.strict?.strategy ?? 'same_category_color_then_parent_fallback',
      },
      parent: {
        score_min: d.thresholds?.parent?.score_min ?? 50,
        base_without_title_min: d.thresholds?.parent?.base_without_title_min ?? 25,
        strategy: d.thresholds?.parent?.strategy ?? 'parent_category_fallback',
      },
      branch: {
        score_min: d.thresholds?.branch?.score_min ?? 25,
        strategy: d.thresholds?.branch?.strategy ?? 'branch_fallback',
      },
      popular: {
        enabled: d.thresholds?.popular?.enabled ?? true,
        strategy: d.thresholds?.popular?.strategy ?? 'popular_in_section',
      },
    },
    attribute_rules: Array.isArray(d.attribute_rules) ? d.attribute_rules : [],
    color_family_dictionary:
      d.color_family_dictionary && Object.keys(d.color_family_dictionary).length > 0
        ? d.color_family_dictionary
        : { ...DEFAULT_COLOR_FAMILIES },
    color_family_palette:
      d.color_family_palette && Object.keys(d.color_family_palette).length > 0
        ? { ...DEFAULT_COLOR_FAMILY_PALETTE, ...d.color_family_palette }
        : { ...DEFAULT_COLOR_FAMILY_PALETTE },
    updated_reason: d.updated_reason ?? '',
  });
  return migrateSlotSetAttributeRules(base);
}

export function normalizeSearchStrategy(
  raw: Partial<SearchStrategyDocument> | null
): SearchStrategyDocument {
  const d = raw ?? ({} as Partial<SearchStrategyDocument>);
  return {
    schema_version: d.schema_version ?? 1,
    enabled: d.enabled ?? true,
    search_synonyms: d.search_synonyms ?? {},
    color_family_dictionary:
      d.color_family_dictionary && Object.keys(d.color_family_dictionary).length > 0
        ? d.color_family_dictionary
        : { ...DEFAULT_COLOR_FAMILIES },
    search: {
      popularity_weight: d.search?.popularity_weight ?? 0.3,
      pins: Array.isArray(d.search?.pins) ? d.search!.pins : [],
      boosts: Array.isArray(d.search?.boosts) ? d.search!.boosts : [],
      product_type_dictionary: Array.isArray(d.search?.product_type_dictionary)
        ? d.search!.product_type_dictionary
        : [],
    },
    updated_reason: d.updated_reason ?? '',
  };
}
