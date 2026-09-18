import React, { useCallback, useState } from 'react';
import { ChevronDownIcon, ChevronRightIcon } from '@heroicons/react/24/outline';
import { NumberInput } from '../components/NumberInput';
import { FeatureCatalogEntry } from '../types/featureCatalog';
import { SimilarStrategyDocument } from '../types/strategy';
import { findFeature, hintKeysForRule } from '../utils/featureCatalogHints';
import { SimilarStrategyFeatureCatalog } from './SimilarStrategyFeatureCatalog';
import {
  AttributeRule,
  createColorSimilarityPreset,
  createDecorPreset,
  createDefaultAttributeRule,
  createGenderPreset,
  createSlotSetPreset,
  SLOT_SET_RULE_PRESETS,
  DEFAULT_COLOR_SIMILARITY,
  MATCH_TYPES,
  RULE_SOURCES,
} from '../types/strategyRules';

type Props = {
  doc: SimilarStrategyDocument;
  onChange: (doc: SimilarStrategyDocument) => void;
  featureCatalog: FeatureCatalogEntry[];
  catalogLoading?: boolean;
};

export const SimilarStrategyRulesTab: React.FC<Props> = ({
  doc,
  onChange,
  featureCatalog,
  catalogLoading = false,
}) => {
  const [activeRuleIndex, setActiveRuleIndex] = useState<number | null>(null);
  const [expandedRuleIds, setExpandedRuleIds] = useState<Set<string>>(() => new Set());
  const rules = doc.attribute_rules || [];

  const ruleKey = (rule: AttributeRule, index: number) => rule.id || `rule-${index}`;

  const toggleRuleExpanded = useCallback((id: string) => {
    setExpandedRuleIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const setRules = (next: AttributeRule[]) => {
    const nextIds = new Set(next.map((r, i) => ruleKey(r, i)));
    setExpandedRuleIds((prev) => new Set(Array.from(prev).filter((id) => nextIds.has(id))));
    onChange({ ...doc, attribute_rules: next });
  };

  const updateRule = (index: number, patch: Partial<AttributeRule>) => {
    const next = rules.map((r, i) => (i === index ? { ...r, ...patch } : r));
    setRules(next);
  };

  const removeRule = (index: number) => {
    setRules(rules.filter((_, i) => i !== index));
  };

  const addColorPreset = () => {
    const without = rules.filter((r) => r.id !== 'color_similarity');
    setRules([...without, createColorSimilarityPreset()]);
  };

  const addGenderPreset = () => {
    const without = rules.filter((r) => r.id !== 'gender');
    setRules([...without, createGenderPreset()]);
  };

  const addDecorPreset = () => {
    const without = rules.filter((r) => r.id !== 'decor' && r.key !== '_decor_set');
    setRules([...without, createDecorPreset()]);
  };

  const addAllSlotSetPresets = () => {
    const ids = new Set(SLOT_SET_RULE_PRESETS.map((p) => p.id));
    const canon = new Set(SLOT_SET_RULE_PRESETS.map((p) => p.canonKey));
    const without = rules.filter(
      (r) =>
        !ids.has(r.id) &&
        !canon.has(r.key) &&
        !SLOT_SET_RULE_PRESETS.some((p) => new RegExp(`^${p.baseKey.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\d*$`, 'i').test(r.key || ''))
    );
    setRules([...without, ...SLOT_SET_RULE_PRESETS.map(createSlotSetPreset)]);
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          className="btn-secondary text-sm"
          onClick={() => setRules([...rules, createDefaultAttributeRule()])}
        >
          + Добавить правило
        </button>
        <button type="button" className="btn-outline text-sm" onClick={addColorPreset}>
          Включить гибкий цвет
        </button>
        <button type="button" className="btn-outline text-sm" onClick={addGenderPreset}>
          Пол / аудитория
        </button>
        <button type="button" className="btn-outline text-sm" onClick={addDecorPreset}>
          Декор (набор)
        </button>
        <button type="button" className="btn-outline text-sm" onClick={addAllSlotSetPresets}>
          Наборы слотов (все)
        </button>
      </div>

      {rules.length === 0 && (
        <p className="text-sm text-slate-500 py-4 border border-dashed border-slate-200 rounded-lg text-center">
          Правил нет — similar использует только базовые веса категории, артикула и цены.
        </p>
      )}

      <div className="grid lg:grid-cols-[240px_1fr] gap-4">
        <SimilarStrategyFeatureCatalog
          features={featureCatalog}
          loading={catalogLoading}
          onPick={(entry) => {
            if (activeRuleIndex == null) {
              const next = [...rules, createDefaultAttributeRule()];
              const idx = next.length - 1;
              next[idx] = {
                ...next[idx],
                source: entry.source as AttributeRule['source'],
                key: entry.key,
                label_ru: entry.label_ru || entry.key,
                match_type:
                  entry.suggested_match_types?.[0] || next[idx].match_type,
              };
              setRules(next);
              setActiveRuleIndex(idx);
              return;
            }
            updateRule(activeRuleIndex, {
              source: entry.source as AttributeRule['source'],
              key: entry.key,
              label_ru: entry.label_ru || entry.key,
              match_type:
                entry.suggested_match_types?.[0] ||
                rules[activeRuleIndex]?.match_type ||
                'exact',
            });
          }}
        />
        <div className="space-y-3">
          {rules.map((rule, index) => {
            const id = ruleKey(rule, index);
            return (
            <RuleCard
              key={id}
              rule={rule}
              featureCatalog={featureCatalog}
              isActive={activeRuleIndex === index}
              expanded={expandedRuleIds.has(id)}
              onToggleExpanded={() => toggleRuleExpanded(id)}
              onFocus={() => setActiveRuleIndex(index)}
              onChange={(patch) => updateRule(index, patch)}
              onRemove={() => {
                removeRule(index);
                if (activeRuleIndex === index) setActiveRuleIndex(null);
                setExpandedRuleIds((prev) => {
                  const next = new Set(prev);
                  next.delete(id);
                  return next;
                });
              }}
            />
            );
          })}
        </div>
      </div>
    </div>
  );
};

const RuleCard: React.FC<{
  rule: AttributeRule;
  featureCatalog: FeatureCatalogEntry[];
  isActive: boolean;
  expanded: boolean;
  onToggleExpanded: () => void;
  onFocus: () => void;
  onChange: (patch: Partial<AttributeRule>) => void;
  onRemove: () => void;
}> = ({ rule, featureCatalog, isActive, expanded, onToggleExpanded, onFocus, onChange, onRemove }) => {
  const hints = hintKeysForRule(featureCatalog, rule.source);
  const meta = findFeature(featureCatalog, rule.source, rule.key);
  const listId = `rule-key-hints-${rule.id}`;
  const isColor = rule.match_type === 'color_similarity';
  const colorCfg = rule.color || DEFAULT_COLOR_SIMILARITY;
  const title = rule.label_ru?.trim() || rule.key?.trim() || rule.id;

  return (
    <div
      className={`border rounded-xl bg-white ${
        isActive ? 'border-violet-400 ring-1 ring-violet-200' : 'border-slate-200'
      }`}
      onFocus={onFocus}
    >
      <div className="flex flex-wrap items-center gap-2 p-3">
        <button
          type="button"
          onClick={onToggleExpanded}
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-slate-500 hover:bg-slate-100 hover:text-slate-800"
          aria-expanded={expanded}
          aria-label={expanded ? 'Свернуть правило' : 'Развернуть правило'}
        >
          {expanded ? (
            <ChevronDownIcon className="h-4 w-4" />
          ) : (
            <ChevronRightIcon className="h-4 w-4" />
          )}
        </button>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={rule.enabled}
            onChange={(e) => onChange({ enabled: e.target.checked })}
          />
          <span className="font-medium text-slate-800">{title}</span>
        </label>
        {!expanded && (
          <span className="truncate text-xs text-slate-500">
            {rule.key}
            {rule.match_type ? ` · ${rule.match_type}` : ''}
            {rule.max_score ? ` · max ${rule.max_score}` : ''}
          </span>
        )}
        <span className="text-xs font-mono text-slate-400">{rule.id}</span>
        <button type="button" onClick={onRemove} className="text-xs text-red-600 hover:underline ml-auto">
          Удалить
        </button>
      </div>

      {expanded && (
      <div className="space-y-3 border-t border-slate-100 px-4 pb-4 pt-3">
      <div className="grid sm:grid-cols-2 gap-3">
        <Field label="Название (RU)">
          <input
            type="text"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
            value={rule.label_ru}
            onChange={(e) => onChange({ label_ru: e.target.value })}
          />
        </Field>
        <Field label="Ключ поля">
          <input
            type="text"
            list={listId}
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
            value={rule.key}
            onFocus={onFocus}
            onChange={(e) => onChange({ key: e.target.value })}
            placeholder="начните вводить или выберите из списка"
          />
          <datalist id={listId}>
            {hints.map((h) => (
              <option key={h} value={h} />
            ))}
          </datalist>
          {meta && (
            <p className="text-[11px] text-slate-500 mt-1">
              {meta.description}
              {meta.column ? ` · колонка ${meta.column}` : ''}
              {meta.suggested_match_types?.length
                ? ` · match: ${meta.suggested_match_types.join(', ')}`
                : ''}
            </p>
          )}
        </Field>
        <Field label="Источник">
          <select
            className="w-full rounded-lg border border-slate-300 px-2 py-2 text-sm"
            value={rule.source}
            onChange={(e) => onChange({ source: e.target.value })}
          >
            {RULE_SOURCES.map((s) => (
              <option key={s.id} value={s.id}>
                {s.label}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Тип сопоставления">
          <select
            className="w-full rounded-lg border border-slate-300 px-2 py-2 text-sm"
            value={rule.match_type}
            onChange={(e) => onChange({ match_type: e.target.value })}
          >
            {MATCH_TYPES.map((m) => (
              <option key={m.id} value={m.id}>
                {m.label}
              </option>
            ))}
          </select>
        </Field>
        <Field label="weight">
          <NumberInput value={rule.weight} min={-50} max={200} onChange={(v) => onChange({ weight: v })} />
        </Field>
        <Field label="max_score">
          <NumberInput value={rule.max_score} min={0} max={200} onChange={(v) => onChange({ max_score: v })} />
        </Field>
        <Field label="missing_policy">
          <input
            type="text"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
            value={rule.missing_policy}
            onChange={(e) => onChange({ missing_policy: e.target.value })}
          />
        </Field>
        <Field label="scope.root_sections (через запятую)">
          <input
            type="text"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
            value={(rule.scope?.root_sections || []).join(', ')}
            onChange={(e) =>
              onChange({
                scope: {
                  root_sections: e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean),
                },
              })
            }
            placeholder="women, men"
          />
        </Field>
      </div>

      {isColor && (
        <div className="border-t border-slate-100 pt-3">
          <p className="text-xs font-semibold text-violet-800 mb-2">Параметры color_similarity</p>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
            {(
              [
                ['raw_exact_score', 'raw_exact'],
                ['primary_exact_score', 'primary'],
                ['alias_exact_score', 'alias'],
                ['family_score', 'family'],
                ['shade_overlap_score', 'shade'],
                ['conflict_penalty', 'penalty'],
              ] as const
            ).map(([key, label]) => (
              <Field key={key} label={label}>
                <NumberInput
                  value={colorCfg[key]}
                  min={-50}
                  max={100}
                  onChange={(v) =>
                    onChange({
                      color: { ...colorCfg, [key]: v },
                    })
                  }
                />
              </Field>
            ))}
          </div>
        </div>
      )}
      </div>
      )}
    </div>
  );
};

const Field: React.FC<{ label: string; children: React.ReactNode }> = ({ label, children }) => (
  <div>
    <label className="block text-xs font-medium text-slate-600 mb-1">{label}</label>
    {children}
  </div>
);
