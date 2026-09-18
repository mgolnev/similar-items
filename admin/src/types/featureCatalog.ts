export interface FeatureCatalogEntry {
  key: string;
  source: string;
  label_ru: string;
  column?: string;
  description: string;
  suggested_match_types?: string[];
  sample_count?: number;
  sample_values?: string[];
}

export interface FeatureCatalogResponse {
  features: FeatureCatalogEntry[];
}
