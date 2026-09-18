import { AttributeRule } from '../types/strategyRules';
import {
  colorFamiliesForString,
  colorRuleScore,
  colorTokens,
  explainColorSimilarity,
  isPartialColorMatch,
} from './colorMatch';

const families: Record<string, string[]> = {
  blue: ['синий', 'голубой', 'деним', 'айс'],
  brown: ['коричневый', 'шоколадный'],
  white: ['молочный', 'белый'],
  black: ['черный'],
};

const rule: AttributeRule = {
  id: 'color_similarity',
  enabled: true,
  source: 'column',
  key: 'color',
  match_type: 'color_similarity',
  max_score: 30,
  color: {
    raw_exact_score: 30,
    primary_exact_score: 24,
    alias_exact_score: 22,
    family_score: 18,
    shade_overlap_score: 6,
    conflict_penalty: -50,
  },
};

describe('colorMatch', () => {
  it('разбивает color-токены по / и скобкам', () => {
    expect(colorTokens('синий /лайт айс/ (синий)')).toEqual(
      expect.arrayContaining(['синий', 'лайт', 'айс'])
    );
  });

  it('находит семейства по алиасам', () => {
    const fam = colorFamiliesForString('синий /лайт айс/', families);
    expect(fam.has('blue')).toBe(true);
  });

  it('same family: синий vs синий с другим shade даёт score > 0', () => {
    const e = explainColorSimilarity(
      'синий /лайт айс/ (синий)',
      'синий /медиум айс овердайт/ (синий)',
      families,
      rule
    );
    expect(e).not.toBeNull();
    expect(e!.finalScore).toBeGreaterThan(0);
  });

  it('multi-color vs single-color = 0', () => {
    const e = explainColorSimilarity('коричневый/молочный', 'коричневый', families, rule);
    expect(e!.matchPath).toBe('multi_mismatch');
    expect(e!.finalScore).toBe(0);
    expect(isPartialColorMatch(e)).toBe(true);
  });

  it('multi-color partial overlap — reduced score', () => {
    const full = explainColorSimilarity(
      'коричневый/молочный',
      'коричневый/белый',
      families,
      rule
    );
    const partial = explainColorSimilarity(
      'коричневый/молочный',
      'коричневый/черный',
      families,
      rule
    );
    expect(full!.finalScore).toBeGreaterThanOrEqual(18);
    expect(partial!.finalScore).toBeGreaterThan(0);
    expect(partial!.finalScore).toBeLessThan(18);
  });

  it('применяет penalty для конфликтующих цветовых семейств', () => {
    const e = explainColorSimilarity('черный', 'белый', families, rule);
    expect(e!.matchPath).toBe('conflict');
    expect(e!.conflict).toBe(true);
    expect(e!.finalScore).toBe(-50);

    const extendedFamilies = {
      ...families,
      beige: ['бежевый'],
      orange: ['оранжевый'],
      yellow: ['оранжевый', 'желтый'],
    };
    const mixed = explainColorSimilarity(
      'бежевый',
      'оранжевый/черный',
      extendedFamilies,
      rule
    );
    expect(mixed!.isCandMulti).toBe(true);
    expect(mixed!.matchPath).toBe('conflict');
    expect(mixed!.finalScore).toBe(-50);

    const unknown = explainColorSimilarity('бежевый', 'неизвестный цвет', extendedFamilies, rule);
    expect(unknown!.matchPath).toBe('none');
    expect(unknown!.finalScore).toBe(0);
  });

  it('общий alias нескольких семейств остаётся single-color', () => {
    const overlappingFamilies: Record<string, string[]> = {
      beige: ['бежевый', 'молочный'],
      white: ['белый', 'молочный'],
      brown: ['коричневый'],
    };
    const e = explainColorSimilarity(
      'бежевый',
      'молочный',
      overlappingFamilies,
      rule
    );

    expect(e!.baseFamilies).toEqual(['beige']);
    expect(e!.candFamilies).toEqual(expect.arrayContaining(['beige', 'white']));
    expect(e!.isBaseMulti).toBe(false);
    expect(e!.isCandMulti).toBe(false);
    expect(e!.matchPath).toBe('family');
    expect(e!.finalScore).toBe(18);

    const trueMulti = explainColorSimilarity(
      'коричневый/молочный',
      'коричневый',
      overlappingFamilies,
      rule
    );
    expect(trueMulti!.isBaseMulti).toBe(true);
    expect(trueMulti!.matchPath).toBe('multi_mismatch');
  });

  it('colorRuleScore читает rule_scores.color_similarity', () => {
    expect(colorRuleScore({ rule_scores: { color_similarity: 12 } })).toBe(12);
    expect(colorRuleScore({ rule_scores: { color_similarity: -20 } })).toBe(-20);
    expect(colorRuleScore({ color: 5 })).toBe(5);
  });
});
