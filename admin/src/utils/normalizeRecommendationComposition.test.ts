import {
  DEFAULT_RECOMMENDATION_COMPOSITION,
  normalizeRecommendationComposition,
} from './normalizeRecommendationComposition';

describe('normalizeRecommendationComposition', () => {
  it('is opt-in by default', () => {
    expect(normalizeRecommendationComposition()).toEqual(DEFAULT_RECOMMENDATION_COMPOSITION);
  });

  it('keeps the enabled flag and normalizes the schema', () => {
    expect(
      normalizeRecommendationComposition({
        schema_version: 99,
        show_outfit_first: true,
        serve_composed_via_similar: true,
        updated_reason: 'MVP',
      })
    ).toEqual({
      schema_version: 1,
      show_outfit_first: true,
      serve_composed_via_similar: true,
      updated_reason: 'MVP',
    });
  });
});
