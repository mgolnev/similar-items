import React, { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import {
  ArrowPathIcon,
  CalendarDaysIcon,
  CheckCircleIcon,
  CloudArrowUpIcon,
  CubeIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { feedApi, FeedDataStatus } from '../services/api';

const DEFAULT_FEED_URL = '';

function normalizeFeedUrl(url?: string | null): string {
  const trimmed = (url ?? '').trim();
  if (!trimmed) {
    return DEFAULT_FEED_URL;
  }
  return trimmed;
}

const INTERVAL_OPTIONS = [
  { value: 0, label: 'Выключено' },
  { value: 30, label: 'Каждые 30 минут' },
  { value: 60, label: 'Каждый час' },
  { value: 360, label: 'Каждые 6 часов' },
  { value: 1440, label: 'Раз в сутки' },
];

function formatDateTime(value?: string | null) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('ru-RU');
}

function intervalLabel(minutes?: number) {
  if (!minutes) return 'выключено';
  const opt = INTERVAL_OPTIONS.find((o) => o.value === minutes);
  if (opt) return opt.label.toLowerCase();
  if (minutes % 1440 === 0) return `каждые ${minutes / 1440} сут.`;
  if (minutes % 60 === 0) return `каждые ${minutes / 60} ч.`;
  return `каждые ${minutes} мин.`;
}

function StatusCard({
  title,
  ok,
  value,
  meta,
}: {
  title: string;
  ok: boolean;
  value: React.ReactNode;
  meta?: React.ReactNode;
}) {
  return (
    <div
      className={`rounded-xl border p-4 ${
        ok ? 'bg-green-50 border-green-200' : 'bg-gray-50 border-gray-200'
      }`}
    >
      <div className="flex items-start justify-between gap-2 mb-1">
        <div className={`text-sm font-medium ${ok ? 'text-green-800' : 'text-gray-600'}`}>{title}</div>
        {ok ? (
          <CheckCircleIcon className="w-5 h-5 text-green-600 shrink-0" />
        ) : (
          <XCircleIcon className="w-5 h-5 text-gray-400 shrink-0" />
        )}
      </div>
      <div className={`text-lg font-bold ${ok ? 'text-green-900' : 'text-gray-900'}`}>{value}</div>
      {meta && <div className={`mt-2 text-xs ${ok ? 'text-green-800' : 'text-gray-600'}`}>{meta}</div>}
    </div>
  );
}

export const Feed: React.FC = () => {
  const [busy, setBusy] = useState(false);
  const [savingSchedule, setSavingSchedule] = useState(false);
  const [status, setStatus] = useState<FeedDataStatus | null>(null);
  const [feedUrl, setFeedUrl] = useState('');
  const [syncIntervalMinutes, setSyncIntervalMinutes] = useState(60);

  const loadAll = useCallback(async () => {
    try {
      const [s, settings] = await Promise.all([feedApi.getDataStatus(), feedApi.getFeedSettings()]);
      setStatus(s);
      setFeedUrl(normalizeFeedUrl(settings.feed_url || s.default_feed_url));
      setSyncIntervalMinutes(settings.sync_interval_minutes ?? s.feed_sync_interval_minutes ?? 60);
    } catch {
      /* не критично */
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  const refreshFeed = async () => {
    const url = feedUrl.trim();
    if (!url) {
      toast.error('Укажите URL фида');
      return;
    }
    setBusy(true);
    try {
      await feedApi.processFeed(url);
      toast.success('Фид загружен');
      await loadAll();
    } catch (e: any) {
      const msg = e?.response?.data?.error ?? e?.message ?? 'Ошибка';
      toast.error(String(msg));
    } finally {
      setBusy(false);
    }
  };

  const saveSchedule = async () => {
    const url = feedUrl.trim();
    if (!url) {
      toast.error('Укажите URL фида');
      return;
    }
    setSavingSchedule(true);
    try {
      await feedApi.saveFeedSettings({
        feed_url: url,
        sync_interval_minutes: syncIntervalMinutes,
      });
      toast.success('Расписание сохранено');
      await loadAll();
    } catch (e: any) {
      const msg = e?.response?.data?.error?.message ?? e?.message ?? 'Ошибка сохранения';
      toast.error(String(msg));
    } finally {
      setSavingSchedule(false);
    }
  };

  const schedulerOn = Boolean(status?.feed_sync_scheduler_enabled && (status.feed_sync_interval_minutes ?? 0) > 0);

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-8">
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-1">Загрузка фида</h2>
        <p className="text-sm text-gray-500">
          Каталог товаров для сервиса «Похожие»: ручная загрузка и автоматическое обновление по расписанию.
        </p>
      </div>

      {status && (
        <div>
          <h3 className="text-sm font-semibold text-gray-700 mb-3">Статус данных</h3>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <StatusCard
              title="Каталог"
              ok={Boolean(status.database_imported && (status.db_products_count ?? 0) > 0)}
              value={
                status.db_products_count
                  ? `${status.db_products_count.toLocaleString('ru-RU')} товаров`
                  : 'Пусто'
              }
              meta={
                status.database_imported
                  ? 'Цветомодели в PostgreSQL (после cc-central фида ~8 000)'
                  : 'Загрузите фид, чтобы наполнить каталог'
              }
            />
            <StatusCard
              title="Последняя загрузка фида"
              ok={Boolean(status.feed_uploaded)}
              value={status.feed_uploaded ? 'Загружен' : 'Не загружался'}
              meta={
                <>
                  <div>Обновлён: {formatDateTime(status.feed_last_sync)}</div>
                  {status.feed_next_sync && schedulerOn && (
                    <div>Следующий автозапуск: {formatDateTime(status.feed_next_sync)}</div>
                  )}
                </>
              }
            />
            <StatusCard
              title="Автообновление"
              ok={schedulerOn}
              value={schedulerOn ? 'Включено' : 'Выключено'}
              meta={
                schedulerOn
                  ? `Интервал: ${intervalLabel(status.feed_sync_interval_minutes)}`
                  : 'Расписание не настроено или интервал = 0'
              }
            />
          </div>
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-semibold text-gray-800">
          <CloudArrowUpIcon className="w-5 h-5 text-primary-600" />
          URL фида и ручная загрузка
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">URL XML-фида</label>
          <div className="flex flex-wrap items-stretch gap-3">
            <input
              value={feedUrl}
              onChange={(e) => setFeedUrl(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && refreshFeed()}
              placeholder="https://catalog.example.com/products.xml"
              className="flex-1 min-w-[280px] border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
            <button
              onClick={refreshFeed}
              disabled={busy}
              className="bg-primary-600 hover:bg-primary-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-5 py-2 inline-flex items-center gap-2"
            >
              <ArrowPathIcon className={`w-4 h-4 ${busy ? 'animate-spin' : ''}`} />
              {busy ? 'Загрузка…' : 'Загрузить сейчас'}
            </button>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-semibold text-gray-800">
          <CalendarDaysIcon className="w-5 h-5 text-primary-600" />
          Расписание автозагрузки
        </div>
        <p className="text-xs text-gray-500">
          Интервал сохраняется в базе. После изменения интервала перезапустите backend, чтобы планировщик
          подхватил новый tick.
        </p>
        <div className="grid sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Периодичность</label>
            <select
              value={syncIntervalMinutes}
              onChange={(e) => setSyncIntervalMinutes(Number(e.target.value))}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {INTERVAL_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-end">
            <button
              onClick={saveSchedule}
              disabled={savingSchedule}
              className="bg-gray-900 hover:bg-gray-800 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-5 py-2 inline-flex items-center gap-2"
            >
              <CubeIcon className="w-4 h-4" />
              {savingSchedule ? 'Сохранение…' : 'Сохранить расписание'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
