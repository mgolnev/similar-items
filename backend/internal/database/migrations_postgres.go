package database

import "fmt"

const createPostgresSchema = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    id TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    sku TEXT PRIMARY KEY,
    sales_sku TEXT,
    group_id TEXT,
    name TEXT,
    price DOUBLE PRECISION DEFAULT 0,
    oldprice DOUBLE PRECISION DEFAULT 0,
    discount DOUBLE PRECISION DEFAULT 0,
    gender TEXT,
    image_url TEXT,
    sale_start_date TEXT,
    available BOOLEAN DEFAULT FALSE,
    categories TEXT,
    url TEXT,
    size_name TEXT DEFAULT '',
    size_completeness TEXT DEFAULT '0/0',
    size_out_of_stock_percent DOUBLE PRECISION DEFAULT 0,
    store_stock INTEGER DEFAULT 0,
    warehouse_stock INTEGER DEFAULT 0,
    category_path TEXT DEFAULT '[]',
    leaf_category TEXT DEFAULT '',
    root_section TEXT DEFAULT 'unknown',
    subsection TEXT DEFAULT '',
    attributes TEXT DEFAULT '{}',
    normalized_color TEXT DEFAULT '',
    product_type TEXT DEFAULT '',
    publication_status TEXT DEFAULT 'published',
    feed_version TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS sync_status (
    name TEXT PRIMARY KEY,
    last_synced_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS runtime_settings (
    key TEXT PRIMARY KEY,
    value_json TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
`

const createPostgresIndexes = `
CREATE INDEX IF NOT EXISTS idx_products_root_section ON products(root_section);
CREATE INDEX IF NOT EXISTS idx_products_publication_status ON products(publication_status);
CREATE INDEX IF NOT EXISTS idx_products_leaf_category ON products(leaf_category);
CREATE INDEX IF NOT EXISTS idx_products_feed_version ON products(feed_version);
CREATE INDEX IF NOT EXISTS idx_products_normalized_color ON products(normalized_color);
CREATE INDEX IF NOT EXISTS idx_products_price ON products(price);
CREATE INDEX IF NOT EXISTS idx_products_gender ON products(gender);
CREATE INDEX IF NOT EXISTS idx_products_group_key ON products(COALESCE(NULLIF(TRIM(group_id), ''), NULLIF(TRIM(sales_sku), ''), TRIM(sku)));
CREATE INDEX IF NOT EXISTS idx_products_group_id ON products(group_id);
CREATE INDEX IF NOT EXISTS idx_products_sales_sku ON products(sales_sku);
`

const ensurePostgresPrimaryKeys = `
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE t.relname = 'products' AND c.contype = 'p'
  ) THEN
    ALTER TABLE products ADD PRIMARY KEY (sku);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON c.conrelid = t.oid
    WHERE t.relname = 'sync_status' AND c.contype = 'p'
  ) THEN
    ALTER TABLE sync_status ADD PRIMARY KEY (name);
  END IF;
END $$;
`

func RunPostgresMigrations(db *DB) error {
	if _, err := db.Exec(createPostgresSchema); err != nil {
		return fmt.Errorf("create postgres schema: %w", err)
	}
	if _, err := db.Exec(createPostgresIndexes); err != nil {
		return fmt.Errorf("create postgres indexes: %w", err)
	}
	if _, err := db.Exec(ensurePostgresPrimaryKeys); err != nil {
		return fmt.Errorf("ensure postgres primary keys: %w", err)
	}
	if _, err := db.Exec(createSimilarStrategyPresetsTable); err != nil {
		return fmt.Errorf("create similar_strategy_presets: %w", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (id) VALUES ('postgres_final_schema') ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("mark postgres schema migration: %w", err)
	}
	return nil
}
