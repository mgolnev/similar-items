import axios from 'axios';
import { API_BASE_URL } from './api';
import { FeatureCatalogResponse } from '../types/featureCatalog';
import {
  SimilarPreviewResult,
  SimilarStrategyDocument,
} from '../types/strategy';
import { normalizeStrategy } from '../utils/normalizeStrategy';

export interface VisualPreviewRow {
  base: {
    id: string;
    sku: string;
    name: string;
    leaf_category?: string;
    image_url: string;
  };
  items: Array<{
    id: string;
    sku: string;
    name: string;
    image_url: string;
    score?: number;
  }>;
  error?: string;
}

export interface VisualPreviewResult {
  section: string;
  sample_size: number;
  similar_limit: number;
  rows: VisualPreviewRow[];
  source: 'serving_cache';
  cache_ready: boolean;
  serving_generation: number;
}

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

export type CategoryAuditParams = {
  section?: string;
  topN?: number;
  maxAvg?: number | '';
  minSample?: number;
};

export interface StrategyPresetSummary {
  id: string;
  name: string;
  is_production: boolean;
  updated_at?: string;
}

export interface StrategyPresetView extends StrategyPresetSummary {
  strategy: SimilarStrategyDocument;
}

export interface StrategyPresetListResponse {
  presets: StrategyPresetSummary[];
  production_id: string;
}

export const strategyApi = {
  getStrategy: async (): Promise<{
    strategy: SimilarStrategyDocument;
    builtin_default: SimilarStrategyDocument;
    persisted: boolean;
    production_id?: string;
    presets?: StrategyPresetSummary[];
  }> => {
    const res = await api.get('/api/admin/similar-strategy');
    const data = res.data ?? {};
    return {
      strategy: normalizeStrategy(data.strategy),
      builtin_default: normalizeStrategy(data.builtin_default),
      persisted: Boolean(data.persisted),
      production_id: data.production_id,
      presets: data.presets,
    };
  },

  listPresets: async (): Promise<StrategyPresetListResponse> => {
    const res = await api.get('/api/admin/similar-strategy/presets');
    return res.data ?? { presets: [], production_id: '' };
  },

  getPreset: async (id: string): Promise<StrategyPresetView> => {
    const res = await api.get(`/api/admin/similar-strategy/presets/${encodeURIComponent(id)}`);
    const data = res.data ?? {};
    return {
      ...data,
      strategy: normalizeStrategy(data.strategy),
    };
  },

  createPreset: async (
    name: string,
    strategy: SimilarStrategyDocument
  ): Promise<StrategyPresetView> => {
    const res = await api.post('/api/admin/similar-strategy/presets', { name, strategy });
    const data = res.data ?? {};
    return { ...data, strategy: normalizeStrategy(data.strategy) };
  },

  updatePreset: async (
    id: string,
    name: string,
    strategy: SimilarStrategyDocument,
    updatedReason: string
  ): Promise<StrategyPresetView> => {
    const res = await api.put(`/api/admin/similar-strategy/presets/${encodeURIComponent(id)}`, {
      name,
      strategy,
      updated_reason: updatedReason,
    });
    const data = res.data ?? {};
    return { ...data, strategy: normalizeStrategy(data.strategy) };
  },

  deletePreset: async (id: string): Promise<void> => {
    await api.delete(`/api/admin/similar-strategy/presets/${encodeURIComponent(id)}`);
  },

  promotePreset: async (id: string): Promise<StrategyPresetView> => {
    const res = await api.post(
      `/api/admin/similar-strategy/presets/${encodeURIComponent(id)}/promote`
    );
    // новый формат ответа: { preset, warm_started }; старый — view напрямую
    const data = res.data ?? {};
    const view = data.preset ?? data;
    return { ...view, strategy: normalizeStrategy(view.strategy) };
  },

  saveStrategy: async (
    strategy: SimilarStrategyDocument,
    updatedReason: string
  ): Promise<SimilarStrategyDocument> => {
    const res = await api.put('/api/admin/similar-strategy', {
      strategy,
      updated_reason: updatedReason,
    });
    return normalizeStrategy(res.data?.strategy);
  },

  previewSimilar: async (
    productId: string,
    section: string,
    limit = 12,
    strategy?: SimilarStrategyDocument
  ): Promise<SimilarPreviewResult> => {
    const res = await api.post('/api/admin/similar-strategy/preview', {
      product_id: productId,
      section,
      limit,
      strategy,
    });
    return res.data;
  },

  visualPreview: async (
    section: string,
    sampleSize = 10,
    similarLimit = 12
  ): Promise<VisualPreviewResult> => {
    const res = await api.post('/api/admin/similar-strategy/visual-preview', {
      section,
      sample_size: sampleSize,
      similar_limit: similarLimit,
    });
    return res.data;
  },

  getFeatureCatalog: async (): Promise<FeatureCatalogResponse> => {
    const res = await api.get('/api/admin/similar-strategy/feature-catalog');
    return res.data ?? { features: [] };
  },

  /** Скачать CSV аудита категорий (может занять десятки секунд на полном каталоге). */
  downloadCategoryAudit: async (params: CategoryAuditParams = {}): Promise<void> => {
    const qs = new URLSearchParams();
    qs.set('section', params.section || 'all');
    qs.set('top_n', String(params.topN ?? 30));
    qs.set('min_sample', String(params.minSample ?? 5));
    if (params.maxAvg !== '' && params.maxAvg != null && Number(params.maxAvg) > 0) {
      qs.set('max_avg', String(params.maxAvg));
    }
    const res = await api.get(`/api/admin/similar-strategy/category-audit.csv?${qs}`, {
      responseType: 'blob',
      timeout: 10 * 60 * 1000,
    });
    const disposition = String(res.headers['content-disposition'] || '');
    const match = /filename="?([^"]+)"?/i.exec(disposition);
    const filename = match?.[1] || `category-audit-${params.section || 'all'}.csv`;
    const url = window.URL.createObjectURL(res.data);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  },
};
