import React from 'react';
import { SimilarStrategyDocument } from '../types/strategy';

type Props = {
  doc: SimilarStrategyDocument;
};

// Визуальная схема каскада отбора: как кандидат проходит strict → parent →
// branch → popular, и почему title не может «вытащить» через base-гейт.
export const ThresholdCascadeExplainer: React.FC<Props> = ({ doc }) => {
  const t = doc.thresholds;
  const steps = [
    {
      label: 'Strict',
      cond: `score ≥ ${t.strict.score_min} И база-без-title ≥ ${t.strict.base_without_title_min}`,
      tone: 'bg-emerald-50 border-emerald-200 text-emerald-900',
    },
    {
      label: 'Parent fallback',
      cond: `score ≥ ${t.parent.score_min} И база-без-title ≥ ${t.parent.base_without_title_min}`,
      tone: 'bg-sky-50 border-sky-200 text-sky-900',
    },
    {
      label: 'Branch fallback',
      cond: `score ≥ ${t.branch.score_min}`,
      tone: 'bg-violet-50 border-violet-200 text-violet-900',
    },
    {
      label: 'Popular',
      cond: 'добор популярными из секции',
      tone: 'bg-slate-50 border-slate-200 text-slate-700',
    },
  ];

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4">
      <p className="mb-1 text-sm font-semibold text-slate-800">Как работает отбор</p>
      <p className="mb-3 text-xs text-slate-500">
        Движок применяет уровни сверху вниз и останавливается, как только набралось достаточно
        кандидатов на лимит. Следующий уровень включается только при нехватке.
      </p>
      <div className="flex flex-wrap items-stretch gap-2">
        {steps.map((s, i) => (
          <React.Fragment key={s.label}>
            <div className={`min-w-[170px] flex-1 rounded-lg border px-3 py-2 ${s.tone}`}>
              <div className="text-xs font-semibold">{s.label}</div>
              <div className="mt-0.5 font-mono text-[11px] leading-tight">{s.cond}</div>
            </div>
            {i < steps.length - 1 && (
              <div className="flex items-center text-slate-300">→</div>
            )}
          </React.Fragment>
        ))}
      </div>
      <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
        Важно: на strict/parent отдельно проверяется «база без title». Балл за название
        <b> не может вытащить</b> кандидата через эти уровни, если базовый балл (категория +
        атрибуты + цена) ниже порога. Title влияет только на общий <code>score</code> и сортировку.
      </p>
    </div>
  );
};
