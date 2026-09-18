import { SimilarStrategyDocument } from '../types/strategy';
import {
  buildPresetExportFilename,
  parsePresetImportText,
  serializePresetExport,
} from './presetTransfer';

const sampleStrategy: SimilarStrategyDocument = {
  schema_version: 1,
  enabled: true,
  weights: {
    category_leaf: 70,
    category_shared_leaf: 60,
    category_parent_deep: 50,
    category_parent: 40,
    category_parent_shallow: 10,
    category_branch: 5,
    gender_or_age_group: 0,
    article_prefix: 13,
    price: 0,
  },
  price: { range_percent: 5 },
  title: {
    enabled: true,
    max_score: 30,
    exact_score: 40,
    jaccard_max_score: 60,
    model_token_score: 20,
    model_max_score: 35,
    feature_token_score: 20,
    feature_max_score: 40,
    synonym_token_score: 15,
    synonym_max_score: 60,
    no_category_cap: 1,
    ignore_category_mismatch: false,
    min_token_length: 2,
    stopwords: ['clever'],
    model_words: ['slim'],
    feature_words: ['принт'],
    synonym_groups: [],
    ignore_modifier_words_enabled: true,
  },
  thresholds: {
    strict: { score_min: 80, base_without_title_min: 50, strategy: 'same_category_color_then_parent_fallback' },
    parent: { score_min: 50, base_without_title_min: 25, strategy: 'parent_category_fallback' },
    branch: { score_min: 20, strategy: 'branch_fallback' },
    popular: { enabled: true, strategy: 'popular_in_section' },
  },
  attribute_rules: [],
  color_family_dictionary: { blue: ['синий'] },
  color_family_palette: { blue: '#2F5DA8' },
  updated_reason: '',
};

describe('presetTransfer', () => {
  it('builds export filename from preset name', () => {
    expect(buildPresetExportFilename('0206', new Date('2026-07-20T10:00:00Z'))).toBe(
      'similar-preset-0206-20260720.json'
    );
  });

  it('serializes and parses wrapped export payload', () => {
    const raw = serializePresetExport('0206', sampleStrategy);
    const parsed = parsePresetImportText(raw);
    expect(parsed.name).toBe('0206');
    expect(parsed.strategy.weights.category_leaf).toBe(70);
    expect(parsed.strategy.color_family_dictionary.blue).toContain('синий');
  });

  it('parses API preset view shape', () => {
    const parsed = parsePresetImportText(
      JSON.stringify({
        id: 'p_local',
        name: '1707',
        is_production: false,
        strategy: sampleStrategy,
      })
    );
    expect(parsed.name).toBe('1707');
    expect(parsed.strategy.enabled).toBe(true);
  });

  it('parses bare strategy document', () => {
    const parsed = parsePresetImportText(JSON.stringify(sampleStrategy));
    expect(parsed.name).toBe('Импорт');
    expect(parsed.strategy.weights.category_branch).toBe(5);
  });

  it('rejects invalid json', () => {
    expect(() => parsePresetImportText('{bad')).toThrow(/JSON/i);
  });
});
