import React, { useMemo } from 'react';
import { BeakerIcon } from '@heroicons/react/24/outline';
import { SimilarExplainItem, SimilarPreviewResult } from '../types/strategy';
import { TitleConfigLike, analyzeTitle, explainTitleScore } from '../utils/titleMatch';
import {
  TitleScorePanel,
  TitleTokensBlock,
  TitleTokensLegend,
} from '../components/TitleTokens';
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

function categoryContext(row: SimilarExplainItem): { ok: boolean; kind: string } {
  const b = row.breakdown;
  return {
    ok: Boolean(b.category_kind) || b.category > 0,
    kind: b.category_kind || '',
  };
}

function titleRows(result: SimilarPreviewResult): SimilarExplainItem[] {
  const rows =
    (result.ranked_candidates?.length ?? 0) > 0 ? result.ranked_candidates! : result.items;
  return [...rows].sort(
    (a, b) =>
      b.breakdown.title - a.breakdown.title ||
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
  titleConfig: TitleConfigLike;
  onSectionChange: (v: string) => void;
  onProductIdChange: (v: string) => void;
  onLimitChange: (v: number) => void;
  onRun: () => void;
};

export const SimilarStrategyTitlePreviewTab: React.FC<Props> = ({
  section,
  productId,
  limit,
  loading,
  result,
  titleConfig,
  onSectionChange,
  onProductIdChange,
  onLimitChange,
  onRun,
}) => {
  const baseName = result?.base_product.name ?? '';
  const baseAnalysis = useMemo(
    () => (baseName ? analyzeTitle(baseName, titleConfig) : null),
    [baseName, titleConfig]
  );
  const rows = useMemo(() => (result ? titleRows(result) : []), [result]);
  const stats = useMemo(() => {
    if (!rows.length) return null;
    let fired = 0;
    let sum = 0;
    for (const r of rows) {
      const t = r.breakdown.title;
      if (t > 0) fired += 1;
      sum += t;
    }
    return { fired, total: rows.length, avg: sum / rows.length };
  }, [rows]);

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600">
        Выберите базовый SKU — ниже разбор названия по токенам для базы и каждого кандидата (как во
        вкладке «Название»). Зелёная обводка — совпавшие токены. Учитываются несохранённые
        stopwords / model_words.
      </p>

      {!titleConfig.enabled && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
          Title matching выключен в настройках — балл названия будет 0 у всех кандидатов.
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
                {result.base_product.leaf_category && (
                  <p className="mt-0.5 text-xs text-slate-500">
                    Категория: {result.base_product.leaf_category}
                  </p>
                )}
              </div>
            </div>

            {baseAnalysis && (
              <div className="w-full rounded-lg border border-slate-200 bg-white p-4">
                <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
                  <p className="text-xs font-semibold text-slate-700">Токены базового названия</p>
                  <TitleTokensLegend />
                </div>
                <TitleTokensBlock label="База" analysis={baseAnalysis} variant="stacked" />
              </div>
            )}
          </div>

          {stats && (
            <div className="rounded-lg border border-amber-200 bg-amber-50/60 px-3 py-2 text-xs text-amber-950">
              Название &gt; 0 у{' '}
              <strong>
                {stats.fired}/{stats.total}
              </strong>{' '}
              кандидатов · средний вклад{' '}
              <strong className="tabular-nums">{stats.avg.toFixed(1)}</strong> баллов
            </div>
          )}

          <div className="space-y-3">
            <h4 className="text-sm font-semibold text-slate-800">
              Кандидаты по баллу названия (топ до отсечения порогами)
            </h4>
            {rows.length === 0 ? (
              <p className="text-slate-500 text-sm">Кандидаты не найдены — проверьте SKU и секцию.</p>
            ) : (
              rows.map((row, idx) => (
                <TitleCandidateCard
                  key={`${productColorSizeArticle(row)}-${idx}`}
                  row={row}
                  index={idx + 1}
                  baseName={baseName}
                  titleConfig={titleConfig}
                />
              ))
            )}
          </div>
        </>
      )}
    </div>
  );
};

const TitleCandidateCard: React.FC<{
  row: SimilarExplainItem;
  index: number;
  baseName: string;
  titleConfig: TitleConfigLike;
}> = ({ row, index, baseName, titleConfig }) => {
  const category = categoryContext(row);
  const explain = explainTitleScore(baseName, row.name, category, titleConfig);
  const serverTitle = row.breakdown.title;
  const clientTitle = explain.finalScore;
  const mismatch = serverTitle !== clientTitle;

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
        {serverTitle > 0 && (
          <span
            className={`rounded px-2 py-0.5 text-xs tabular-nums font-semibold ${
              mismatch ? 'bg-red-50 text-red-800' : 'bg-amber-50 text-amber-900'
            }`}
            title={
              mismatch
                ? `Backend: ${serverTitle}, клиент: ${clientTitle}`
                : 'Балл названия с backend'
            }
          >
            {serverTitle}
            {mismatch ? ' ⚠' : ''}
          </span>
        )}
      </div>

      <TitleTokensBlock label="Кандидат" analysis={explain.cand} variant="stacked" />
      <TitleScorePanel
        explain={explain}
        hasCategoryMatch={category.ok}
        hideBaseTokens
        showCandidate={false}
      />
      {mismatch && (
        <p className="text-[11px] text-red-700">
          Backend начислил {serverTitle} баллов, локальный пересчёт — {clientTitle}. Если расхождение
          остаётся после сохранения пресета — перезапустите backend{' '}
          <span className="font-mono">(go build ./cmd/similar && PORT=8081 …)</span>.
        </p>
      )}
    </div>
  );
};
