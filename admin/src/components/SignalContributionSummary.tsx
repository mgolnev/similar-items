import React from 'react';
import { AttributeRule } from '../types/strategyRules';
import { SimilarExplainItem, SimilarScoreBreakdown } from '../types/strategy';

type Props = {
  rows: SimilarExplainItem[];
  attributeRules?: AttributeRule[];
};

type SignalStat = {
  label: string;
  kind: 'category' | 'additive' | 'rule' | 'title';
  fired: number;
  penalties: number;
  total: number;
  avg: number;
};

function ruleValue(b: SimilarScoreBreakdown, ruleId: string): number {
  if (ruleId === 'color_similarity') {
    return b.rule_scores?.color_similarity ?? b.color ?? 0;
  }
  return b.rule_scores?.[ruleId] ?? 0;
}

// Свод «какой сигнал и насколько часто срабатывает» на текущей выборке —
// помогает уловить вклад каждой настройки, не вчитываясь в таблицу.
export const SignalContributionSummary: React.FC<Props> = ({ rows, attributeRules = [] }) => {
  if (!rows.length) return null;
  const n = rows.length;

  const collect = (
    label: string,
    kind: SignalStat['kind'],
    pick: (b: SimilarScoreBreakdown) => number
  ): SignalStat => {
    let fired = 0;
    let penalties = 0;
    let sum = 0;
    for (const r of rows) {
      const v = pick(r.breakdown);
      if (v !== 0) fired += 1;
      if (v < 0) penalties += 1;
      sum += v;
    }
    return { label, kind, fired, penalties, total: n, avg: sum / n };
  };

  const stats: SignalStat[] = [
    collect('Категория', 'category', (b) => b.category),
    collect('Префикс артикула', 'additive', (b) => b.article_prefix),
    collect('Цена', 'additive', (b) => b.price),
    ...attributeRules
      .filter((r) => r.enabled)
      .map((r) => collect(r.label_ru || r.id, 'rule', (b) => ruleValue(b, r.id))),
    collect('Название', 'title', (b) => b.title),
  ].filter((s) => s.fired > 0);

  if (!stats.length) return null;

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-3">
      <p className="mb-2 text-xs font-semibold text-slate-700">
        Вклад сигналов на этой выборке ({n} кандидат{plural(n)})
      </p>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
        {stats.map((s) => (
          <SignalCard key={`${s.kind}-${s.label}`} stat={s} />
        ))}
      </div>
    </div>
  );
};

const KIND_ACCENT: Record<SignalStat['kind'], string> = {
  category: 'border-l-emerald-400',
  additive: 'border-l-sky-400',
  rule: 'border-l-violet-400',
  title: 'border-l-amber-400',
};

const SignalCard: React.FC<{ stat: SignalStat }> = ({ stat }) => {
  const dead = stat.fired === 0;
  const hasPenalties = stat.penalties > 0;
  const rate = Math.round((stat.fired / stat.total) * 100);
  return (
    <div
      className={`rounded-md border border-l-4 px-2.5 py-1.5 ${
        dead
          ? 'border-slate-200 bg-slate-50'
          : hasPenalties
            ? 'border-slate-200 border-l-red-500 bg-red-50/40'
            : `border-slate-200 ${KIND_ACCENT[stat.kind]}`
      }`}
    >
      <div className={`truncate text-xs font-medium ${dead ? 'text-slate-400' : 'text-slate-800'}`}>
        {stat.label}
      </div>
      <div className="mt-0.5 flex items-baseline justify-between gap-1 text-[11px]">
        <span className={dead ? 'text-slate-400' : 'text-slate-600'}>
          {stat.fired}/{stat.total} · {rate}%
          {hasPenalties ? ` · штрафов ${stat.penalties}` : ''}
        </span>
        <span
          className={`tabular-nums font-semibold ${
            dead ? 'text-slate-300' : stat.avg < 0 ? 'text-red-700' : 'text-slate-800'
          }`}
        >
          ср. {stat.avg.toFixed(1)}
        </span>
      </div>
    </div>
  );
};

function plural(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return '';
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return 'а';
  return 'ов';
}
