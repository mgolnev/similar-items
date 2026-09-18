import axios from 'axios';

/** Явный backend URL или пустая строка для CRA proxy. */
export const API_BASE_URL = (
  process.env.REACT_APP_API_URL ||
  process.env.REACT_APP_API_BASE_URL ||
  ''
).trim();

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
});

export interface FeedDataStatus {
  database_imported?: boolean;
  db_products_count?: number;
  feed_uploaded?: boolean;
  feed_last_sync?: string;
  feed_next_sync?: string;
  feed_sync_interval_minutes?: number;
  feed_sync_scheduler_enabled?: boolean;
  default_feed_url?: string;
}

export interface FeedSettings {
  feed_url: string;
  sync_interval_minutes: number;
  scheduler_enabled: boolean;
  sync_interval_persisted?: boolean;
}

export const feedApi = {
  processFeed: async (feedUrl?: string): Promise<any> => {
    const response = await api.get('/api/process-feed', {
      params: feedUrl ? { url: feedUrl } : undefined,
    });
    return response.data;
  },

  getDataStatus: async (): Promise<FeedDataStatus> => {
    const response = await api.get('/api/data-status');
    return response.data;
  },

  getFeedSettings: async (): Promise<FeedSettings> => {
    const response = await api.get('/api/admin/feed-settings');
    return response.data;
  },

  saveFeedSettings: async (settings: {
    feed_url: string;
    sync_interval_minutes: number;
  }): Promise<FeedSettings> => {
    const response = await api.put('/api/admin/feed-settings', settings);
    return response.data;
  },
};
