import { featuresForSource, hintKeysForRule } from './featureCatalogHints';
import { FeatureCatalogEntry } from '../types/featureCatalog';

const sample: FeatureCatalogEntry[] = [
  { key: 'color', source: 'column', label_ru: 'Цвет', description: 'd' },
  { key: 'Цвет', source: 'attributes', label_ru: 'Цвет фид', description: 'd' },
];

describe('featureCatalogHints', () => {
  it('filters by source', () => {
    expect(featuresForSource(sample, 'column')).toHaveLength(1);
    expect(hintKeysForRule(sample, 'attributes')).toEqual(['Цвет']);
  });
});
