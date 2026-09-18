import React, { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { ArrowPathIcon, LinkIcon } from '@heroicons/react/24/outline';
import { recommendationCompositionApi } from '../services/recommendationCompositionApi';
import {
  RecommendationCompositionConfig,
  RecommendationCompositionPreview,
} from '../types/recommendationComposition';
import { DEFAULT_RECOMMENDATION_COMPOSITION } from '../utils/normalizeRecommendationComposition';

export const RecommendationCompositionPanel: React.FC = () => {
  const [config, setConfig] = useState<RecommendationCompositionConfig>(
    DEFAULT_RECOMMENDATION_COMPOSITION
  );
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [reason, setReason] = useState('');
  const [productId, setProductId] = useState('');
  const [section, setSection] = useState('all');
  const [limit, setLimit] = useState(12);
  const [preview, setPreview] = useState<RecommendationCompositionPreview | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [loadError, setLoadError] = useState('');

  const load = async () => {
    setLoading(true);
    setLoadError('');
    try {
      const response = await recommendationCompositionApi.getStrategy();
      setConfig(response.strategy);
    } catch {
      setLoadError('Не удалось загрузить настройку витрины');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const save = async () => {
    setSaving(true);
    try {
      const saved = await recommendationCompositionApi.saveStrategy(config, reason.trim());
      setConfig(saved);
      setReason('');
      toast.success('Настройка витрины сохранена — прогрев не требуется');
    } catch {
      toast.error('Не удалось сохранить настройку витрины');
    } finally {
      setSaving(false);
    }
  };

  const reset = async () => {
    setSaving(true);
    try {
      setConfig(await recommendationCompositionApi.resetStrategy());
      setPreview(null);
      toast.success('Композиция витрины сброшена');
    } catch {
      toast.error('Не удалось сбросить композицию витрины');
    } finally {
      setSaving(false);
    }
  };

  const runPreview = async () => {
    if (!productId.trim()) {
      toast.error('Укажите артикул');
      return;
    }
    setPreviewing(true);
    try {
      setPreview(
        await recommendationCompositionApi.preview(productId.trim(), section, limit, config)
      );
    } catch {
      setPreview(null);
      toast.error('Не удалось собрать витрину для этого артикула');
    } finally {
      setPreviewing(false);
    }
  };

  if (loading) {
    return <div className="rounded-xl border border-slate-200 bg-white p-4 text-sm text-slate-500">Загрузка композиции витрины…</div>;
  }

  if (loadError) {
    return (
      <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800">
        {loadError}{' '}
        <button type="button" onClick={load} className="underline">Повторить</button>
      </div>
    );
  }

  return (
    <section className="rounded-xl border border-violet-200 bg-white shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-4 border-b border-violet-100 bg-violet-50/70 px-5 py-4">
        <div>
          <h2 className="flex items-center gap-2 text-base font-semibold text-slate-900">
            <LinkIcon className="h-5 w-5 text-violet-600" />
            Состав витрины рекомендаций
          </h2>
          <p className="mt-1 max-w-2xl text-sm text-slate-600">
            Отдельный слой объединяет готовые «Комплекты» и «Похожие». Скоринг и кэши алгоритмов не меняются.
          </p>
        </div>
        <span className="rounded-full bg-white px-3 py-1 text-xs font-medium text-violet-700 ring-1 ring-violet-200">
          Без прогрева
        </span>
      </div>

      <div className="space-y-5 p-5">
        <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-slate-200 p-4">
          <input
            type="checkbox"
            checked={config.show_outfit_first}
            onChange={(event) =>
              setConfig({ ...config, show_outfit_first: event.target.checked })
            }
            className="mt-0.5 rounded border-slate-300 text-violet-600"
          />
          <span>
            <span className="block text-sm font-semibold text-slate-900">
              Показывать комплектные товары первыми
            </span>
            <span className="mt-1 block text-xs text-slate-500">
              Все товары из атрибута «Сочетается с», затем похожие товары в текущем порядке. Общий limit сохраняется.
            </span>
          </span>
        </label>

        <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-amber-200 bg-amber-50/60 p-4">
          <input
            type="checkbox"
            checked={config.serve_composed_via_similar}
            onChange={(event) =>
              setConfig({ ...config, serve_composed_via_similar: event.target.checked })
            }
            className="mt-0.5 rounded border-amber-300 text-amber-600"
          />
          <span>
            <span className="block text-sm font-semibold text-slate-900">
              Отдавать итоговую выдачу через <code>/similar</code>
            </span>
            <span className="mt-1 block text-xs text-slate-600">
              Включает комплекты для текущего фронта без его доработки. Выключение сразу возвращает прежний ответ <code>/similar</code>.
            </span>
            {!config.show_outfit_first && config.serve_composed_via_similar && (
              <span className="mt-2 block text-xs font-medium text-amber-800">
                Комплектные товары не появятся, пока выключена настройка выше.
              </span>
            )}
          </span>
        </label>

        <div className="flex flex-wrap items-center gap-2">
          <input
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder="Комментарий к изменению"
            className="min-w-[240px] flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm"
          />
          <button type="button" onClick={save} disabled={saving} className="btn-primary text-sm">
            {saving ? 'Сохраняем…' : 'Сохранить настройку'}
          </button>
          <button type="button" onClick={reset} disabled={saving} className="btn-outline text-sm">
            <ArrowPathIcon className="mr-1 inline h-4 w-4" />
            Выключить и сбросить
          </button>
        </div>

        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <h3 className="text-sm font-semibold text-slate-900">Проверка итогового порядка</h3>
          <div className="mt-3 flex flex-wrap gap-2">
            <input
              value={productId}
              onChange={(event) => setProductId(event.target.value)}
              placeholder="Артикул, например GPT011728-1"
              className="min-w-[250px] flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm"
            />
            <select
              value={section}
              onChange={(event) => setSection(event.target.value)}
              className="rounded-lg border border-slate-300 px-3 py-2 text-sm"
            >
              <option value="all">all</option>
              <option value="women">women</option>
              <option value="men">men</option>
              <option value="kids">kids</option>
              <option value="teens">teens</option>
            </select>
            <input
              type="number"
              min={1}
              max={40}
              value={limit}
              onChange={(event) => setLimit(Math.max(1, Math.min(40, Number(event.target.value) || 1)))}
              className="w-20 rounded-lg border border-slate-300 px-3 py-2 text-sm"
              aria-label="Лимит итоговой витрины"
            />
            <button type="button" onClick={runPreview} disabled={previewing} className="btn-outline text-sm">
              {previewing ? 'Собираем…' : 'Проверить витрину'}
            </button>
          </div>

          {preview && (
            <div className="mt-4 space-y-2">
              {preview.warning && (
                <p className="text-xs text-amber-700">Комплекты временно недоступны — показаны только похожие.</p>
              )}
              {preview.items.length === 0 ? (
                <p className="text-sm text-slate-500">Рекомендаций нет.</p>
              ) : (
                preview.items.map((item, index) => (
                  <div key={`${item.id}-${item.source}`} className="flex items-center gap-3 rounded-lg bg-white px-3 py-2 text-sm ring-1 ring-slate-200">
                    <span className="w-6 text-right font-mono text-xs text-slate-400">{index + 1}</span>
                    <span className="font-mono font-medium text-slate-900">{item.id}</span>
                    <span className={`ml-auto rounded-full px-2 py-0.5 text-xs font-medium ${item.source === 'outfit' ? 'bg-amber-100 text-amber-800' : 'bg-slate-100 text-slate-600'}`}>
                      {item.source === 'outfit' ? 'Комплект' : 'Похожий'}
                    </span>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </section>
  );
};
