// Клиентское зеркало backend-логики разбора названия
// (internal/discovery/similar/title.go). Используется в админке для
// визуализации: «как название разбирается по словам» и почему пара
// получила свой title-балл. Числа совпадают с серверным breakdown.title.

export type TitleTokenClass = 'stop' | 'model' | 'feature' | 'normal' | 'short';

export interface TitleToken {
  text: string;
  cls: TitleTokenClass;
  matched?: boolean;
  synonymMatched?: boolean;
}

export interface TitleAnalysis {
  raw: string;
  normalized: string;
  tokens: TitleToken[];
  normalSet: string[];
  modelSet: string[];
  featureSet: string[];
}

export interface TitleSynonymGroupLike {
  id: string;
  words: string[];
}

export interface TitleConfigLike {
  enabled: boolean;
  exact_score: number;
  jaccard_max_score: number;
  model_token_score: number;
  model_max_score: number;
  feature_token_score: number;
  feature_max_score: number;
  synonym_token_score?: number;
  synonym_max_score?: number;
  no_category_cap: number;
  ignore_category_mismatch?: boolean;
  min_token_length: number;
  stopwords: string[];
  model_words: string[];
  feature_words: string[];
  synonym_groups?: TitleSynonymGroupLike[];
  ignore_modifier_words_enabled: boolean;
}

export const DEFAULT_FEATURE_WORDS = [
  'принт', 'принтом', 'принтами',
  'лампас', 'лампасы', 'лампасами',
  'завязки', 'завязками', 'завязок',
  'рубчик', 'рубчика',
  'вышивка', 'вышивкой',
  'капюшон', 'капюшоном',
  'полоска', 'полоски', 'полосками',
  'страз', 'стразами',
  'блестки', 'блестками',
  'аппликация', 'аппликацией',
];

export const DEFAULT_TITLE_SYNONYM_GROUPS: TitleSynonymGroupLike[] = [
  { id: 'outerwear_light', words: ['ветровка', 'куртка'] },
  { id: 'fit_loose', words: ['свободные', 'широкие', 'широкий'] },
  { id: 'jacket_blazer', words: ['жакет', 'пиджак'] },
];

export interface TitleScoreExplain {
  base: TitleAnalysis;
  cand: TitleAnalysis;
  matchedNormal: string[];
  matchedModel: string[];
  matchedFeature: string[];
  matchedSynonymGroups: string[];
  exactMatch: boolean;
  overlap: number;
  overlapNum: number;
  overlapDen: number;
  overlapMode: 'one_sided' | 'jaccard';
  exactScore: number;
  jaccardScore: number;
  modelScore: number;
  featureRawScore: number;
  featureScore: number;
  synonymScore: number;
  rawScore: number;
  cap: number;
  noCategoryCapped: boolean;
  categoryGate: string;
  globallyCapped: boolean;
  finalScore: number;
  enabled: boolean;
}

export interface TitleCategoryContext {
  ok: boolean;
  kind?: string;
}

const NON_LETTER_NUM = /[^\p{L}\p{N}]+/gu;
const MULTI_SPACE = /\s+/g;

export function normalizeTitleText(input: string): string {
  let s = (input ?? '').trim().toLowerCase();
  s = s.replace(/ё/g, 'е');
  s = s.replace(NON_LETTER_NUM, ' ').replace(MULTI_SPACE, ' ');
  return s.trim();
}

function perTokenScore(cfg: TitleConfigLike, kind: 'model' | 'feature' | 'synonym'): number {
  const v =
    kind === 'model'
      ? cfg.model_token_score
      : kind === 'feature'
        ? cfg.feature_token_score
        : cfg.synonym_token_score ?? 0;
  return v > 0 ? v : 5;
}

function dictionaryScore(
  matchedCount: number,
  perToken: number,
  maxScore: number
): number {
  let score = matchedCount * perToken;
  if (maxScore > 0 && score > maxScore) {
    score = maxScore;
  }
  return score;
}

/** word → groupId; первое вхождение побеждает. */
export function buildSynonymIndex(groups: TitleSynonymGroupLike[] | undefined): Map<string, string> {
  const out = new Map<string, string>();
  for (const g of groups ?? []) {
    const id = (g.id || g.words.join('|')).trim();
    if (!id) continue;
    for (const w of g.words ?? []) {
      const key = normalizeTitleText(w);
      if (!key || out.has(key)) continue;
      out.set(key, id);
    }
  }
  return out;
}

export function analyzeTitle(raw: string, cfg: TitleConfigLike): TitleAnalysis {
  const normalized = normalizeTitleText(raw);
  const stops = new Set(cfg.stopwords ?? []);
  const models = new Set(cfg.model_words ?? []);
  const features = new Set(cfg.feature_words ?? []);
  const minLen = cfg.min_token_length ?? 0;

  const tokens: TitleToken[] = [];
  const normalSet = new Set<string>();
  const modelSet = new Set<string>();
  const featureSet = new Set<string>();

  for (const t of normalized.split(' ').filter(Boolean)) {
    if (stops.has(t)) {
      tokens.push({ text: t, cls: 'stop' });
      continue;
    }
    if (models.has(t)) {
      tokens.push({ text: t, cls: 'model' });
      modelSet.add(t);
      continue;
    }
    if (features.has(t)) {
      tokens.push({ text: t, cls: 'feature' });
      featureSet.add(t);
      continue;
    }
    if (Array.from(t).length >= minLen) {
      tokens.push({ text: t, cls: 'normal' });
      normalSet.add(t);
    } else {
      tokens.push({ text: t, cls: 'short' });
    }
  }

  return {
    raw,
    normalized,
    tokens,
    normalSet: Array.from(normalSet),
    modelSet: Array.from(modelSet),
    featureSet: Array.from(featureSet),
  };
}

function intersection(a: string[], b: Set<string>): string[] {
  return a.filter((x) => b.has(x));
}

function collectGroupSurfaces(
  tokens: string[],
  index: Map<string, string>
): Map<string, Set<string>> {
  const out = new Map<string, Set<string>>();
  for (const t of tokens) {
    const gid = index.get(t);
    if (!gid) continue;
    let set = out.get(gid);
    if (!set) {
      set = new Set();
      out.set(gid, set);
    }
    set.add(t);
  }
  return out;
}

function matchedSynonymScore(
  baseTokens: string[],
  candTokens: string[],
  index: Map<string, string>,
  perToken: number,
  maxScore: number
): { score: number; groups: string[] } {
  if (index.size === 0 || maxScore <= 0) {
    return { score: 0, groups: [] };
  }
  const baseGroups = collectGroupSurfaces(baseTokens, index);
  if (baseGroups.size === 0) return { score: 0, groups: [] };
  const candGroups = collectGroupSurfaces(candTokens, index);
  if (candGroups.size === 0) return { score: 0, groups: [] };

  let score = 0;
  const groups: string[] = [];
  baseGroups.forEach((baseWords, gid) => {
    if (score >= maxScore) return;
    const candWords = candGroups.get(gid);
    if (!candWords) return;
    let exact = false;
    baseWords.forEach((w) => {
      if (candWords.has(w)) exact = true;
    });
    if (exact) return;
    groups.push(gid);
    score += perToken;
  });
  if (score > maxScore) score = maxScore;
  return { score, groups };
}

export function explainTitleScore(
  baseName: string,
  candName: string,
  categoryInput: boolean | TitleCategoryContext,
  cfg: TitleConfigLike
): TitleScoreExplain {
  const category: TitleCategoryContext =
    typeof categoryInput === 'boolean'
      ? { ok: categoryInput, kind: categoryInput ? 'leaf' : '' }
      : categoryInput;
  const base = analyzeTitle(baseName, cfg);
  const cand = analyzeTitle(candName, cfg);

  const candNormal = new Set(cand.normalSet);
  const candModel = new Set(cand.modelSet);
  const candFeature = new Set(cand.featureSet);
  const matchedNormal = intersection(base.normalSet, candNormal);
  const matchedModel = intersection(base.modelSet, candModel);
  const matchedFeature = intersection(base.featureSet, candFeature);

  const exactMatch = base.normalized !== '' && base.normalized === cand.normalized;
  const ignore = Boolean(cfg.ignore_modifier_words_enabled);
  const overlapMode: 'one_sided' | 'jaccard' = ignore ? 'one_sided' : 'jaccard';

  const inter = matchedNormal.length;
  let overlap = 0;
  let overlapDen = 0;
  if (base.normalSet.length !== 0 || cand.normalSet.length !== 0) {
    if (ignore) {
      overlapDen = base.normalSet.length;
      overlap = overlapDen > 0 ? inter / overlapDen : 0;
    } else {
      overlapDen = base.normalSet.length + cand.normalSet.length - inter;
      overlap = overlapDen > 0 ? inter / overlapDen : 0;
    }
  }

  const enabled = Boolean(cfg.enabled);
  const emptyTitles = base.normalized === '' || cand.normalized === '';

  let exactScore = 0;
  let jaccardScore = 0;
  let modelScore = 0;
  let featureRawScore = 0;
  let featureScore = 0;
  let synonymScore = 0;
  let matchedSynonymGroups: string[] = [];
  let rawScore = 0;
  let noCategoryCapped = false;
  let globallyCapped = false;
  let finalFloat = 0;
  let categoryGate = '';

  const synonymMax = cfg.synonym_max_score ?? 0;
  const synonymIndex =
    synonymMax > 0 ? buildSynonymIndex(cfg.synonym_groups) : new Map<string, string>();

  const cap =
    (cfg.jaccard_max_score ?? 0) +
    (cfg.model_max_score ?? 0) +
    (cfg.feature_max_score ?? 0) +
    (cfg.exact_score ?? 0) +
    synonymMax;

  if (enabled && !emptyTitles) {
    exactScore = exactMatch ? cfg.exact_score ?? 0 : 0;
    jaccardScore = overlap * (cfg.jaccard_max_score ?? 0);
    modelScore = dictionaryScore(
      matchedModel.length,
      perTokenScore(cfg, 'model'),
      cfg.model_max_score ?? 0
    );
    featureRawScore = dictionaryScore(
      matchedFeature.length,
      perTokenScore(cfg, 'feature'),
      cfg.feature_max_score ?? 0
    );
    featureScore = featureRawScore;
    if (!cfg.ignore_category_mismatch && featureRawScore > 0) {
      switch (category.kind) {
        case 'leaf':
        case 'shared_leaf':
        case 'parent_deep':
          break;
        case 'parent':
          featureScore *= 0.5;
          categoryGate = 'feature_half_mid_parent';
          break;
        default:
          featureScore = 0;
          categoryGate = 'feature_blocked_weak_category';
      }
    }
    const allBase = [...base.normalSet, ...base.modelSet, ...base.featureSet];
    const allCand = [...cand.normalSet, ...cand.modelSet, ...cand.featureSet];
    const syn = matchedSynonymScore(
      allBase,
      allCand,
      synonymIndex,
      perTokenScore(cfg, 'synonym'),
      synonymMax
    );
    synonymScore = syn.score;
    matchedSynonymGroups = syn.groups;
    rawScore = exactScore + jaccardScore + modelScore + featureScore + synonymScore;

    finalFloat = rawScore;
    if (
      !cfg.ignore_category_mismatch &&
      !category.ok &&
      finalFloat > (cfg.no_category_cap ?? 0)
    ) {
      finalFloat = cfg.no_category_cap ?? 0;
      noCategoryCapped = true;
      if (!categoryGate) categoryGate = 'no_category_cap';
    }
    if (finalFloat > cap) {
      finalFloat = cap;
      globallyCapped = true;
    }
  }

  const finalScore = Math.round(finalFloat);
  const synonymTokenSet = new Set<string>();
  if (matchedSynonymGroups.length > 0) {
    for (const g of cfg.synonym_groups ?? []) {
      if (!matchedSynonymGroups.includes(g.id)) continue;
      for (const w of g.words) synonymTokenSet.add(normalizeTitleText(w));
    }
  }

  base.tokens = base.tokens.map((t) => ({
    ...t,
    matched:
      (t.cls === 'normal' && candNormal.has(t.text)) ||
      (t.cls === 'model' && candModel.has(t.text)) ||
      (t.cls === 'feature' && candFeature.has(t.text)),
    synonymMatched: synonymTokenSet.has(t.text),
  }));
  cand.tokens = cand.tokens.map((t) => ({
    ...t,
    matched:
      (t.cls === 'normal' && new Set(base.normalSet).has(t.text)) ||
      (t.cls === 'model' && new Set(base.modelSet).has(t.text)) ||
      (t.cls === 'feature' && new Set(base.featureSet).has(t.text)),
    synonymMatched: synonymTokenSet.has(t.text),
  }));

  return {
    base,
    cand,
    matchedNormal,
    matchedModel,
    matchedFeature,
    matchedSynonymGroups,
    exactMatch,
    overlap,
    overlapNum: inter,
    overlapDen,
    overlapMode,
    exactScore,
    jaccardScore,
    modelScore,
    featureRawScore,
    featureScore,
    synonymScore,
    rawScore,
    cap,
    noCategoryCapped,
    categoryGate,
    globallyCapped,
    finalScore,
    enabled,
  };
}
