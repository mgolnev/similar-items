import React, { useState } from 'react';
import axios from 'axios';
import toast from 'react-hot-toast';
import { API_BASE_URL } from '../services/api';

const SECTIONS = ['all', 'women', 'men', 'kids', 'teens'];

const SHOWCASE_ENDPOINTS = {
  recommendations: {
    label: 'Итоговая выдача',
    path: 'recommendations',
    button: 'Показать итоговую выдачу',
    empty: 'Рекомендации не найдены',
  },
  outfits: {
    label: 'Комплекты',
    path: 'outfits',
    button: 'Показать комплекты',
    empty: 'Комплекты не найдены',
  },
  similar: {
    label: 'Похожие товары',
    path: 'similar',
    button: 'Показать похожие',
    empty: 'Похожие не найдены',
  },
} as const;

type ShowcaseEndpoint = keyof typeof SHOWCASE_ENDPOINTS;

type ApiCallMeta = {
  method: string;
  url: string;
  status: number;
  durationMs: number;
};

export const SimilarShowcase: React.FC = () => {
  const [article, setArticle] = useState('');
  const [section, setSection] = useState('all');
  const [limit, setLimit] = useState(12);
  const [endpoint, setEndpoint] = useState<ShowcaseEndpoint>('recommendations');
  const [loading, setLoading] = useState(false);
  const [responseBody, setResponseBody] = useState<unknown>(null);
  const [meta, setMeta] = useState<ApiCallMeta | null>(null);
  const [queried, setQueried] = useState(false);

  const fetchProducts = async () => {
    const id = article.trim();
    if (!id) {
      toast.error('Укажите артикул товара');
      return;
    }
    setLoading(true);
    const endpointConfig = SHOWCASE_ENDPOINTS[endpoint];
    const path = `/api/v1/client/products/${encodeURIComponent(id)}/${endpointConfig.path}`;
    const url = `${API_BASE_URL}${path}?section=${encodeURIComponent(section)}&limit=${limit}`;
    const started = performance.now();
    try {
      const res = await axios.get(url);
      setResponseBody(res.data);
      setMeta({
        method: 'GET',
        url,
        status: res.status,
        durationMs: Math.round(performance.now() - started),
      });
      setQueried(true);
      const items = (res.data as { items?: unknown })?.items;
      if (Array.isArray(items) && items.length === 0) {
        toast(endpointConfig.empty, { icon: 'ℹ️' });
      }
    } catch (e: any) {
      const status = e?.response?.status ?? 0;
      const body = e?.response?.data ?? { error: e?.message ?? 'Ошибка запроса' };
      setResponseBody(body);
      setMeta({
        method: 'GET',
        url,
        status: status || 0,
        durationMs: Math.round(performance.now() - started),
      });
      setQueried(true);
      const msg =
        e?.response?.data?.error?.message ?? e?.response?.data?.error ?? e?.message ?? 'Ошибка запроса';
      toast.error(`Запрос завершился с ошибкой: ${msg}`);
    } finally {
      setLoading(false);
    }
  };

  const items = Array.isArray((responseBody as { items?: unknown })?.items)
    ? ((responseBody as { items: unknown[] }).items as unknown[])
    : null;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <h2 className="text-xl font-semibold text-gray-900 mb-4">Витрина рекомендаций</h2>
      <p className="text-sm text-gray-500 mb-6">
        Сравните итоговую выдачу, комплекты и похожие товары. Публичные API отдают только массив id
        (SKU группы), без карточек — обогащение на BFF. Ниже — сырой ответ бэкенда.
      </p>

      <div className="flex flex-wrap items-end gap-3 mb-8">
        <div className="flex flex-col">
          <label className="text-xs text-gray-500 mb-1">Артикул (SKU)</label>
          <input
            value={article}
            onChange={(e) => setArticle(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && fetchProducts()}
            placeholder="например, BOW002608-1"
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm w-64 focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <div className="flex flex-col">
          <label className="text-xs text-gray-500 mb-1">Ручка</label>
          <select
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value as ShowcaseEndpoint)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            {Object.entries(SHOWCASE_ENDPOINTS).map(([key, config]) => (
              <option key={key} value={key}>
                {config.label}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col">
          <label className="text-xs text-gray-500 mb-1">Секция</label>
          <select
            value={section}
            onChange={(e) => setSection(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            {SECTIONS.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col">
          <label className="text-xs text-gray-500 mb-1">Лимит</label>
          <input
            type="number"
            min={1}
            max={60}
            value={limit}
            onChange={(e) => setLimit(Math.max(1, Number(e.target.value) || 12))}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm w-24 focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <button
          onClick={fetchProducts}
          disabled={loading}
          className="bg-primary-600 hover:bg-primary-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-5 py-2 transition-colors"
        >
          {loading ? 'Загрузка…' : SHOWCASE_ENDPOINTS[endpoint].button}
        </button>
      </div>

      {queried && meta && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-700 font-mono break-all">
            <div>
              <span className="font-semibold text-gray-900">{meta.method}</span> {meta.url}
            </div>
            <div className="mt-1 text-xs text-gray-500">
              HTTP {meta.status || '—'} · {meta.durationMs} ms
            </div>
          </div>

          {items && items.length > 0 && (
            <div className="rounded-lg border border-gray-200 bg-white px-4 py-3">
              <div className="text-xs font-medium text-gray-500 mb-2">
                items ({items.length}) — порядок выдачи
              </div>
              <ol className="list-decimal list-inside text-sm font-mono text-gray-900 space-y-0.5">
                {items.map((item, idx) => (
                  <li key={`${String(item)}-${idx}`}>{String(item)}</li>
                ))}
              </ol>
            </div>
          )}

          {items && items.length === 0 && !loading && (
            <div className="text-sm text-gray-500">
              items: [] — {SHOWCASE_ENDPOINTS[endpoint].empty.toLocaleLowerCase()}.
            </div>
          )}

          <div>
            <div className="text-xs font-medium text-gray-500 mb-2">Response JSON</div>
            <pre className="rounded-lg border border-gray-200 bg-gray-900 text-gray-100 text-xs p-4 overflow-x-auto whitespace-pre-wrap break-all">
              {JSON.stringify(responseBody, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
};
