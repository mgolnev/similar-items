import axios from 'axios';
import { API_BASE_URL } from './api';
import {
  RecommendationCompositionConfig,
  RecommendationCompositionPreview,
} from '../types/recommendationComposition';
import { normalizeRecommendationComposition } from '../utils/normalizeRecommendationComposition';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

export const recommendationCompositionApi = {
  getStrategy: async (): Promise<{
    strategy: RecommendationCompositionConfig;
    builtin_default: RecommendationCompositionConfig;
  }> => {
    const response = await api.get('/api/admin/recommendation-composition');
    return {
      strategy: normalizeRecommendationComposition(response.data?.strategy),
      builtin_default: normalizeRecommendationComposition(response.data?.builtin_default),
    };
  },

  saveStrategy: async (
    strategy: RecommendationCompositionConfig,
    updatedReason: string
  ): Promise<RecommendationCompositionConfig> => {
    const response = await api.put('/api/admin/recommendation-composition', {
      strategy,
      updated_reason: updatedReason,
    });
    return normalizeRecommendationComposition(response.data?.strategy);
  },

  resetStrategy: async (): Promise<RecommendationCompositionConfig> => {
    const response = await api.post('/api/admin/recommendation-composition/reset');
    return normalizeRecommendationComposition(response.data?.strategy);
  },

  preview: async (
    productId: string,
    section: string,
    limit: number,
    strategy: RecommendationCompositionConfig
  ): Promise<RecommendationCompositionPreview> => {
    const response = await api.post('/api/admin/recommendation-composition/preview', {
      product_id: productId,
      section,
      limit,
      strategy,
    });
    return {
      items: Array.isArray(response.data?.items) ? response.data.items : [],
      warning: response.data?.warning,
    };
  },
};
