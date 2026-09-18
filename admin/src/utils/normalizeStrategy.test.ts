import { normalizeSearchStrategy, normalizeStrategy } from './normalizeStrategy';
import { SimilarStrategyDocument } from '../types/strategy';

describe('normalizeStrategy', () => {
  it('fills defaults for empty payload', () => {
    const doc = normalizeStrategy({});
    expect(doc.schema_version).toBe(1);
    expect(doc.weights.category_leaf).toBe(100);
    expect(doc.title.ignore_category_mismatch).toBe(false);
  });

  it('preserves disabled category adjustment for title score', () => {
    const baseline = normalizeStrategy({});
    const doc = normalizeStrategy({
      ...baseline,
      title: { ...baseline.title, ignore_category_mismatch: true },
    });
    expect(doc.title.ignore_category_mismatch).toBe(true);
  });

  it('normalizes search strategy defaults', () => {
    const doc = normalizeSearchStrategy({});
    expect(doc.search_synonyms).toEqual({});
    expect(doc.search.popularity_weight).toBe(0.3);
  });

  it('preserves custom search synonyms', () => {
    const doc = normalizeSearchStrategy({
      search_synonyms: { fit_slim: ['slim', 'слим'] },
    });
    expect(doc.search_synonyms.fit_slim).toHaveLength(2);
  });

  it('includes default color families', () => {
    const doc = normalizeStrategy({});
    expect(doc.color_family_dictionary.blue).toContain('синий');
    expect(doc.color_family_palette.blue).toMatch(/^#/);
  });

  it('migrates legacy gender weight to attribute rule', () => {
    const doc = normalizeStrategy({
      weights: { gender_or_age_group: 25 } as SimilarStrategyDocument['weights'],
    });
    expect(doc.weights.gender_or_age_group).toBe(0);
    const gender = doc.attribute_rules.find((r) => r.id === 'gender');
    expect(gender?.weight).toBe(25);
    expect(gender?.key).toBe('gender');
  });

  it('preserves attribute_rules and adds slot-set rules', () => {
    const doc = normalizeStrategy({
      attribute_rules: [
        {
          id: 'r1',
          enabled: true,
          label_ru: 'Тест',
          source: 'column',
          key: 'color',
          match_type: 'color_similarity',
          weight: 30,
          max_score: 30,
          missing_policy: 'no_score',
        },
      ],
    });
    expect(doc.attribute_rules).toHaveLength(5);
    expect(doc.attribute_rules[0].match_type).toBe('color_similarity');
    for (const id of ['decor', 'pattern', 'material', 'fabric_type']) {
      expect(doc.attribute_rules.find((r) => r.id === id)?.match_type).toBe('set_overlap');
    }
  });
});
