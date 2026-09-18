import { SimilarStrategyDocument } from '../types/strategy';
import { normalizeStrategy } from './normalizeStrategy';

export const PRESET_EXPORT_VERSION = 1;

export interface SimilarStrategyPresetExport {
  export_version: number;
  exported_at: string;
  name: string;
  strategy: SimilarStrategyDocument;
}

function slugifyPresetName(name: string): string {
  const slug = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9\u0400-\u04ff]+/gi, '-')
    .replace(/^-+|-+$/g, '');
  return slug || 'preset';
}

export function buildPresetExportFilename(name: string, at = new Date()): string {
  const stamp = at.toISOString().slice(0, 10).replace(/-/g, '');
  return `similar-preset-${slugifyPresetName(name)}-${stamp}.json`;
}

export function buildPresetExportPayload(
  name: string,
  strategy: SimilarStrategyDocument,
  at = new Date()
): SimilarStrategyPresetExport {
  return {
    export_version: PRESET_EXPORT_VERSION,
    exported_at: at.toISOString(),
    name: name.trim() || 'Без названия',
    strategy: normalizeStrategy(strategy),
  };
}

export function serializePresetExport(name: string, strategy: SimilarStrategyDocument): string {
  return JSON.stringify(buildPresetExportPayload(name, strategy), null, 2);
}

export function downloadPresetExport(name: string, strategy: SimilarStrategyDocument): void {
  const payload = buildPresetExportPayload(name, strategy);
  const blob = new Blob([JSON.stringify(payload, null, 2)], {
    type: 'application/json;charset=utf-8',
  });
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = buildPresetExportFilename(payload.name);
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(url);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function extractStrategy(raw: Record<string, unknown>): SimilarStrategyDocument | null {
  if (isRecord(raw.strategy)) {
    return normalizeStrategy(raw.strategy as Partial<SimilarStrategyDocument>);
  }
  if (typeof raw.schema_version === 'number' || isRecord(raw.weights) || isRecord(raw.title)) {
    return normalizeStrategy(raw as Partial<SimilarStrategyDocument>);
  }
  return null;
}

export function parsePresetImportText(text: string): { name: string; strategy: SimilarStrategyDocument } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    throw new Error('Файл не является корректным JSON');
  }
  if (!isRecord(parsed)) {
    throw new Error('Ожидается JSON-объект с полем strategy');
  }

  const strategy = extractStrategy(parsed);
  if (!strategy) {
    throw new Error('Не найден документ стратегии (поле strategy или корень документа)');
  }

  const name =
    (typeof parsed.name === 'string' && parsed.name.trim()) ||
    (typeof parsed.preset_name === 'string' && parsed.preset_name.trim()) ||
    'Импорт';

  return { name: name.trim(), strategy };
}

export async function readPresetImportFile(file: File): Promise<{ name: string; strategy: SimilarStrategyDocument }> {
  const text = await file.text();
  return parsePresetImportText(text);
}
