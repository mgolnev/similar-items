import { FeatureCatalogEntry } from '../types/featureCatalog';

export function featuresForSource(
  features: FeatureCatalogEntry[],
  source: string
): FeatureCatalogEntry[] {
  const s = source.trim();
  return features.filter((f) => f.source === s);
}

export function hintKeysForRule(
  features: FeatureCatalogEntry[],
  source: string
): string[] {
  return featuresForSource(features, source).map((f) => f.key);
}

export function findFeature(
  features: FeatureCatalogEntry[],
  source: string,
  key: string
): FeatureCatalogEntry | undefined {
  const k = key.trim();
  return features.find((f) => f.source === source && f.key === k);
}
