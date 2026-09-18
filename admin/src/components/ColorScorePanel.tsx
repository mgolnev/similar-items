import React from 'react';
import {
  ColorScoreExplain,
  PARTIAL_COLOR_MATCH_LABEL,
  isPartialColorMatch,
} from '../utils/colorMatch';

const MATCH_PATH_LABEL: Record<string, string> = {
  raw_exact: 'точное совпадение строки',
  primary: 'совпал primary-токен',
  alias: 'совпал токен-алиас',
  family: 'общее цветовое семейство',
  multi_family: 'пересечение multi-color семейств',
  conflict: 'конфликт семейств (black/white, red/green)',
  multi_mismatch: 'одноцветный и составной цвет — без оценки',
  none: 'нет совпадения',
};

export const ColorFamilyChips: React.FC<{
  families: string[];
  matched?: Set<string>;
  palette: Record<string, string>;
}> = ({ families, matched, palette }) => {
  if (families.length === 0) {
    return <span className="text-xs text-slate-400">семейства не найдены</span>;
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {families.map((id) => {
        const hit = matched?.has(id);
        const hex = palette[id] || '#D1D5DB';
        return (
          <span
            key={id}
            title={id}
            className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-mono ${
              hit ? 'border-emerald-400 bg-emerald-50 text-emerald-900' : 'border-slate-200 bg-white text-slate-700'
            }`}
          >
            <span
              className="inline-block h-2.5 w-2.5 rounded-full border border-black/10"
              style={{ backgroundColor: hex }}
            />
            {id}
          </span>
        );
      })}
    </div>
  );
};

export const ColorScorePanel: React.FC<{
  explain: ColorScoreExplain;
  palette: Record<string, string>;
}> = ({ explain, palette }) => {
  const matched = new Set(explain.matchedFamilies);
  const matchLabel = isPartialColorMatch(explain)
    ? PARTIAL_COLOR_MATCH_LABEL
    : MATCH_PATH_LABEL[explain.matchPath] || explain.matchPath;
  return (
    <div className="space-y-3 text-xs text-slate-600">
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <p className="font-medium text-slate-700">База</p>
          <p className="font-mono text-slate-800 break-words">{explain.baseColor}</p>
          <ColorFamilyChips families={explain.baseFamilies} palette={palette} />
          <p className="text-[11px] text-slate-500">
            токены: {explain.baseTokens.join(' · ') || '—'}
          </p>
        </div>
        <div className="space-y-1.5">
          <p className="font-medium text-slate-700">Кандидат</p>
          <p className="font-mono text-slate-800 break-words">{explain.candColor}</p>
          <ColorFamilyChips families={explain.candFamilies} matched={matched} palette={palette} />
          <p className="text-[11px] text-slate-500">
            токены: {explain.candTokens.join(' · ') || '—'}
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-violet-50 px-3 py-2">
        <span className="text-slate-500">{matchLabel}</span>
        {explain.rawExactScore > 0 && <Part label="raw" value={explain.rawExactScore} />}
        {explain.primaryScore > 0 && <Part label="primary" value={explain.primaryScore} />}
        {explain.aliasScore > 0 && <Part label="alias" value={explain.aliasScore} />}
        {explain.familyScore > 0 && <Part label="family" value={explain.familyScore} />}
        {explain.shadeBonus > 0 && <Part label="shade+" value={explain.shadeBonus} />}
        {explain.conflict && (
          <span className="font-semibold text-red-700">штраф {explain.rawBeforeCap}</span>
        )}
        <span className="text-slate-400">→</span>
        <span className="font-semibold text-violet-900">итог: {explain.finalScore}</span>
        {explain.maxScore > 0 && explain.rawBeforeCap > explain.maxScore && (
          <span className="text-amber-700">обрезано max_score={explain.maxScore}</span>
        )}
      </div>
    </div>
  );
};

const Part: React.FC<{ label: string; value: number }> = ({ label, value }) => (
  <span>
    <span className="text-slate-400">{label}</span>{' '}
    <span className="tabular-nums font-medium text-slate-800">{value}</span>
  </span>
);
