import React, { useEffect, useMemo, useState } from 'react';
import { TitleConfigLike, analyzeTitle, explainTitleScore } from '../utils/titleMatch';
import {
  TitleScorePanel,
  TitleTokensBlock,
  TitleTokensLegend,
} from '../components/TitleTokens';

type Props = {
  config: TitleConfigLike;
};

type PlaygroundDraft = {
  baseName: string;
  candName: string;
  categoryMatch: boolean;
};

const DRAFT_KEY = 'similar-strategy-title-playground';
const DEFAULT_DRAFT: PlaygroundDraft = {
  baseName: 'Брюки палаццо широкие Clever',
  candName: '',
  categoryMatch: true,
};

function loadDraft(): PlaygroundDraft {
  try {
    const raw = sessionStorage.getItem(DRAFT_KEY);
    if (!raw) return DEFAULT_DRAFT;
    const parsed = JSON.parse(raw) as Partial<PlaygroundDraft>;
    return {
      baseName: typeof parsed.baseName === 'string' ? parsed.baseName : DEFAULT_DRAFT.baseName,
      candName: typeof parsed.candName === 'string' ? parsed.candName : DEFAULT_DRAFT.candName,
      categoryMatch:
        typeof parsed.categoryMatch === 'boolean'
          ? parsed.categoryMatch
          : DEFAULT_DRAFT.categoryMatch,
    };
  } catch {
    return DEFAULT_DRAFT;
  }
}

// Интерактивный разбор названия «по словам» прямо во вкладке «Названия».
// Реагирует на несохранённые изменения stopwords / model_words / порогов.
export const SimilarStrategyTitlePlayground: React.FC<Props> = ({ config }) => {
  const [draft, setDraft] = useState<PlaygroundDraft>(loadDraft);
  const { baseName, candName, categoryMatch } = draft;

  useEffect(() => {
    sessionStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
  }, [draft]);

  const compare = candName.trim().length > 0;

  const baseAnalysis = useMemo(() => analyzeTitle(baseName, config), [baseName, config]);
  const explain = useMemo(
    () => (compare ? explainTitleScore(baseName, candName, categoryMatch, config) : null),
    [compare, baseName, candName, categoryMatch, config]
  );

  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50/60 p-4 space-y-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-semibold text-slate-800">Разбор названия по словам</p>
        <TitleTokensLegend />
      </div>
      <p className="text-xs text-slate-500">
        Поле слева — название. Чипы пересчитываются вживую под текущие списки и пороги (даже
        несохранённые). Заполните «Кандидат», чтобы увидеть совпадения и итоговый балл.
      </p>

      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">Название (база)</label>
          <input
            type="text"
            value={baseName}
            onChange={(e) => setDraft((d) => ({ ...d, baseName: e.target.value }))}
            placeholder="Например: Брюки палаццо широкие"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Кандидат (необязательно)
          </label>
          <input
            type="text"
            value={candName}
            onChange={(e) => setDraft((d) => ({ ...d, candName: e.target.value }))}
            placeholder="Для сравнения двух названий"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
          />
        </div>
      </div>

      {!compare && (
        <div className="space-y-2 rounded-lg border border-slate-200 bg-white p-3">
          <TitleTokensBlock label="Токены" analysis={baseAnalysis} />
        </div>
      )}

      {compare && explain && (
        <div className="space-y-2 rounded-lg border border-slate-200 bg-white p-3">
          <label className="flex items-center gap-2 text-xs text-slate-600">
            <input
              type="checkbox"
              checked={categoryMatch}
              onChange={(e) => setDraft((d) => ({ ...d, categoryMatch: e.target.checked }))}
              className="rounded border-slate-300 text-violet-600"
            />
            Категории совпадают (иначе действует no_category_cap)
          </label>
          <TitleScorePanel explain={explain} hasCategoryMatch={categoryMatch} />
        </div>
      )}
    </div>
  );
};
