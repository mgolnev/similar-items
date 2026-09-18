import React from 'react';
import {
  TitleAnalysis,
  TitleScoreExplain,
  TitleToken,
  TitleTokenClass,
} from '../utils/titleMatch';

const CLASS_STYLE: Record<TitleTokenClass, string> = {
  normal: 'bg-sky-50 border-sky-200 text-sky-800',
  model: 'bg-violet-100 border-violet-300 text-violet-900',
  feature: 'bg-amber-50 border-amber-300 text-amber-900',
  stop: 'bg-slate-100 border-slate-200 text-slate-400 line-through',
  short: 'bg-slate-50 border-slate-200 text-slate-300',
};

const CLASS_HINT: Record<TitleTokenClass, string> = {
  normal: 'обычный токен (учитывается в overlap)',
  model: 'модельное слово (+балл за совпадение, до model_max)',
  feature: 'особенность (+балл за совпадение, до feature_max)',
  stop: 'стоп-слово — выбрасывается',
  short: 'короче min_token_length — не учитывается',
};

export const TitleTokenChip: React.FC<{ token: TitleToken }> = ({ token }) => {
  const ring = token.matched
    ? 'ring-2 ring-emerald-400 ring-offset-1'
    : token.synonymMatched
      ? 'ring-2 ring-fuchsia-400 ring-offset-1'
      : '';
  const hint = token.synonymMatched
    ? `${CLASS_HINT[token.cls]} · синоним группы`
    : CLASS_HINT[token.cls];
  return (
    <span
      title={hint}
      className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-mono ${
        CLASS_STYLE[token.cls]
      } ${ring}`}
    >
      {token.text}
    </span>
  );
};

export const TitleTokensRow: React.FC<{
  label: string;
  analysis: TitleAnalysis;
}> = ({ label, analysis }) => (
  <div className="flex flex-wrap items-start gap-2">
    <span className="mt-0.5 w-20 shrink-0 text-xs font-medium text-slate-500">{label}</span>
    <div className="flex flex-wrap gap-1.5">
      {analysis.tokens.length === 0 ? (
        <span className="text-xs text-slate-400">—</span>
      ) : (
        analysis.tokens.map((t, i) => <TitleTokenChip key={`${t.text}-${i}`} token={t} />)
      )}
    </div>
  </div>
);

/** Токены + нормализованная строка (как во вкладке «Название» playground). */
export const TitleTokensBlock: React.FC<{
  label: string;
  analysis: TitleAnalysis;
  variant?: 'inline' | 'stacked';
}> = ({ label, analysis, variant = 'inline' }) => {
  const chips =
    analysis.tokens.length === 0 ? (
      <span className="text-xs text-slate-400">—</span>
    ) : (
      analysis.tokens.map((t, i) => <TitleTokenChip key={`${t.text}-${i}`} token={t} />)
    );

  if (variant === 'stacked') {
    return (
      <div className="space-y-2">
        <p className="text-xs font-medium text-slate-500">{label}</p>
        <div className="flex flex-wrap gap-1.5">{chips}</div>
        <p className="text-[11px] text-slate-500 break-words">
          Нормализовано:{' '}
          <span className="font-mono text-slate-600">{analysis.normalized || '—'}</span>
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-1">
      <TitleTokensRow label={label} analysis={analysis} />
      <p className="text-[11px] text-slate-500 pl-[5.5rem] break-words">
        Нормализовано: <span className="font-mono">{analysis.normalized || '—'}</span>
      </p>
    </div>
  );
};

export const TitleTokensLegend: React.FC = () => (
  <div className="flex flex-wrap gap-3 text-[11px] text-slate-500">
    <LegendItem cls="normal" label="обычный (overlap)" />
    <LegendItem cls="model" label="модельное" />
    <LegendItem cls="feature" label="особенность" />
    <LegendItem cls="stop" label="стоп-слово" />
    <LegendItem cls="short" label="короткое (откинуто)" />
    <span className="inline-flex items-center gap-1">
      <span className="inline-block h-3 w-3 rounded ring-2 ring-emerald-400" />
      совпало у обоих
    </span>
    <span className="inline-flex items-center gap-1">
      <span className="inline-block h-3 w-3 rounded ring-2 ring-fuchsia-400" />
      синоним
    </span>
  </div>
);

const LegendItem: React.FC<{ cls: TitleTokenClass; label: string }> = ({ cls, label }) => (
  <span className="inline-flex items-center gap-1">
    <span className={`inline-block h-3 w-3 rounded border ${CLASS_STYLE[cls]}`} />
    {label}
  </span>
);

function fmt(n: number): string {
  return Number.isInteger(n) ? String(n) : n.toFixed(2);
}

export const TitleScorePanel: React.FC<{
  explain: TitleScoreExplain;
  hasCategoryMatch: boolean;
  showCandidate?: boolean;
  hideBaseTokens?: boolean;
}> = ({ explain, hasCategoryMatch, showCandidate = true, hideBaseTokens = false }) => {
  if (!explain.enabled) {
    return (
      <p className="text-xs text-amber-700">
        Title matching выключен — балл по названию не начисляется.
      </p>
    );
  }
  return (
    <div className="space-y-2">
      {!hideBaseTokens && <TitleTokensRow label="База" analysis={explain.base} />}
      {showCandidate && <TitleTokensRow label="Кандидат" analysis={explain.cand} />}

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-slate-50 px-3 py-2 text-xs text-slate-600">
        <FormulaPart
          label="exact"
          value={explain.exactScore}
          hint={explain.exactMatch ? 'названия совпали целиком' : 'нет точного совпадения'}
        />
        <FormulaPart
          label={explain.overlapMode === 'one_sided' ? 'overlap |∩|/|база|' : 'jaccard'}
          value={explain.jaccardScore}
          hint={`${explain.overlapNum}/${explain.overlapDen || 0} × max = ${fmt(
            explain.jaccardScore
          )}`}
        />
        <FormulaPart
          label="model"
          value={explain.modelScore}
          hint={`${explain.matchedModel.length} совпадений (до model_max)`}
        />
        <FormulaPart
          label="feature"
          value={explain.featureScore}
          hint={
            explain.featureRawScore !== explain.featureScore
              ? `${explain.matchedFeature.length} совпадений: ${fmt(explain.featureRawScore)} до проверки категории → ${fmt(explain.featureScore)}`
              : `${explain.matchedFeature.length} совпадений (до feature_max)`
          }
        />
        <FormulaPart
          label="synonym"
          value={explain.synonymScore}
          hint={
            explain.matchedSynonymGroups.length
              ? `группы: ${explain.matchedSynonymGroups.join(', ')}`
              : 'нет чистых синонимичных совпадений'
          }
        />
        <span className="text-slate-400">→</span>
        <span className="font-semibold text-violet-800">
          итог по названию: {explain.finalScore}
        </span>
        {explain.noCategoryCapped && (
          <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[11px] text-amber-800">
            обрезано no_category_cap (категория не совпала)
          </span>
        )}
        {explain.categoryGate === 'feature_blocked_weak_category' && (
          <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[11px] text-amber-800">
            feature=0 из-за слабого совпадения категории
          </span>
        )}
        {explain.categoryGate === 'feature_half_mid_parent' && (
          <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[11px] text-amber-800">
            feature ×½ из-за родительской категории
          </span>
        )}
        {explain.globallyCapped && (
          <span className="rounded bg-amber-100 px-1.5 py-0.5 text-[11px] text-amber-800">
            обрезано общим cap
          </span>
        )}
        {!hasCategoryMatch && !explain.noCategoryCapped && (
          <span className="text-[11px] text-slate-400">категория не совпала</span>
        )}
      </div>
    </div>
  );
};

const FormulaPart: React.FC<{ label: string; value: number; hint?: string }> = ({
  label,
  value,
  hint,
}) => (
  <span className="inline-flex items-center gap-1" title={hint}>
    <span className="text-slate-400">{label}</span>
    <span className={`tabular-nums font-medium ${value > 0 ? 'text-slate-800' : 'text-slate-300'}`}>
      {fmt(value)}
    </span>
  </span>
);
