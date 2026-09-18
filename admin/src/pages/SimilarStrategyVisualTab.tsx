import React from 'react';
import toast from 'react-hot-toast';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { VisualPreviewRow } from '../services/strategyApi';
import { productColorSizeArticle, productShopHref } from '../utils/productArticle';

const SECTIONS = [
  { id: 'women', label: 'Женщины' },
  { id: 'men', label: 'Мужчины' },
  { id: 'kids', label: 'Дети' },
  { id: 'teens', label: 'Подростки' },
  { id: 'all', label: 'Все' },
];

type Props = {
  section: string;
  loading: boolean;
  rows: VisualPreviewRow[];
  onSectionChange: (v: string) => void;
  onRefresh: () => void;
};

async function copyArticle(article: string) {
  const text = article.trim();
  if (!text) return;
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
    } else {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.setAttribute('readonly', '');
      ta.style.position = 'fixed';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      ta.remove();
    }
    toast.success(`Скопировано: ${text}`);
  } catch {
    toast.error('Не удалось скопировать артикул');
  }
}

const ArticleButton: React.FC<{ article: string; className?: string }> = ({
  article,
  className = '',
}) => {
  if (!article) {
    return <span className={className}>—</span>;
  }
  return (
    <button
      type="button"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        void copyArticle(article);
      }}
      title="Скопировать артикул"
      className={`${className} hover:text-violet-700 hover:underline cursor-pointer`}
    >
      {article}
    </button>
  );
};

export const SimilarStrategyVisualTab: React.FC<Props> = ({
  section,
  loading,
  rows,
  onSectionChange,
  onRefresh,
}) => (
  <div className="space-y-4">
    <div className="flex flex-wrap items-end gap-3">
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
      <button
        type="button"
        onClick={onRefresh}
        disabled={loading}
        className="btn-primary text-sm flex items-center gap-1.5"
      >
        <ArrowPathIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        {loading ? 'Обновляем…' : 'Обновить'}
      </button>
      <p className="text-xs text-slate-500 pb-1">
        10 случайных товаров с фото и по 12 похожих из опубликованного кэша. Клик по
        артикулу — копировать в буфер.
      </p>
      <span className="mb-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700">
        Быстрый просмотр · без пересчёта
      </span>
    </div>

    <p className="text-xs text-slate-400">
      Несохранённые настройки здесь не применяются. После сохранения дождитесь завершения
      прогрева — вкладка автоматически читает новое опубликованное поколение.
    </p>

    {rows.length === 0 && !loading && (
      <p className="text-sm text-slate-400">Нажмите «Обновить», чтобы загрузить выборку.</p>
    )}

    <div className="space-y-5">
      {rows.map((row, idx) => {
        const baseArticle = productColorSizeArticle(row.base);
        return (
          <div
            key={`${row.base.id || row.base.sku}-${idx}`}
            className="rounded-xl border border-slate-200 bg-white p-3 shadow-sm"
          >
            <div className="flex gap-3 items-stretch min-w-0">
              <div className="w-28 shrink-0 flex flex-col items-center gap-1.5">
                <a
                  href={productShopHref(row.base)}
                  target="_blank"
                  rel="noreferrer"
                  className="aspect-[3/4] w-full rounded-lg bg-slate-100 overflow-hidden border border-slate-200 block"
                  title={row.base.name}
                >
                  {row.base.image_url ? (
                    <img
                      src={row.base.image_url}
                      alt={row.base.name || baseArticle}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <div className="h-full w-full flex items-center justify-center text-[10px] text-slate-400">
                      нет фото
                    </div>
                  )}
                </a>
                <ArticleButton
                  article={baseArticle}
                  className="text-[11px] font-mono text-slate-700 text-center break-all leading-tight"
                />
              </div>

              <div className="min-w-0 flex-1">
                <div className="mb-2 flex items-baseline justify-between gap-2">
                  <p className="text-xs text-slate-500 truncate" title={row.base.name}>
                    {row.base.name || 'Без названия'}
                  </p>
                  {row.error && (
                    <span className="shrink-0 text-[11px] text-amber-700">{row.error}</span>
                  )}
                </div>
                <div className="flex gap-2 overflow-x-auto pb-1 snap-x">
                  {(row.items || []).map((item, i) => {
                    const art = productColorSizeArticle(item);
                    return (
                      <div
                        key={`${item.id || item.sku}-${i}`}
                        className="w-20 shrink-0 snap-start flex flex-col gap-1"
                      >
                        <a
                          href={productShopHref(item)}
                          target="_blank"
                          rel="noreferrer"
                          className="aspect-[3/4] w-full rounded-md bg-slate-100 overflow-hidden border border-slate-200 block"
                          title={item.name}
                        >
                          {item.image_url ? (
                            <img
                              src={item.image_url}
                              alt={item.name || art}
                              className="h-full w-full object-cover"
                              loading="lazy"
                            />
                          ) : (
                            <div className="h-full w-full flex items-center justify-center text-[9px] text-slate-400">
                              —
                            </div>
                          )}
                        </a>
                        <ArticleButton
                          article={art}
                          className="text-[10px] font-mono text-slate-600 leading-tight break-all text-left"
                        />
                        {typeof item.score === 'number' && (
                          <span className="text-[10px] text-violet-700 tabular-nums">
                            {Math.round(item.score)}
                          </span>
                        )}
                      </div>
                    );
                  })}
                  {!loading && (row.items || []).length === 0 && !row.error && (
                    <p className="text-xs text-slate-400 self-center">Похожих нет</p>
                  )}
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  </div>
);
