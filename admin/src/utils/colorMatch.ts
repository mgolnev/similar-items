import { AttributeRule, ColorSimilarityRule, DEFAULT_COLOR_SIMILARITY } from '../types/strategyRules';

export function normalizeColorField(s: string): string {
  return (s ?? '').trim().toLowerCase().replace(/ё/g, 'е');
}

const COLOR_TOKEN_SPLIT = /[/(),;|]+/;

export function colorTokens(color: string): string[] {
  let c = normalizeColorField(color);
  if (!c) return [];
  const parts = c.split(COLOR_TOKEN_SPLIT);
  const out: string[] = [];
  const seen = new Set<string>();
  for (const p of parts) {
    for (const w of p.split(/\s+/).filter(Boolean)) {
      const t = w.trim();
      if (Array.from(t).length < 2) continue;
      if (seen.has(t)) continue;
      seen.add(t);
      out.push(t);
    }
  }
  if (out.length === 0 && c) return [c];
  return out;
}

function tokenMatchesAlias(tok: string, alias: string): string {
  let a = normalizeColorField(alias);
  if (tok === a) return tok;
  if (tok.includes(a) || a.includes(tok)) {
    if (Array.from(a).length >= 3 || Array.from(tok).length >= 3) return a;
  }
  return '';
}

export function colorFamiliesForString(
  color: string,
  families: Record<string, string[]>
): Set<string> {
  return colorFamilyInfo(color, families).families;
}

function colorFamilySetsForString(
  color: string,
  families: Record<string, string[]>
): Set<string>[] {
  const found: Set<string>[] = [];
  const normalized = normalizeColorField(color);
  for (const rawSegment of normalized.split(COLOR_TOKEN_SPLIT)) {
    const segment = rawSegment.trim().split(/\s+/).filter(Boolean).join(' ');
    if (!segment) continue;

    const exact = new Set<string>();
    for (const [famID, aliases] of Object.entries(families)) {
      for (const alias of aliases) {
        if (normalizeColorField(alias) === segment) {
          exact.add(famID);
          break;
        }
      }
    }
    if (exact.size > 0) {
      found.push(exact);
      continue;
    }

    for (const tok of segment.split(/\s+/).filter(Boolean)) {
      const matched = new Set<string>();
      for (const [famID, aliases] of Object.entries(families)) {
        for (const alias of aliases) {
          const normalizedAlias = normalizeColorField(alias);
          if (/\s/.test(normalizedAlias)) continue;
          if (tokenMatchesAlias(tok, normalizedAlias)) {
            matched.add(famID);
            break;
          }
        }
      }
      if (matched.size === 0) continue;
      found.push(matched);
      break;
    }
  }
  return found;
}

function colorFamilyInfo(
  color: string,
  families: Record<string, string[]>
): { families: Set<string>; isMulti: boolean } {
  const segmentSets = colorFamilySetsForString(color, families);
  const all = new Set<string>();
  if (segmentSets.length === 0) return { families: all, isMulti: false };

  const common = new Set(segmentSets[0]);
  for (const segmentSet of segmentSets) {
    for (const famID of Array.from(segmentSet)) all.add(famID);
    for (const famID of Array.from(common)) {
      if (!segmentSet.has(famID)) common.delete(famID);
    }
  }
  return {
    families: all,
    isMulti: segmentSets.length > 1 && common.size === 0,
  };
}

function familySetSimilarityScore(a: Set<string>, b: Set<string>, maxScore: number): number {
  if (maxScore <= 0 || a.size === 0 || b.size === 0) return 0;
  let common = 0;
  for (const fam of Array.from(a)) {
    if (b.has(fam)) common += 1;
  }
  if (common === 0) return 0;
  const union = a.size + b.size - common;
  if (union <= 0) return 0;
  let score = Math.floor((maxScore * common) / union);
  if (score === 0) score = 1;
  return score;
}

export function shadeOverlapScore(a: string, b: string): number {
  const ta = colorTokens(a);
  const tb = colorTokens(b);
  if (ta.length === 0 || tb.length === 0) return 0;
  const setB = new Set(tb);
  let common = 0;
  for (const t of ta) {
    if (setB.has(t)) common += 1;
  }
  const union = ta.length + tb.length - common;
  if (union === 0) return 0;
  const ratio = common / union;
  if (ratio < 0.25) return 0;
  return Math.round(ratio * 100);
}

function primaryColorToken(color: string): string {
  const tokens = colorTokens(color);
  if (tokens.length === 0) return normalizeColorField(color);
  return tokens[0];
}

export interface ColorScoreExplain {
  baseColor: string;
  candColor: string;
  baseNorm: string;
  candNorm: string;
  baseFamilies: string[];
  candFamilies: string[];
  matchedFamilies: string[];
  baseTokens: string[];
  candTokens: string[];
  isBaseMulti: boolean;
  isCandMulti: boolean;
  matchPath: string;
  rawExactScore: number;
  primaryScore: number;
  aliasScore: number;
  familyScore: number;
  shadeBonus: number;
  conflict: boolean;
  rawBeforeCap: number;
  maxScore: number;
  finalScore: number;
}

export const PARTIAL_COLOR_MATCH_LABEL =
  'Частичное совпадение цвета — без бонуса и штрафа';

export function isPartialColorMatch(explain: ColorScoreExplain | null): boolean {
  return Boolean(
    explain?.matchPath === 'multi_mismatch' && explain.matchedFamilies.length > 0
  );
}

export function explainColorSimilarity(
  baseColor: string,
  candColor: string,
  families: Record<string, string[]>,
  rule?: AttributeRule | null
): ColorScoreExplain | null {
  const bc = baseColor.trim();
  const cc = candColor.trim();
  if (!bc || !cc || Object.keys(families).length === 0) return null;

  const cfg: ColorSimilarityRule = rule?.color
    ? { ...DEFAULT_COLOR_SIMILARITY, ...rule.color }
    : { ...DEFAULT_COLOR_SIMILARITY };

  const bNorm = normalizeColorField(bc);
  const cNorm = normalizeColorField(cc);
  const bInfo = colorFamilyInfo(bc, families);
  const cInfo = colorFamilyInfo(cc, families);
  const bFam = bInfo.families;
  const cFam = cInfo.families;
  const bMulti = bInfo.isMulti;
  const cMulti = cInfo.isMulti;
  const matchedFamilies = Array.from(bFam).filter((f) => cFam.has(f));
  const knownFamiliesDisjoint =
    bFam.size > 0 && cFam.size > 0 && matchedFamilies.length === 0;

  let score = 0;
  let matchPath = 'none';
  let conflict = false;
  let rawExactScore = 0;
  let primaryScore = 0;
  let aliasScore = 0;
  let familyScore = 0;
  let shadeBonus = 0;

  if (bNorm === cNorm) {
    score = cfg.raw_exact_score;
    rawExactScore = score;
    matchPath = 'raw_exact';
  } else if (knownFamiliesDisjoint) {
    score = cfg.conflict_penalty;
    conflict = true;
    matchPath = 'conflict';
  } else if (bMulti !== cMulti) {
    matchPath = 'multi_mismatch';
  } else if (bMulti && cMulti) {
    score = familySetSimilarityScore(bFam, cFam, cfg.family_score);
    familyScore = score;
    matchPath = 'multi_family';
  } else if (primaryColorToken(bc) === primaryColorToken(cc) && cfg.primary_exact_score > score) {
    score = cfg.primary_exact_score;
    primaryScore = score;
    matchPath = 'primary';
  } else {
    const bt = colorTokens(bc);
    const ct = colorTokens(cc);
    for (const t of bt) {
      for (const u of ct) {
        if (t === u && cfg.alias_exact_score > score) {
          score = cfg.alias_exact_score;
          aliasScore = score;
          matchPath = 'alias';
        }
      }
    }
  }

  if (!bMulti && !cMulti && bFam.size > 0 && cFam.size > 0) {
    for (const f of Array.from(bFam)) {
      if (cFam.has(f) && cfg.family_score > score) {
        score = cfg.family_score;
        familyScore = score;
        matchPath = 'family';
      }
    }
  }

  if (!conflict && score > 0) {
    const shade = shadeOverlapScore(bc, cc);
    if (shade > 0) {
      const add = Math.floor((cfg.shade_overlap_score * shade) / 100);
      if (add > 0) {
        score += add;
        shadeBonus = add;
      }
    }
  }

  if (score === 0) {
    return {
      baseColor: bc,
      candColor: cc,
      baseNorm: bNorm,
      candNorm: cNorm,
      baseFamilies: Array.from(bFam),
      candFamilies: Array.from(cFam),
      matchedFamilies,
      baseTokens: colorTokens(bc),
      candTokens: colorTokens(cc),
      isBaseMulti: bMulti,
      isCandMulti: cMulti,
      matchPath,
      rawExactScore,
      primaryScore,
      aliasScore,
      familyScore,
      shadeBonus,
      conflict,
      rawBeforeCap: 0,
      maxScore: rule?.max_score ?? rule?.weight ?? 0,
      finalScore: 0,
    };
  }

  let max = rule?.max_score ?? 0;
  if (max <= 0) max = rule?.weight ?? 0;
  const rawBeforeCap = score;
  if (max > 0 && score > max) score = max;

  return {
    baseColor: bc,
    candColor: cc,
    baseNorm: bNorm,
    candNorm: cNorm,
    baseFamilies: Array.from(bFam),
    candFamilies: Array.from(cFam),
    matchedFamilies,
    baseTokens: colorTokens(bc),
    candTokens: colorTokens(cc),
    isBaseMulti: bMulti,
    isCandMulti: cMulti,
    matchPath,
    rawExactScore,
    primaryScore,
    aliasScore,
    familyScore,
    shadeBonus,
    conflict,
    rawBeforeCap,
    maxScore: max,
    finalScore: score,
  };
}

export function colorRuleScore(breakdown: {
  color?: number;
  rule_scores?: Record<string, number>;
}): number {
  return breakdown.rule_scores?.color_similarity ?? breakdown.color ?? 0;
}
