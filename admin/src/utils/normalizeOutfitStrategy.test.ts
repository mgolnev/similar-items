import { normalizeOutfitStrategy } from './normalizeOutfitStrategy';

describe('normalizeOutfitStrategy', () => {
  it('normalizes limits and removes duplicate keys', () => {
    expect(
      normalizeOutfitStrategy({
        schema_version: 0,
        enabled: true,
        attribute_keys: ['Сочетается с', 'Сочетается с', ''],
        max_items: 99,
        include_reverse_links: true,
      })
    ).toEqual({
      schema_version: 1,
      enabled: true,
      attribute_keys: ['Сочетается с'],
      max_items: 40,
      include_reverse_links: true,
      updated_reason: '',
    });
  });
});
