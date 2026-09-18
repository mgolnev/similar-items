package database

const createSimilarStrategyPresetsTable = `
CREATE TABLE IF NOT EXISTS similar_strategy_presets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    document_json TEXT NOT NULL,
    is_production BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_similar_strategy_presets_production ON similar_strategy_presets(is_production);
`
