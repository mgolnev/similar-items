import React, { useMemo, useState } from 'react';
import { FeatureCatalogEntry } from '../types/featureCatalog';

type Props = {
  features: FeatureCatalogEntry[];
  loading?: boolean;
  onPick: (entry: FeatureCatalogEntry) => void;
};

export const SimilarStrategyFeatureCatalog: React.FC<Props> = ({
  features,
  loading,
  onPick,
}) => {
  const [filter, setFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState<string>('all');

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase();
    return features.filter((f) => {
      if (sourceFilter !== 'all' && f.source !== sourceFilter) return false;
      if (!q) return true;
      return (
        f.key.toLowerCase().includes(q) ||
        f.label_ru.toLowerCase().includes(q) ||
        (f.column || '').toLowerCase().includes(q) ||
        f.description.toLowerCase().includes(q)
      );
    });
  }, [features, filter, sourceFilter]);

  return (
    <div className="border border-slate-200 rounded-xl bg-slate-50/80 overflow-hidden">
      <div className="px-3 py-2 border-b border-slate-200 bg-white">
        <p className="text-xs font-semibold text-slate-800">Справочник признаков</p>
        <p className="text-[11px] text-slate-500 mt-0.5">
          Клик — подставить ключ и источник в правило
        </p>
      </div>
      <div className="p-2 space-y-2 border-b border-slate-100">
        <input
          type="text"
          placeholder="Поиск…"
          className="w-full rounded border border-slate-300 px-2 py-1 text-xs"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
        <select
          className="w-full rounded border border-slate-300 px-2 py-1 text-xs"
          value={sourceFilter}
          onChange={(e) => setSourceFilter(e.target.value)}
        >
          <option value="all">Все источники</option>
          <option value="column">column</option>
          <option value="attributes">attributes</option>
          <option value="computed">computed</option>
        </select>
      </div>
      <div className="max-h-64 overflow-y-auto text-xs">
        {loading && <p className="p-3 text-slate-500">Загрузка…</p>}
        {!loading && filtered.length === 0 && (
          <p className="p-3 text-slate-500">Ничего не найдено</p>
        )}
        {filtered.map((f) => (
          <button
            key={`${f.source}:${f.key}`}
            type="button"
            className="w-full text-left px-3 py-2 hover:bg-violet-50 border-b border-slate-100 last:border-0"
            onClick={() => onPick(f)}
          >
            <span className="font-mono text-violet-800">{f.key}</span>
            <span className="text-slate-400 ml-1">· {f.source}</span>
            {f.column && (
              <span className="block text-[10px] text-slate-400">колонка: {f.column}</span>
            )}
            <span className="block text-slate-600 truncate">{f.label_ru}</span>
            {f.sample_count != null && f.sample_count > 0 && (
              <span className="text-[10px] text-slate-400">в фиде: {f.sample_count}</span>
            )}
          </button>
        ))}
      </div>
    </div>
  );
};
