import React, { useMemo } from 'react';
import { BeakerIcon } from '@heroicons/react/24/outline';
import { ColorFamilyChips, ColorScorePanel } from '../components/ColorScorePanel';
import { SimilarExplainItem, SimilarPreviewResult, SimilarStrategyDocument } from '../types/strategy';
import { AttributeRule } from '../types/strategyRules';
import { colorFamiliesForString, colorRuleScore, explainColorSimilarity } from '../utils/colorMatch';
import { productColorSizeArticle, productShopHref } from '../utils/productArticle';

const SECTIONS = [
  { id: 'women', label: 'Женщины' },
  { id: 'men', label: 'Мужчины' },
  { id: 'kids', label: 'Дети' },
  { id: 'teens', label: 'Подростки' },
  { id: 'all', label: 'Все' },
];

const PREVIEW_PRODUCT_IMAGE_CLASS =
  'h-24 w-auto max-w-24 object-contain rounded shrink-0';

function findColorRule(rules: AttributeRule[] = []): AttributeRule | null {
  return (
    rules.find((r) => r.enabled && r.id === 'color_similarity') ||
    rules.find((r) => r.enabled && r.match_type === 'color_similarity') ||
    null
  );
}

function colorRows(result: SimilarPreviewResult): SimilarExplainItem[] {
  const rows =
    (result.ranked_candidates?.length ?? 0) > 0 ? result.ranked_candidates! : result.items;
  return [...rows].sort(
    (a, b) =>
      colorRuleScore(b.breakdown) - colorRuleScore(a.breakdown) ||
      b.score - a.score ||
      a.name.localeCompare(b.name, 'ru')
  );
}

type Props = {
  section: string;
  productId: string;
  limit: number;
  loading: boolean;
  result: SimilarPreviewResult | null;
  doc: SimilarStrategyDocument;
  onSectionChange: (v: string) => void;
  onProductIdChange: (v: string) => void;
  onLimitChange: (v: number) => void;
  onRun: () => void;
};

export const SimilarStrategyColorPreviewTab: React.FC<Props> = ({
  section,
  productId,
  limit,
  loading,
  result,
  doc,
  onSectionChange,
  onProductIdChange,
  onLimitChange,
  onRun,
}) => {
  const colorRule = findColorRule(doc.attribute_rules);
  const families = doc.color_family_dictionary || {};
  const palette = doc.color_family_palette || {};
  const baseColor = result?.base_product.rule_details?.color_similarity?.base_value?.trim() ?? '';
  const rows = useMemo(() => (result ? colorRows(result) : []), [result]);

  const stats = useMemo(() => {
    if (!rows.length) return null;
    let fired = 0;
    let sum = 0;
    for (const r of rows) {
      const v = colorRuleScore(r.breakdown);
      if (v > 0) fired += 1;
      sum += v;
    }
    return { fired, total: rows.length, avg: sum / rows.length };
  }, [rows]);

  return (
    <div className="space-y-4">
      {!colorRule && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
          Правило color_similarity не найдено или выключено — добавьте пресет «Гибкий цвет» во вкладке
          «Правила атрибутов».
        </p>
      )}

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
            placeholder="например GRT000823"
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
          <div className="space-y-3 rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm">
            <div className="flex flex-wrap items-start gap-4">
              {result.base_product.image_url && (
                <img
                  src={result.base_product.image_url}
                  alt=""
                  className={PREVIEW_PRODUCT_IMAGE_CLASS}
                />
              )}
              <div className="min-w-0 flex-1">
                <a
                  href={productShopHref(result.base_product)}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-medium text-slate-900 hover:text-primary-600 hover:underline"
                >
                  {result.base_product.name}
                </a>
                <p className="mt-0.5 text-xs text-slate-500 font-mono">
                  {productColorSizeArticle(result.base_product)}
                </p>
              </div>
            </div>

            <div className="w-full rounded-lg border border-violet-200 bg-white p-4">
              <p className="mb-2 text-xs font-semibold text-violet-900">Цвет базового товара</p>
              {baseColor ? (
                <>
                  <p className="mb-2 font-mono text-sm text-slate-800 break-words">{baseColor}</p>
                  <ColorFamilyChips
                    families={Array.from(colorFamiliesForString(baseColor, families))}
                    palette={palette}
                  />
                </>
              ) : (
                <p className="text-xs text-amber-700">Нет color_similarity в rule_details — перезапустите backend.</p>
              )}
            </div>
          </div>

          {stats && (
            <div className="rounded-lg border border-violet-200 bg-violet-50/60 px-3 py-2 text-xs text-violet-950">
              Цвет &gt; 0 у{' '}
              <strong>
                {stats.fired}/{stats.total}
              </strong>{' '}
              кандидатов · средний вклад{' '}
              <strong className="tabular-nums">{stats.avg.toFixed(1)}</strong> баллов
            </div>
          )}

          <div className="space-y-3">
            <h4 className="text-sm font-semibold text-slate-800">
              Кандидаты по баллу цвета (топ до отсечения порогами)
            </h4>
            {rows.length === 0 ? (
              <p className="text-slate-500 text-sm">Кандидаты не найдены — проверьте SKU и секцию.</p>
            ) : (
              rows.map((row, idx) => (
                <ColorCandidateCard
                  key={`${productColorSizeArticle(row)}-${idx}`}
                  row={row}
                  index={idx + 1}
                  baseColor={baseColor}
                  families={families}
                  palette={palette}
                  colorRule={colorRule}
                />
              ))
            )}
          </div>
        </>
      )}
    </div>
  );
};

const ColorCandidateCard: React.FC<{
  row: SimilarExplainItem;
  index: number;
  baseColor: string;
  families: Record<string, string[]>;
  palette: Record<string, string>;
  colorRule: AttributeRule | null;
}> = ({ row, index, baseColor, families, palette, colorRule }) => {
  const candColor = row.rule_details?.color_similarity?.cand_value?.trim() ?? '';
  const serverScore = colorRuleScore(row.breakdown);
  const explain =
    baseColor && candColor
      ? explainColorSimilarity(baseColor, candColor, families, colorRule)
      : null;
  const clientScore = explain?.finalScore ?? 0;
  const mismatch = explain !== null && serverScore !== clientScore;

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 space-y-3">
      <div className="flex flex-wrap items-start gap-3">
        <span className="text-xs text-slate-400 tabular-nums">{index}</span>
        {row.image_url && (
          <img src={row.image_url} alt="" className={PREVIEW_PRODUCT_IMAGE_CLASS} />
        )}
        <div className="min-w-0 flex-1">
          <a
            href={productShopHref(row)}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm font-medium text-slate-900 hover:text-primary-600 hover:underline"
          >
            {row.name}
          </a>
          <p className="mt-0.5 font-mono text-[11px] text-slate-500">
            {productColorSizeArticle(row)}
          </p>
        </div>
        {serverScore !== 0 && (
          <span
            className={`rounded px-2 py-0.5 text-xs tabular-nums font-semibold ${
              serverScore < 0
                ? 'bg-red-50 text-red-800'
                : mismatch
                  ? 'bg-red-50 text-red-800'
                  : 'bg-violet-50 text-violet-900'
            }`}
            title={
              mismatch
                ? `Backend: ${serverScore}, клиент: ${clientScore}`
                : 'Балл цвета с backend'
            }
          >
            {serverScore}
            {mismatch ? ' ⚠' : ''}
          </span>
        )}
      </div>

      {candColor && (
        <p className="font-mono text-xs text-slate-600 break-words">{candColor}</p>
      )}

      {explain && <ColorScorePanel explain={explain} palette={palette} />}

      {mismatch && (
        <p className="text-[11px] text-red-700">
          Backend: {serverScore}, локально: {clientScore}. Проверьте словарь семейств и color-настройки
          правила.
        </p>
      )}
    </div>
  );
};
