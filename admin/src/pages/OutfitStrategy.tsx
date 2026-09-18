import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import toast from 'react-hot-toast';
import {
  ArrowPathIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  LinkIcon,
  SquaresPlusIcon,
} from '@heroicons/react/24/outline';
import { outfitApi } from '../services/outfitApi';
import {
  OutfitPreviewResult,
  OutfitStrategyDocument,
  OutfitWarmStatus,
} from '../types/outfit';
import {
  DEFAULT_OUTFIT_STRATEGY,
  normalizeOutfitStrategy,
} from '../utils/normalizeOutfitStrategy';

function errorMessage(error: any): string {
  return String(
    error?.response?.data?.error?.message || error?.message || 'Не удалось выполнить запрос'
  );
}

function formatWarmTime(unix?: number): string {
  if (!unix) return 'ещё не было';
  return new Date(unix * 1000).toLocaleString('ru-RU');
}

function ProductCard({ product }: { product: OutfitPreviewResult['base_product'] }) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <div className="aspect-square bg-slate-100">
        {product.image_url ? (
          <img src={product.image_url} alt={product.name} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full items-center justify-center text-xs text-slate-400">Нет фото</div>
        )}
      </div>
      <div className="p-3">
        <div className="text-xs font-semibold text-violet-700">{product.id}</div>
        <div className="mt-1 line-clamp-2 text-sm text-slate-800">{product.name || 'Без названия'}</div>
      </div>
    </div>
  );
}

export const OutfitStrategy: React.FC = () => {
  const [strategy, setStrategy] = useState<OutfitStrategyDocument>(DEFAULT_OUTFIT_STRATEGY);
  const [builtinDefault, setBuiltinDefault] = useState<OutfitStrategyDocument>(DEFAULT_OUTFIT_STRATEGY);
  const [warm, setWarm] = useState<OutfitWarmStatus | null>(null);
  const [article, setArticle] = useState('');
  const [preview, setPreview] = useState<OutfitPreviewResult | null>(null);
  const [reason, setReason] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [previewing, setPreviewing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [configResponse, warmResponse] = await Promise.all([
        outfitApi.getStrategy(),
        outfitApi.getWarmStatus(),
      ]);
      setStrategy(configResponse.strategy);
      setBuiltinDefault(configResponse.builtin_default);
      setWarm(warmResponse);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    const timer = window.setInterval(async () => {
      try {
        setWarm(await outfitApi.getWarmStatus());
      } catch {
        // Статус вторичен; ошибки основного действия показываются отдельно.
      }
    }, 3000);
    return () => window.clearInterval(timer);
  }, []);

  const progress = useMemo(() => {
    if (!warm?.total) return 0;
    return Math.min(100, Math.round((warm.done / warm.total) * 100));
  }, [warm]);

  const save = async () => {
    const normalized = normalizeOutfitStrategy(strategy);
    if (!normalized.attribute_keys.length) {
      toast.error('Добавьте хотя бы один ключ атрибута');
      return;
    }
    setSaving(true);
    try {
      setStrategy(await outfitApi.saveStrategy(normalized, reason));
      setReason('');
      setPreview(null);
      toast.success('Настройки сохранены, новое поколение прогревается');
      setWarm(await outfitApi.getWarmStatus());
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const reset = async () => {
    setSaving(true);
    try {
      setStrategy(await outfitApi.resetStrategy());
      setPreview(null);
      toast.success('Возвращены настройки MVP, кэш прогревается');
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const runPreview = async () => {
    const productId = article.trim();
    if (!productId) {
      toast.error('Укажите артикул');
      return;
    }
    setPreviewing(true);
    try {
      setPreview(await outfitApi.preview(productId, normalizeOutfitStrategy(strategy)));
    } catch (error) {
      setPreview(null);
      toast.error(errorMessage(error));
    } finally {
      setPreviewing(false);
    }
  };

  if (loading) {
    return <div className="p-8 text-sm text-slate-500">Загрузка настроек образов…</div>;
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-4 py-5">
          <div>
            <div className="flex items-center gap-2">
              <SquaresPlusIcon className="h-7 w-7 text-violet-600" />
              <h1 className="text-2xl font-bold text-slate-900">Комплекты</h1>
              <span className="rounded-full bg-violet-100 px-2 py-1 text-[11px] font-semibold text-violet-700">MVP</span>
            </div>
            <p className="mt-1 text-sm text-slate-500">Комплектность вещь + вещь из атрибута фида</p>
          </div>
          <Link to="/similar-strategy" className="btn-outline gap-2">
            Похожие товары
            <ArrowRightIcon className="h-4 w-4" />
          </Link>
        </div>
      </header>

      <main className="mx-auto max-w-6xl space-y-6 px-4 py-6">
        <section className="rounded-xl border border-violet-200 bg-violet-50 p-4 text-sm text-violet-950">
          <div className="font-semibold">Что делает MVP</div>
          <p className="mt-1 text-violet-800">
            Берёт только явные артикулы из настроенных атрибутов, проверяет наличие опубликованной карточки и
            отдаёт готовый список из прогретого поколения. Расширение похожими товарами комплекта будет следующим этапом.
          </p>
        </section>

        <div className="grid gap-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(300px,0.65fr)]">
          <section className="space-y-5 rounded-xl border border-slate-200 bg-white p-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <h2 className="font-semibold text-slate-900">Настройки алгоритма</h2>
                <p className="mt-1 text-xs text-slate-500">Независимы от настроек «Похожих товаров».</p>
              </div>
              <label className="flex items-center gap-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  checked={strategy.enabled}
                  onChange={(event) => setStrategy({ ...strategy, enabled: event.target.checked })}
                  className="rounded border-slate-300 text-violet-600"
                />
                Включён
              </label>
            </div>

            <div>
              <label className="mb-1 block text-xs font-medium text-slate-600">Ключи атрибутов, по одному в строке</label>
              <textarea
                value={strategy.attribute_keys.join('\n')}
                onChange={(event) =>
                  setStrategy({
                    ...strategy,
                    attribute_keys: event.target.value.split('\n'),
                  })
                }
                rows={4}
                className="w-full rounded-lg border-slate-300 text-sm focus:border-violet-500 focus:ring-violet-500"
              />
              <p className="mt-1 text-xs text-slate-400">Регистр, пробелы и дефисы в названии ключа не влияют на сравнение.</p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-slate-600">Максимум товаров в образе</label>
                <input
                  type="number"
                  min={1}
                  max={40}
                  value={strategy.max_items}
                  onChange={(event) => setStrategy({ ...strategy, max_items: Number(event.target.value) })}
                  className="w-full rounded-lg border-slate-300 text-sm focus:border-violet-500 focus:ring-violet-500"
                />
              </div>
              <label className="flex items-center gap-3 rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  checked={strategy.include_reverse_links}
                  onChange={(event) =>
                    setStrategy({ ...strategy, include_reverse_links: event.target.checked })
                  }
                  className="rounded border-slate-300 text-violet-600"
                />
                Добавлять обратные связи
              </label>
            </div>

            <div>
              <label className="mb-1 block text-xs font-medium text-slate-600">Комментарий к изменению</label>
              <input
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Например: включили обратные связи"
                className="w-full rounded-lg border-slate-300 text-sm focus:border-violet-500 focus:ring-violet-500"
              />
            </div>

            <div className="flex flex-wrap gap-2">
              <button onClick={save} disabled={saving} className="btn-primary gap-2 disabled:opacity-50">
                <ArrowPathIcon className={`h-4 w-4 ${saving ? 'animate-spin' : ''}`} />
                Сохранить и прогреть
              </button>
              <button
                onClick={() => setStrategy(builtinDefault)}
                disabled={saving}
                className="btn-outline"
              >
                Подставить дефолт
              </button>
              <button onClick={reset} disabled={saving} className="btn-outline text-slate-500">
                Сбросить опубликованные
              </button>
            </div>
          </section>

          <section className="rounded-xl border border-slate-200 bg-white p-5">
            <h2 className="font-semibold text-slate-900">Поколение кэша</h2>
            {warm ? (
              <div className="mt-4 space-y-4">
                <div className="flex items-center gap-2">
                  {warm.ready ? (
                    <CheckCircleIcon className="h-5 w-5 text-emerald-600" />
                  ) : (
                    <ExclamationTriangleIcon className="h-5 w-5 text-amber-500" />
                  )}
                  <div className="text-sm font-medium text-slate-800">
                    {warm.running ? `Прогрев: ${warm.phase}` : warm.ready ? 'Готово' : 'Нет активного поколения'}
                  </div>
                </div>
                {warm.running && (
                  <div>
                    <div className="h-2 overflow-hidden rounded-full bg-slate-100">
                      <div className="h-full bg-violet-600 transition-all" style={{ width: `${progress}%` }} />
                    </div>
                    <div className="mt-1 text-xs text-slate-500">{warm.done} / {warm.total}</div>
                  </div>
                )}
                <dl className="grid grid-cols-2 gap-3 text-sm">
                  <div><dt className="text-xs text-slate-400">Serving generation</dt><dd className="font-semibold">{warm.serving_generation || '—'}</dd></div>
                  <div><dt className="text-xs text-slate-400">Карточек со связями</dt><dd className="font-semibold">{warm.linked_cards}</dd></div>
                  <div><dt className="text-xs text-slate-400">Разрешено ссылок</dt><dd className="font-semibold text-emerald-700">{warm.resolved_links}</dd></div>
                  <div><dt className="text-xs text-slate-400">Не найдено</dt><dd className="font-semibold text-amber-700">{warm.unresolved_links}</dd></div>
                </dl>
                <div className="text-xs text-slate-400">Последний прогрев: {formatWarmTime(warm.last_warmed_at_unix)}</div>
                {warm.last_error && <div className="rounded-lg bg-red-50 p-2 text-xs text-red-700">{warm.last_error}</div>}
              </div>
            ) : (
              <div className="mt-3 text-sm text-slate-400">Статус недоступен</div>
            )}
          </section>
        </div>

        <section className="rounded-xl border border-slate-200 bg-white p-5">
          <div className="flex flex-wrap items-end gap-3">
            <div className="min-w-[260px] flex-1">
              <label className="mb-1 block text-xs font-medium text-slate-600">Проверка комплектности по артикулу</label>
              <input
                value={article}
                onChange={(event) => setArticle(event.target.value)}
                onKeyDown={(event) => event.key === 'Enter' && runPreview()}
                placeholder="Например, GSH013345-1"
                className="w-full rounded-lg border-slate-300 text-sm focus:border-violet-500 focus:ring-violet-500"
              />
            </div>
            <button onClick={runPreview} disabled={previewing} className="btn-primary gap-2 disabled:opacity-50">
              <LinkIcon className="h-4 w-4" />
              {previewing ? 'Проверяем…' : 'Проверить образ'}
            </button>
          </div>

          {preview && (
            <div className="mt-6 space-y-6">
              <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Базовая вещь</div>
                <div className="w-44"><ProductCard product={preview.base_product} /></div>
              </div>
              <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Вещи комплекта</div>
                {preview.items.length ? (
                  <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-5">
                    {preview.items.map((item) => <ProductCard key={item.id} product={item} />)}
                  </div>
                ) : (
                  <div className="rounded-lg border border-dashed border-slate-300 p-5 text-sm text-slate-500">
                    Разрешённых товаров комплекта нет. Диагностика ссылок — ниже.
                  </div>
                )}
              </div>
              <div>
                <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">Проверка ссылок</div>
                {preview.references.length ? (
                  <div className="divide-y divide-slate-100 rounded-lg border border-slate-200">
                    {preview.references.map((ref, index) => (
                      <div key={`${ref.direction}-${ref.reference}-${index}`} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm">
                        <div><span className="font-mono text-xs text-slate-700">{ref.reference}</span><span className="ml-2 text-xs text-slate-400">{ref.direction === 'direct' ? 'прямая' : 'обратная'}</span></div>
                        <span className={`rounded-full px-2 py-1 text-xs font-medium ${ref.status === 'resolved' ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'}`}>
                          {ref.status === 'resolved' ? `Найден: ${ref.resolved_id}` : ref.status === 'self_reference' ? 'Ссылка на себя' : 'Товар не найден'}
                        </span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-sm text-slate-500">У товара нет настроенных атрибутов комплектности.</div>
                )}
              </div>
            </div>
          )}
        </section>
      </main>
    </div>
  );
};
