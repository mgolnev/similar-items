import React, { useState } from 'react';
import { BeakerIcon } from '@heroicons/react/24/outline';
import { AttributeRule } from '../types/strategyRules';
import { SimilarExplainItem, SimilarPreviewResult, SimilarRuleScoreDetail, SimilarScoreBreakdown } from '../types/strategy';
import { TitleConfigLike, explainTitleScore } from '../utils/titleMatch';
import {
  PARTIAL_COLOR_MATCH_LABEL,
  explainColorSimilarity,
  isPartialColorMatch,
} from '../utils/colorMatch';
import { TitleScorePanel } from '../components/TitleTokens';
import { SignalContributionSummary } from '../components/SignalContributionSummary';
import { productColorSizeArticle, productShopHref } from '../utils/productArticle';

const SECTIONS = [
  { id: 'women', label: 'Женщины' },
  { id: 'men', label: 'Мужчины' },
  { id: 'kids', label: 'Дети' },
  { id: 'teens', label: 'Подростки' },
  { id: 'all', label: 'Все' },
];

const CATEGORY_KIND: Record<string, string> = {
  leaf: 'лист',
  shared_leaf: 'общий leaf',
  parent_deep: 'родитель deep',
  parent: 'родитель mid',
  parent_shallow: 'родитель мелкий',
  branch: 'ветка',
};

const TITLE_GATE_HINT: Record<string, string> = {
  feature_half_mid_parent: 'feature ×½',
  feature_blocked_weak_category: 'feature=0',
  no_category_cap: 'no_category_cap',
};

const PREVIEW_PRODUCT_IMAGE_CLASS =
  'h-24 w-auto max-w-24 object-contain rounded shrink-0';

type Props = {
  section: string;
  productId: string;
  limit: number;
  loading: boolean;
  result: SimilarPreviewResult | null;
  onSectionChange: (v: string) => void;
  onProductIdChange: (v: string) => void;
  onLimitChange: (v: number) => void;
  onRun: () => void;
  attributeRules?: AttributeRule[];
  colorFamilies?: Record<string, string[]>;
  titleConfig?: TitleConfigLike;
};

export const SimilarStrategyPreviewTab: React.FC<Props> = ({
  section,
  productId,
  limit,
  loading,
  result,
  onSectionChange,
  onProductIdChange,
  onLimitChange,
  onRun,
  attributeRules = [],
  colorFamilies = {},
  titleConfig,
}) => {
  const enabledRules = attributeRules.filter((r) => r.enabled);
  const baseName = result?.base_product.name ?? '';
  return (
  <div className="space-y-4">
    <div className="flex flex-wrap gap-3 items-end">
      <div>
        <label className="block text-xs font-medium text-slate-600 mb-1">Секция пула</label>
        <select
          value={section}
          onChange={(e) => onSectionChange(e.target.value)}
          className="rounded-lg border border-slate-300 text-sm px-2 py-1.5"
        >
          {SECTIONS.map((s) => (
            <option key={s.id} value={s.id}>
              {s.label}
            </option>
          ))}
        </select>
      </div>
      <div className="flex-1 min-w-[200px]">
        <label className="block text-xs font-medium text-slate-600 mb-1">SKU / group_id</label>
        <input
          type="text"
          placeholder="например GAC024650-2 красный/белый"
          value={productId}
          onChange={(e) => onProductIdChange(e.target.value)}
          className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
        />
      </div>
      <div>
        <label className="block text-xs font-medium text-slate-600 mb-1">Лимит</label>
        <input
          type="number"
          min={1}
          max={40}
          value={limit}
          onChange={(e) => onLimitChange(Number(e.target.value) || 12)}
          className="w-20 rounded-lg border border-slate-300 px-2 py-1.5 text-sm"
        />
      </div>
      <button
        type="button"
        onClick={onRun}
        disabled={loading}
        className="btn-primary text-sm flex items-center gap-1"
      >
        <BeakerIcon className="w-4 h-4" />
        {loading ? 'Считаем…' : 'Проверить'}
      </button>
    </div>

    {result && (
      <>
        <div className="flex flex-wrap gap-4 items-center p-3 bg-slate-50 rounded-lg border border-slate-200 text-sm">
          {result.base_product.image_url && (
            <img
              src={result.base_product.image_url}
              alt=""
              className={PREVIEW_PRODUCT_IMAGE_CLASS}
            />
          )}
          <div>
            <a
              href={productShopHref(result.base_product)}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium text-slate-900 hover:text-primary-600 hover:underline"
            >
              {result.base_product.name}
            </a>
            <p className="text-xs text-slate-500 font-mono">
              {productColorSizeArticle(result.base_product)}
            </p>
            {result.base_product.leaf_category && (
              <p className="text-xs text-slate-500">Категория: {result.base_product.leaf_category}</p>
            )}
          </div>
          {result.fallback_used && (
            <div className="ml-auto max-w-sm text-right text-xs text-amber-700">
              использован запасной уровень отбора
            </div>
          )}
        </div>

        <BaseRuleAttributesPanel
          attributeRules={enabledRules}
          ruleDetails={result.base_product.rule_details}
        />
        {!result.base_product.rule_details && (
          <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
            Диагностика атрибутов недоступна — перезапустите backend (<code className="font-mono">go build && restart</code>).
          </p>
        )}

        <SignalContributionSummary
          rows={
            (result.ranked_candidates?.length ?? 0) > 0
              ? result.ranked_candidates
              : result.items
          }
          attributeRules={enabledRules}
        />

        <ScoreTable
          title="В выдаче (после порогов)"
          rows={result.items}
          attributeRules={enabledRules}
          colorFamilies={colorFamilies}
          baseName={baseName}
          titleConfig={titleConfig}
          emptyHint="Ни один кандидат не прошёл strict/parent/branch — см. топ ниже."
        />
        {(result.ranked_candidates?.length ?? 0) > 0 && (
          <ScoreTable
            title="Топ кандидатов по score (до отсечения порогами)"
            rows={result.ranked_candidates}
            attributeRules={enabledRules}
            colorFamilies={colorFamilies}
            baseName={baseName}
            titleConfig={titleConfig}
          />
        )}
      </>
    )}
  </div>
  );
};

const BUILTIN_WEIGHT_COLUMNS: Array<{ key: keyof SimilarScoreBreakdown; label: string; hint: string }> = [
  { key: 'article_prefix', label: 'Артикул', hint: 'weights.article_prefix — префикс group_id' },
  { key: 'price', label: 'Цена', hint: 'weights.price — коридор цены' },
];

function ruleScoreValue(b: SimilarScoreBreakdown, ruleId: string): number {
  if (ruleId === 'color_similarity') {
    return b.rule_scores?.color_similarity ?? b.color ?? 0;
  }
  return b.rule_scores?.[ruleId] ?? 0;
}

function computeBaseSum(b: SimilarScoreBreakdown, attributeRules: AttributeRule[]): number {
  const hasColorRule = attributeRules.some((r) => r.id === 'color_similarity');
  const legacyColor = hasColorRule ? 0 : b.color;
  const rulesSum = attributeRules.reduce((acc, r) => acc + ruleScoreValue(b, r.id), 0);
  return (
    b.category +
    legacyColor +
    b.article_prefix +
    b.price +
    rulesSum
  );
}

function hasAnyNonZero(
  rows: SimilarExplainItem[],
  pick: (b: SimilarScoreBreakdown) => number
): boolean {
  return rows.some((r) => pick(r.breakdown) !== 0);
}

const ScoreTable: React.FC<{
  title: string;
  rows: SimilarExplainItem[];
  attributeRules?: AttributeRule[];
  colorFamilies?: Record<string, string[]>;
  baseName?: string;
  titleConfig?: TitleConfigLike;
  emptyHint?: string;
}> = ({
  title,
  rows,
  attributeRules = [],
  colorFamilies = {},
  baseName = '',
  titleConfig,
  emptyHint,
}) => {
  const visibleRules = attributeRules.filter((r) =>
    hasAnyNonZero(rows, (b) => ruleScoreValue(b, r.id))
  );
  const visibleBuiltin = BUILTIN_WEIGHT_COLUMNS.filter((col) =>
    hasAnyNonZero(rows, (b) => Number(b[col.key]) || 0)
  );
  const showCategory = hasAnyNonZero(rows, (b) => b.category);
  const showTitle = hasAnyNonZero(rows, (b) => b.title);

  return (
  <div>
    <h4 className="text-sm font-semibold text-slate-800 mb-2">{title}</h4>
    <div className="overflow-x-auto border border-slate-200 rounded-lg">
      <table className="min-w-full text-xs">
        <thead className="bg-slate-50 text-slate-600">
          <tr>
            <th className="py-2 px-2 text-left sticky left-0 bg-slate-50">#</th>
            <th className="py-2 px-2 text-left min-w-[280px] sticky left-8 bg-slate-50">Товар</th>
            <th className="py-2 px-2 text-right font-semibold">Итого</th>
            <th className="py-2 px-2 text-right" title="Сумма всех колонок без title — сверка с «База без title»">
              Σ база
            </th>
            <th className="py-2 px-2 text-right">База без title</th>
            {showCategory && (
              <th className="py-2 px-2 text-right" title="weights.category_*">
                Категория
              </th>
            )}
            {visibleRules.map((r) => (
              <th key={r.id} className="py-2 px-2 text-right text-violet-700" title={r.key}>
                {r.label_ru || r.id}
              </th>
            ))}
            {visibleBuiltin.map((col) => (
              <th key={col.key} className="py-2 px-2 text-right text-slate-600" title={col.hint}>
                {col.label}
              </th>
            ))}
            {showTitle && (
              <th className="py-2 px-2 text-right" title="title — совпадение по названию">
                Название
              </th>
            )}
            <th className="py-2 px-2 text-right" title="Σ база + Название — сверка с «Итого»">
              Σ итого
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, idx) => (
            <PreviewRow
              key={`${productColorSizeArticle(row)}-${idx}`}
              row={row}
              index={idx + 1}
              attributeRules={visibleRules}
              allAttributeRules={attributeRules}
              colorFamilies={colorFamilies}
              builtinColumns={visibleBuiltin}
              showCategory={showCategory}
              showTitle={showTitle}
              baseName={baseName}
              titleConfig={titleConfig}
            />
          ))}
        </tbody>
      </table>
      {rows.length === 0 && emptyHint && (
        <p className="text-slate-500 text-sm p-4">{emptyHint}</p>
      )}
    </div>
  </div>
  );
};

const PreviewRow: React.FC<{
  row: SimilarExplainItem;
  index: number;
  attributeRules?: AttributeRule[];
  allAttributeRules?: AttributeRule[];
  colorFamilies?: Record<string, string[]>;
  builtinColumns?: typeof BUILTIN_WEIGHT_COLUMNS;
  showCategory?: boolean;
  showTitle?: boolean;
  baseName?: string;
  titleConfig?: TitleConfigLike;
}> = ({
  row,
  index,
  attributeRules = [],
  allAttributeRules,
  colorFamilies = {},
  builtinColumns = BUILTIN_WEIGHT_COLUMNS,
  showCategory = true,
  showTitle = true,
  baseName = '',
  titleConfig,
}) => {
  const [open, setOpen] = useState(false);
  const [notesOpen, setNotesOpen] = useState(false);
  const b = row.breakdown;
  // Σ база / сверка всегда по полному набору правил документа, не только по видимым колонкам.
  const baseSum = Math.round(computeBaseSum(b, allAttributeRules ?? attributeRules));
  const totalSum = baseSum + b.title;
  const baseOk = baseSum === Math.round(b.base_without_title);
  const totalOk = totalSum === b.total;
  const catLabel =
    b.category > 0
      ? `${CATEGORY_KIND[b.category_kind] || b.category_kind || '?'} ${b.category}${
          b.category_common ? ` · «${b.category_common}»` : ''
        }${b.category_depth ? ` d=${b.category_depth}` : ''}`
      : '—';

  const hasCategoryMatch = Boolean(b.category_kind) || b.category > 0;
  const canExplainTitle = Boolean(showTitle && titleConfig && baseName);
  const titleExplain =
    canExplainTitle && open
      ? explainTitleScore(
          baseName,
          row.name,
          { ok: hasCategoryMatch, kind: b.category_kind || '' },
          titleConfig!
        )
      : null;
  const colSpan =
    5 +
    (showCategory ? 1 : 0) +
    attributeRules.length +
    builtinColumns.length +
    (showTitle ? 1 : 0) +
    1;

  return (
    <>
      <tr className="border-t border-slate-100 hover:bg-violet-50/40">
        <td className="py-2 px-2 text-slate-400 sticky left-0 bg-white">{index}</td>
        <td className="py-2 px-2 sticky left-8 bg-white align-top">
          <div className="flex gap-2 items-start min-w-[280px]">
            {row.image_url && (
              <img
                src={row.image_url}
                alt=""
                className={PREVIEW_PRODUCT_IMAGE_CLASS}
              />
            )}
            <div>
              <a
                href={productShopHref(row)}
                target="_blank"
                rel="noopener noreferrer"
                className="block text-slate-800 hover:text-primary-600 hover:underline break-words"
              >
                {row.name}
              </a>
              <p className="mt-0.5 font-mono text-[11px] leading-tight text-slate-500">
                {productColorSizeArticle(row)}
              </p>
            </div>
          </div>
        </td>
        <ScoreCell value={b.total} strong />
        <SumCheckCell value={baseSum} ok={baseOk} expected={Math.round(b.base_without_title)} />
        <ScoreCell value={Math.round(b.base_without_title)} />
        {showCategory && (
          <td className="py-2 px-2 text-right tabular-nums text-slate-700">{catLabel}</td>
        )}
        {attributeRules.map((r) => (
          <RuleScoreCell
            key={r.id}
            rule={r}
            value={ruleScoreValue(b, r.id)}
            detail={row.rule_details?.[r.id]}
            colorFamilies={colorFamilies}
          />
        ))}
        {builtinColumns.map((col) => (
          <ScoreCell key={col.key} value={Number(b[col.key]) || 0} />
        ))}
        {showTitle && (
          <TitleScoreCell
            value={b.title}
            gate={b.title_gate}
            expandable={canExplainTitle}
            open={open}
            onToggle={() => setOpen((v) => !v)}
          />
        )}
        <SumCheckCell value={totalSum} ok={totalOk} expected={b.total} />
      </tr>
      {b.score_notes && b.score_notes.length > 0 && (
        <tr className="bg-slate-50/80">
          <td className="px-2 py-1" colSpan={colSpan}>
            <button
              type="button"
              onClick={() => setNotesOpen((v) => !v)}
              className="text-[11px] font-medium text-violet-700 hover:underline"
            >
              {notesOpen ? 'Скрыть разбор' : 'Почему такой балл'}
            </button>
            {notesOpen && (
              <ul className="mt-1 list-disc pl-4 text-[11px] text-slate-600">
                {b.score_notes.map((note) => (
                  <li key={note}>{note}</li>
                ))}
              </ul>
            )}
          </td>
        </tr>
      )}
      {titleExplain && (
        <tr className="bg-violet-50/30">
          <td className="px-2 pb-3" colSpan={colSpan}>
            <div className="rounded-lg border border-violet-200 bg-white p-3">
              <p className="mb-2 text-xs font-semibold text-violet-900">
                Разбор названия: база ↔ кандидат
              </p>
              <TitleScorePanel explain={titleExplain} hasCategoryMatch={hasCategoryMatch} />
            </div>
          </td>
        </tr>
      )}
    </>
  );
};

const TitleScoreCell: React.FC<{
  value: number;
  gate?: string;
  expandable: boolean;
  open: boolean;
  onToggle: () => void;
}> = ({ value, gate, expandable, open, onToggle }) => {
  const gateHint = gate ? TITLE_GATE_HINT[gate] || gate : '';
  if (!expandable) {
    return (
      <td className="py-2 px-2 text-right tabular-nums align-top">
        <div className={value > 0 ? 'text-slate-800' : 'text-slate-300'}>{value > 0 ? value : '—'}</div>
        {gateHint ? <div className="text-[10px] text-amber-700">{gateHint}</div> : null}
      </td>
    );
  }
  return (
    <td className="py-2 px-2 text-right tabular-nums align-top">
      <button
        type="button"
        onClick={onToggle}
        title="Показать разбор названия по словам"
        className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-violet-100 ${
          value > 0 ? 'text-slate-800' : 'text-slate-300'
        }`}
      >
        <span>{value > 0 ? value : '—'}</span>
        <span className={`text-[10px] text-violet-500 transition-transform ${open ? 'rotate-90' : ''}`}>
          ▸
        </span>
      </button>
      {gateHint ? <div className="text-[10px] text-amber-700">{gateHint}</div> : null}
    </td>
  );
};

const ScoreCell: React.FC<{ value: number; strong?: boolean }> = ({ value, strong }) => (
  <td
    className={`py-2 px-2 text-right tabular-nums ${
      value > 0 ? (strong ? 'font-semibold text-violet-800' : 'text-slate-800') : 'text-slate-300'
    }`}
  >
    {value > 0 ? value : '—'}
  </td>
);

const SumCheckCell: React.FC<{ value: number; ok: boolean; expected: number }> = ({
  value,
  ok,
  expected,
}) => (
  <td
    title={ok ? 'Сумма сходится' : `Ожидалось ${expected}, посчитано ${value}`}
    className={`py-2 px-2 text-right tabular-nums font-medium ${
      ok ? 'text-emerald-700' : 'text-red-600'
    }`}
  >
    {value}
    {!ok ? ' ⚠' : ''}
  </td>
);

const RULE_STATUS_HINT: Record<string, string> = {
  match: 'совпадение',
  conflict: 'конфликт значений — применён штраф',
  mismatch: 'разные значения',
  missing_base: 'нет у базового',
  missing_cand: 'нет у кандидата',
  missing_both: 'нет в фиде',
  default_value: 'дефолт фида — балл 0',
  scope: 'не для секции',
  ok: 'есть в фиде',
};

const RULE_STATUS_SHORT: Record<string, string> = {
  match: '✓',
  conflict: 'штраф',
  mismatch: '≠',
  missing_base: 'нет базы',
  missing_cand: 'нет канд.',
  missing_both: 'нет в фиде',
  default_value: 'default 0',
  scope: 'секция',
  ok: '',
};

function formatRuleDetailHint(detail?: { base_value?: string; cand_value?: string; status?: string }): string {
  if (!detail?.status) return '';
  if (detail.status === 'mismatch' && detail.cand_value) {
    return `≠ ${detail.cand_value}`;
  }
  return RULE_STATUS_SHORT[detail.status] || RULE_STATUS_HINT[detail.status] || detail.status;
}

const BaseRuleAttributesPanel: React.FC<{
  attributeRules: AttributeRule[];
  ruleDetails?: Record<string, SimilarRuleScoreDetail>;
}> = ({ attributeRules, ruleDetails }) => {
  if (!ruleDetails) return null;

  const present = attributeRules
    .filter((r) => r.enabled)
    .map((rule) => {
      const detail = ruleDetails[rule.id];
      const value = detail?.base_value?.trim() ?? '';
      return { rule, detail, value };
    })
    .filter((item) => item.value);

  if (present.length === 0) return null;

  return (
    <div className="rounded-lg border border-violet-200 bg-violet-50/60 p-3">
      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-violet-900">
        Атрибуты базового товара (из фида)
      </p>
      <div className="flex flex-wrap gap-2">
        {present.map(({ rule, detail, value }) => (
          <span
            key={rule.id}
            title={detail?.status ? RULE_STATUS_HINT[detail.status] || detail.status : undefined}
            className="inline-flex items-center gap-1 rounded-full border border-violet-200 bg-white px-2.5 py-1 text-xs text-violet-900"
          >
            <span className="font-medium">{rule.label_ru || rule.key}:</span>
            <span className="font-mono">{value}</span>
          </span>
        ))}
      </div>
      <p className="mt-2 text-[11px] text-slate-500">
        Показаны только атрибуты, которые есть в фиде у базового товара. В таблице — балл или причина
        расхождения с кандидатом.
      </p>
    </div>
  );
};

const RuleScoreCell: React.FC<{
  rule: AttributeRule;
  value: number;
  detail?: SimilarRuleScoreDetail;
  colorFamilies?: Record<string, string[]>;
}> = ({ rule, value, detail, colorFamilies = {} }) => {
  const colorExplain =
    rule.match_type === 'color_similarity' && detail?.base_value && detail?.cand_value
      ? explainColorSimilarity(
          detail.base_value,
          detail.cand_value,
          colorFamilies,
          rule
        )
      : null;
  const partialColorMatch = isPartialColorMatch(colorExplain);
  const statusHint = partialColorMatch
    ? PARTIAL_COLOR_MATCH_LABEL
    : detail?.status
      ? RULE_STATUS_HINT[detail.status] || detail.status
      : '';
  const hint = value > 0 ? '' : partialColorMatch ? 'частичное совпадение' : formatRuleDetailHint(detail);
  const title = detail
    ? [
        `${rule.label_ru || rule.key}`,
        detail.base_value ? `база: ${detail.base_value}` : 'база: —',
        detail.cand_value ? `канд.: ${detail.cand_value}` : 'канд.: —',
        statusHint,
      ].join(' · ')
    : rule.key;

  return (
    <td
      title={title}
      className={`py-2 px-1.5 text-right align-top ${
        value > 0
          ? 'text-violet-800'
          : value < 0
            ? 'text-red-700'
          : detail?.status === 'default_value'
            ? 'text-amber-700'
            : detail?.status === 'missing_both'
              ? 'text-amber-600'
              : 'text-slate-400'
      }`}
    >
      <div className="tabular-nums font-medium">{value !== 0 ? value : '—'}</div>
      {hint ? (
        <div className="mt-0.5 max-w-[5.5rem] truncate text-[10px] leading-tight text-slate-500" title={title}>
          {hint}
        </div>
      ) : null}
    </td>
  );
};
