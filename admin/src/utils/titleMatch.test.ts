import { analyzeTitle, explainTitleScore, TitleConfigLike } from './titleMatch';

const cfg: TitleConfigLike = {
  enabled: true,
  exact_score: 10,
  jaccard_max_score: 20,
  model_token_score: 5,
  model_max_score: 15,
  feature_token_score: 5,
  feature_max_score: 15,
  synonym_token_score: 5,
  synonym_max_score: 0,
  no_category_cap: 10,
  min_token_length: 3,
  ignore_modifier_words_enabled: true,
  stopwords: ['clever', 'товар', 'арт', 'для', 'из'],
  model_words: ['палаццо', 'клеш', 'карго', 'оверсайз', 'широкий'],
  feature_words: ['принт', 'принтом', 'лампасы', 'лампасами', 'завязками', 'рубчик'],
  synonym_groups: [],
};

describe('titleMatch', () => {
  it('классифицирует токены: стоп/модельные/особенности/обычные/короткие', () => {
    const a = analyzeTitle('Футболка с принтом оверсайз ГД Clever', cfg);
    const byClass = (c: string) => a.tokens.filter((t) => t.cls === c).map((t) => t.text);
    expect(byClass('model')).toContain('оверсайз');
    expect(byClass('feature')).toContain('принтом');
    expect(byClass('stop')).toContain('clever');
    expect(byClass('short')).toContain('гд');
    expect(byClass('normal')).toContain('футболка');
  });

  it('exact-совпадение даёт exact_score и общий cap', () => {
    const e = explainTitleScore('Брюки', 'брюки', true, cfg);
    expect(e.exactMatch).toBe(true);
    expect(e.finalScore).toBe(30);
  });

  it('one-sided overlap: лишние слова кандидата не штрафуют', () => {
    const e = explainTitleScore('брюки широкие', 'брюки широкие модные новые', true, cfg);
    expect(e.overlapMode).toBe('one_sided');
    expect(e.finalScore).toBe(20);
  });

  it('feature words: совпадение особенностей повышает score', () => {
    const base = 'молочная футболка с завязками в рубчик';
    const same = 'молочная футболка в рубчик с завязками';
    const diff = 'молочная футболка с принтом и лампасами';
    const eSame = explainTitleScore(base, same, true, cfg);
    const eDiff = explainTitleScore(base, diff, true, cfg);
    expect(eSame.featureScore).toBeGreaterThan(0);
    expect(eSame.finalScore).toBeGreaterThan(eDiff.finalScore);
  });

  it('no_category_cap ограничивает при несовпадении категории', () => {
    const e = explainTitleScore('брюки палаццо', 'брюки палаццо', false, cfg);
    expect(e.noCategoryCapped).toBe(true);
    expect(e.finalScore).toBe(cfg.no_category_cap);
  });

  it('повторяет feature-gate backend для слабой teen-категории', () => {
    const base = 'Молочный свитшот поло оверсайз с вышивкой';
    const cand = 'Бежевый свитшот оверсайз из скубы с вышивкой и принтом';
    const e = explainTitleScore(
      base,
      cand,
      { ok: true, kind: 'teen_subsection_class' },
      { ...cfg, feature_words: ['вышивкой', 'принтом'], model_words: ['оверсайз'] }
    );

    expect(e.featureRawScore).toBe(5);
    expect(e.featureScore).toBe(0);
    expect(e.categoryGate).toBe('feature_blocked_weak_category');
  });

  it('может отключить снижение title-score из-за категории', () => {
    const base = 'Молочный свитшот оверсайз с вышивкой';
    const cand = 'Бежевый свитшот оверсайз с вышивкой и принтом';
    const gated = explainTitleScore(
      base,
      cand,
      { ok: true, kind: 'teen_subsection_class' },
      { ...cfg, feature_words: ['вышивкой', 'принтом'], model_words: ['оверсайз'] }
    );
    const ignored = explainTitleScore(
      base,
      cand,
      { ok: true, kind: 'teen_subsection_class' },
      {
        ...cfg,
        feature_words: ['вышивкой', 'принтом'],
        model_words: ['оверсайз'],
        ignore_category_mismatch: true,
      }
    );

    expect(ignored.categoryGate).toBe('');
    expect(ignored.featureScore).toBe(ignored.featureRawScore);
    expect(ignored.finalScore).toBe(gated.finalScore + 5);

    const uncapped = explainTitleScore(
      'брюки палаццо',
      'брюки палаццо',
      false,
      { ...cfg, ignore_category_mismatch: true }
    );
    expect(uncapped.noCategoryCapped).toBe(false);
    expect(uncapped.finalScore).toBeGreaterThan(cfg.no_category_cap);
  });

  it('выключенный title-matching = 0', () => {
    const e = explainTitleScore('брюки', 'брюки', true, { ...cfg, enabled: false });
    expect(e.finalScore).toBe(0);
  });

  it('синонимы: чистая пара даёт synonymScore, exact surface — нет', () => {
    const withSyn: TitleConfigLike = {
      ...cfg,
      synonym_max_score: 15,
      synonym_groups: [{ id: 'outerwear_light', words: ['ветровка', 'куртка'] }],
    };
    const pure = explainTitleScore('Ветровка оверсайз', 'Куртка оверсайз', true, withSyn);
    expect(pure.synonymScore).toBe(5);
    expect(pure.matchedSynonymGroups).toContain('outerwear_light');

    const sameWord = explainTitleScore('Куртка синяя', 'Куртка красная', true, withSyn);
    expect(sameWord.synonymScore).toBe(0);

    const off = explainTitleScore('Ветровка оверсайз', 'Куртка оверсайз', true, cfg);
    expect(off.synonymScore).toBe(0);
    expect(pure.finalScore).toBeGreaterThan(off.finalScore);
  });
});
