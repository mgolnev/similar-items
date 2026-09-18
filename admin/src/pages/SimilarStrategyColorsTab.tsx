import React from 'react';
import { SimilarStrategyDocument } from '../types/strategy';

type Props = {
  doc: SimilarStrategyDocument;
  onChange: (doc: SimilarStrategyDocument) => void;
};

const DEFAULT_PICKER_COLOR = '#D1D5DB';
const isHexColor = (value: string | undefined) => Boolean(value && /^#[0-9a-fA-F]{6}$/.test(value));

export const SimilarStrategyColorsTab: React.FC<Props> = ({ doc, onChange }) => {
  const families = doc.color_family_dictionary || {};
  const palette = doc.color_family_palette || {};

  const updateFamily = (key: string, aliases: string[]) => {
    onChange({
      ...doc,
      color_family_dictionary: { ...families, [key]: aliases },
    });
  };

  const updateFamilyColor = (key: string, color: string) => {
    onChange({
      ...doc,
      color_family_palette: { ...palette, [key]: color },
    });
  };

  const removeFamily = (key: string) => {
    const next = { ...families };
    const nextPalette = { ...palette };
    delete next[key];
    delete nextPalette[key];
    onChange({ ...doc, color_family_dictionary: next, color_family_palette: nextPalette });
  };

  const addFamily = () => {
    const key = `family_${Date.now().toString(36)}`;
    onChange({
      ...doc,
      color_family_dictionary: { ...families, [key]: ['синоним 1'] },
      color_family_palette: { ...palette, [key]: '#D1D5DB' },
    });
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-600">
        Словарь цветовых семейств для правил с <code className="text-xs bg-slate-100 px-1 rounded">match_type=color_similarity</code>.
        Ключ семейства (например <code className="text-xs">blue</code>) — список синонимов, по одному на строку.
        Заливка используется для цветных кружков в карточке каталога.
      </p>
      <button type="button" onClick={addFamily} className="btn-secondary text-sm">
        + Добавить семейство
      </button>
      <div className="space-y-3">
        {Object.entries(families).map(([key, aliases]) => {
          const familyColor = isHexColor(palette[key]) ? palette[key] : DEFAULT_PICKER_COLOR;
          return (
          <div key={key} className="border border-slate-200 rounded-lg p-4 bg-slate-50/50">
            <div className="flex flex-wrap gap-2 items-center mb-2">
              <label className="text-xs font-medium text-slate-600">Ключ семейства</label>
              <input
                type="text"
                className="rounded border border-slate-300 px-2 py-1 text-sm font-mono w-32"
                value={key}
                onChange={(e) => {
                  const newKey = e.target.value.trim();
                  if (!newKey || newKey === key) return;
                  const next = { ...families };
                  const nextPalette = { ...palette };
                  delete next[key];
                  next[newKey] = aliases;
                  if (nextPalette[key]) {
                    nextPalette[newKey] = nextPalette[key];
                    delete nextPalette[key];
                  }
                  onChange({ ...doc, color_family_dictionary: next, color_family_palette: nextPalette });
                }}
              />
              <label className="ml-2 flex items-center gap-2 text-xs font-medium text-slate-600">
                Заливка
                <input
                  type="color"
                  className="h-8 w-10 cursor-pointer rounded border border-slate-300 bg-white p-0.5"
                  value={familyColor}
                  onChange={(e) => updateFamilyColor(key, e.target.value)}
                  aria-label={`Цвет заливки для ${key}`}
                />
                <input
                  type="text"
                  className="w-24 rounded border border-slate-300 px-2 py-1 text-sm font-mono uppercase"
                  value={palette[key] || DEFAULT_PICKER_COLOR}
                  onChange={(e) => updateFamilyColor(key, e.target.value)}
                />
              </label>
              <button
                type="button"
                className="text-xs text-red-600 hover:underline ml-auto"
                onClick={() => removeFamily(key)}
              >
                Удалить
              </button>
            </div>
            <textarea
              rows={3}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
              value={(aliases || []).join('\n')}
              onChange={(e) =>
                updateFamily(
                  key,
                  e.target.value
                    .split('\n')
                    .map((s) => s.trim())
                    .filter(Boolean)
                )
              }
              placeholder="синий&#10;голубой&#10;denim"
            />
            <p className="text-xs text-slate-400 mt-1">{(aliases || []).length} синонимов</p>
          </div>
        );
        })}
      </div>
    </div>
  );
};
