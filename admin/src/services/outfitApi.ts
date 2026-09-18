import axios from 'axios';
import { API_BASE_URL } from './api';
import {
  OutfitPreviewResult,
  OutfitStrategyDocument,
  OutfitWarmStatus,
} from '../types/outfit';
import { normalizeOutfitStrategy } from '../utils/normalizeOutfitStrategy';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

export const outfitApi = {
  getStrategy: async (): Promise<{
    strategy: OutfitStrategyDocument;
    builtin_default: OutfitStrategyDocument;
  }> => {
    const response = await api.get('/api/admin/outfit-strategy');
    return {
      strategy: normalizeOutfitStrategy(response.data?.strategy),
      builtin_default: normalizeOutfitStrategy(response.data?.builtin_default),
    };
  },

  saveStrategy: async (
    strategy: OutfitStrategyDocument,
    updatedReason: string
  ): Promise<OutfitStrategyDocument> => {
    const response = await api.put('/api/admin/outfit-strategy', {
      strategy,
      updated_reason: updatedReason,
    });
    return normalizeOutfitStrategy(response.data?.strategy);
  },

  resetStrategy: async (): Promise<OutfitStrategyDocument> => {
    const response = await api.post('/api/admin/outfit-strategy/reset');
    return normalizeOutfitStrategy(response.data?.strategy);
  },

  preview: async (
    productId: string,
    strategy: OutfitStrategyDocument
  ): Promise<OutfitPreviewResult> => {
    const response = await api.post('/api/admin/outfit-strategy/preview', {
      product_id: productId,
      strategy,
    });
    return response.data;
  },

  getWarmStatus: async (): Promise<OutfitWarmStatus> => {
    const response = await api.get('/api/admin/outfit-strategy/warm-status');
    return response.data;
  },
};
