import React, { useMemo, useState } from 'react';
import { NumberInput } from '../components/NumberInput';
import { TitleScorePanel, TitleTokensLegend } from '../components/TitleTokens';
import { SimilarStrategyDocument, TitleSynonymGroup } from '../types/strategy';
import {
  DEFAULT_TITLE_SYNONYM_GROUPS,
  explainTitleScore,
  normalizeTitleText,
} from '../utils/titleMatch';

type Props = {
  doc: SimilarStrategyDocument;
  onChange: (doc: SimilarStrategyDocument) => void;
};

function wordConflicts(groups: TitleSynonymGroup[]): Map<string, string[]> {
  const seen = new Map<string, string>();
  const conflicts = new Map<string, string[]>();
  for (const g of groups) {
    for (const raw of g.words || []) {
      const w = normalizeTitleText(raw);
      if (!w) continue;
      const prev = seen.get(w);
      if (prev && prev !== g.id) {
        const list = conflicts.get(w) || [prev];
        if (!list.includes(g.id)) list.push(g.id);
        conflicts.set(w, list);
      } else {
        seen.set(w, g.id);
      }
    }
  }
  return conflicts;
}

export const SimilarStrategySynonymsTab: React.FC<Props> = ({ doc, onChange }) => {
  const groups = useMemo(
    () => doc.title.synonym_groups || [],
    [doc.title.synonym_groups]
  );
  const conflicts = useMemo(() => wordConflicts(groups), [groups]);
  const [baseName, setBaseName] = useState('Ветровка оверсайз');
  const [candName, setCandName] = useState('Куртка оверсайз');

  const updateTitle = (patch: Partial<SimilarStrategyDocument['title']>) => {
    onChange({ ...doc, title: { ...doc.title, ...patch } });
  };

  const updateGroup = (idx: number, next: TitleSynonymGroup) => {
    const list = groups.map((g, i) => (i === idx ? next : g));
    updateTitle({ synonym_groups: list });
  };

  const removeGroup = (idx: number) => {
    updateTitle({ synonym_groups: groups.filter((_, i) => i !== idx) });
  };

  const addGroup = () => {
    updateTitle({
      synonym_groups: [
        ...groups,
        { id: `group_${Date.now().toString(36)}`, words: ['слово1', 'слово2'] },
      ],
      // при первом добавлении включаем балл, если ещё выключен
      synonym_max_score: doc.title.synonym_max_score > 0 ? doc.title.synonym_max_score : 15,
      synonym_token_score: doc.title.synonym_token_score > 0 ? doc.title.synonym_token_score : 5,
    });
  };

  const loadDefaults = () => {
    updateTitle({
      synonym_groups: DEFAULT_TITLE_SYNONYM_GROUPS.map((g) => ({
        id: g.id,
        words: [...g.words],
      })),
      synonym_token_score: 5,
      synonym_max_score: 15,
    });
  };

  const explain = useMemo(
    () => explainTitleScore(baseName, candName, true, doc.title),
    [baseName, candName, doc.title]
  );

  return (
    <div className="space-y-5">
      <p className="text-sm text-slate-600">
        Группы равнозначных словоформ для отдельного балла за синонимичность названий.
        Балл начисляется только когда у базы и кандидата есть слова из одной группы, но{' '}
        <em>разные</em> словоформы (например «ветровка» ↔ «куртка»). Одинаковые слова уже
        учитываются в overlap / model / feature — повторно не считаются.
      </p>

      <div className="grid sm:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Балл за группу
          </label>
          <NumberInput
            value={doc.title.synonym_token_score}
            min={0}
            max={200}
            onChange={(v) => updateTitle({ synonym_token_score: v })}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Max балл синонимов
          </label>
          <NumberInput
            value={doc.title.synonym_max_score}
            min={0}
            max={200}
            onChange={(v) => updateTitle({ synonym_max_score: v })}
          />
          <p className="mt-1 text-xs text-slate-500">0 — синонимы выключены (без оверхеда).</p>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="button" onClick={addGroup} className="btn-secondary text-sm">
          + Добавить группу
        </button>
        <button type="button" onClick={loadDefaults} className="btn-outline text-sm">
          Загрузить примеры
        </button>
      </div>

      {conflicts.size > 0 && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Слово в нескольких группах (учитывается только первая):{' '}
          {Array.from(conflicts.entries())
            .map(([w, ids]) => `${w} → ${ids.join(', ')}`)
            .join('; ')}
        </div>
      )}

      <div className="space-y-3">
        {groups.length === 0 && (
          <p className="text-sm text-slate-400">Групп пока нет — добавьте или загрузите примеры.</p>
        )}
        {groups.map((g, idx) => (
          <div key={`${g.id}-${idx}`} className="border border-slate-200 rounded-lg p-4 bg-slate-50/50">
            <div className="flex flex-wrap gap-2 items-center mb-2">
              <label className="text-xs font-medium text-slate-600">ID группы</label>
              <input
                type="text"
                className="rounded border border-slate-300 px-2 py-1 text-sm font-mono w-48"
                value={g.id}
                onChange={(e) => updateGroup(idx, { ...g, id: e.target.value })}
              />
              <button
                type="button"
                className="text-xs text-red-600 hover:underline ml-auto"
                onClick={() => removeGroup(idx)}
              >
                Удалить
              </button>
            </div>
            <textarea
              rows={3}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
              value={(g.words || []).join('\n')}
              onChange={(e) =>
                updateGroup(idx, {
                  ...g,
                  words: e.target.value
                    .split('\n')
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
              placeholder={'ветровка\nкуртка'}
            />
          </div>
        ))}
      </div>

      <div className="rounded-xl border border-slate-200 p-4 space-y-3">
        <p className="text-sm font-semibold text-slate-800">Проверка пары названий</p>
        <TitleTokensLegend />
        <div className="grid sm:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">База</label>
            <input
              type="text"
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
              value={baseName}
              onChange={(e) => setBaseName(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">Кандидат</label>
            <input
              type="text"
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
              value={candName}
              onChange={(e) => setCandName(e.target.value)}
            />
          </div>
        </div>
        <TitleScorePanel explain={explain} hasCategoryMatch />
      </div>
    </div>
  );
};
