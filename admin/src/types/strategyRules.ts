export interface ColorSimilarityRule {
  raw_exact_score: number;
  primary_exact_score: number;
  alias_exact_score: number;
  family_score: number;
  shade_overlap_score: number;
  conflict_penalty: number;
  gate_on_conflict?: boolean | null;
}

export interface AttributeRuleScope {
  root_sections?: string[];
}

export interface AttributeRule {
  id: string;
  enabled: boolean;
  label_ru: string;
  source: 'attributes' | 'column' | 'computed' | string;
  key: string;
  match_type: string;
  weight: number;
  max_score: number;
  missing_policy: string;
  bonus_same?: number;
  penalty_conflict?: number;
  gate_on_conflict?: boolean;
  scope?: AttributeRuleScope;
  color?: ColorSimilarityRule;
}

export const DEFAULT_COLOR_SIMILARITY: ColorSimilarityRule = {
  raw_exact_score: 30,
  primary_exact_score: 24,
  alias_exact_score: 22,
  family_score: 18,
  shade_overlap_score: 6,
  conflict_penalty: -20,
  gate_on_conflict: null,
};

export const DEFAULT_COLOR_FAMILIES: Record<string, string[]> = {
  blue: ['синий', 'голубой', 'деним', 'индиго', 'navy', 'blue'],
  black: ['черный', 'чёрный', 'black'],
  white: ['белый', 'молочный', 'айвори', 'ivory', 'white'],
  gray: ['серый', 'графит', 'меланж', 'gray', 'grey'],
  beige: ['бежевый', 'кремовый', 'песочный', 'beige'],
  brown: ['коричневый', 'шоколадный', 'brown'],
  green: ['зеленый', 'зелёный', 'хаки', 'оливковый', 'green'],
  red: ['красный', 'бордовый', 'red'],
  pink: ['розовый', 'фуксия', 'pink'],
  purple: ['фиолетовый', 'сиреневый', 'лиловый', 'purple'],
  yellow: ['желтый', 'жёлтый', 'yellow'],
  orange: ['оранжевый', 'orange'],
};

export const DEFAULT_COLOR_FAMILY_PALETTE: Record<string, string> = {
  blue: '#2F5DA8',
  black: '#1F1F1F',
  white: '#F7F4ED',
  gray: '#9CA3AF',
  beige: '#D8C3A5',
  brown: '#7A4A2E',
  green: '#6B7F3A',
  red: '#C62828',
  pink: '#E78AA9',
  purple: '#8B5FBF',
  yellow: '#F2C94C',
  orange: '#E9852C',
};

export const MATCH_TYPES = [
  { id: 'exact', label: 'exact' },
  { id: 'token_overlap', label: 'token_overlap' },
  { id: 'set_overlap', label: 'set_overlap' },
  { id: 'plain_vs_decorated', label: 'plain_vs_decorated' },
  { id: 'color_similarity', label: 'color_similarity' },
] as const;

export const RULE_SOURCES = [
  { id: 'attributes', label: 'attributes (JSON фида)' },
  { id: 'column', label: 'column (поле БД)' },
  { id: 'computed', label: 'computed' },
] as const;

export function newAttributeRuleId(): string {
  return `rule_${Date.now().toString(36)}`;
}

export function createDefaultAttributeRule(): AttributeRule {
  return {
    id: newAttributeRuleId(),
    enabled: true,
    label_ru: 'Новое правило',
    source: 'attributes',
    key: '',
    match_type: 'exact',
    weight: 10,
    max_score: 10,
    missing_policy: 'no_score',
  };
}

export function createGenderPreset(weight = 20): AttributeRule {
  return {
    id: 'gender',
    enabled: true,
    label_ru: 'Пол / аудитория',
    source: 'column',
    key: 'gender',
    match_type: 'exact',
    weight,
    max_score: weight,
    missing_policy: 'no_score',
  };
}

export const DECOR_SET_ATTRIBUTE_KEY = '_decor_set';
export const PATTERN_SET_ATTRIBUTE_KEY = '_pattern_set';
export const MATERIAL_SET_ATTRIBUTE_KEY = '_material_set';
export const FABRIC_TYPE_SET_ATTRIBUTE_KEY = '_fabric_type_set';

export type SlotSetRulePreset = {
  id: string;
  canonKey: string;
  label_ru: string;
  weight: number;
  baseKey: string;
};

export const SLOT_SET_RULE_PRESETS: SlotSetRulePreset[] = [
  { id: 'decor', canonKey: DECOR_SET_ATTRIBUTE_KEY, label_ru: 'Декор (набор)', weight: 12, baseKey: 'декор' },
  { id: 'pattern', canonKey: PATTERN_SET_ATTRIBUTE_KEY, label_ru: 'Узор (набор)', weight: 10, baseKey: 'узор' },
  { id: 'material', canonKey: MATERIAL_SET_ATTRIBUTE_KEY, label_ru: 'Материал (набор)', weight: 10, baseKey: 'материал' },
  { id: 'fabric_type', canonKey: FABRIC_TYPE_SET_ATTRIBUTE_KEY, label_ru: 'Тип ткани (набор)', weight: 8, baseKey: 'тип ткани' },
];

export function isSlotSetParamKey(key: string, baseKey: string): boolean {
  const escaped = baseKey.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`^${escaped}\\d*$`, 'i').test(key.trim());
}

export function createSlotSetPreset(preset: SlotSetRulePreset): AttributeRule {
  return {
    id: preset.id,
    enabled: true,
    label_ru: preset.label_ru,
    source: 'attributes',
    key: preset.canonKey,
    match_type: 'set_overlap',
    weight: preset.weight,
    max_score: preset.weight,
    missing_policy: 'no_score',
  };
}

export function createDecorPreset(weight = 12): AttributeRule {
  return createSlotSetPreset(SLOT_SET_RULE_PRESETS[0]);
}

export function createColorSimilarityPreset(): AttributeRule {
  const w = 30;
  const color = { ...DEFAULT_COLOR_SIMILARITY };
  return {
    id: 'color_similarity',
    enabled: true,
    label_ru: 'Гибкий цвет (семейства)',
    source: 'column',
    key: 'color',
    match_type: 'color_similarity',
    weight: w,
    max_score: w,
    missing_policy: 'no_score',
    color,
  };
}
