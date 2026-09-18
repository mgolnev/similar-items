package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/discovery"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

// SimilarRepositoryInterface — данные для similar.
type SimilarRepositoryInterface interface {
	similar.ContextLoader
}

type SimilarRepository struct {
	db *database.DB
}

func NewSimilarRepository(db *database.DB) SimilarRepositoryInterface {
	return &SimilarRepository{db: db}
}

const similarProductSelect = `
	SELECT
		p.sku, p.sales_sku, p.group_id, p.name,
		COALESCE(p.category_path, '[]'), COALESCE(p.leaf_category, ''),
		COALESCE(p.normalized_color, ''), COALESCE(p.product_type, ''),
		COALESCE(p.gender, ''), COALESCE(p.price, 0), COALESCE(p.oldprice, 0), COALESCE(p.image_url, ''),
		COALESCE(p.root_section, ''), COALESCE(p.attributes, '{}'), COALESCE(p.categories, '')
`

func (r *SimilarRepository) LoadProductBySKU(ctx context.Context, sku string) (*similar.ProductRow, error) {
	// Подавляющее большинство запросов приходит с offer SKU. Ищем его отдельным
	// primary-key lookup; только затем проверяем group_id/sales_sku aliases.
	p, err := scanSimilarProductRow(r.db.QueryRowContext(
		ctx,
		similarProductSelect+` FROM products p WHERE p.sku = $1 LIMIT 1`,
		sku,
	))
	if err == sql.ErrNoRows {
		// Прежний OR вместе с ORDER BY sku LIMIT 1 заставлял PostgreSQL
		// сканировать products_pkey от начала таблицы. UNION сохраняет выбор
		// минимального SKU, но каждая ветка использует свой индекс.
		aliasQuery := `
		WITH candidate_skus AS (
			SELECT sku FROM products WHERE group_id = $1
			UNION
			SELECT sku FROM products WHERE sales_sku = $1
		),
		picked AS (
			SELECT sku FROM candidate_skus ORDER BY sku ASC LIMIT 1
		)
	` + similarProductSelect + `
		FROM products p
		JOIN picked ON picked.sku = p.sku
		`
		p, err = scanSimilarProductRow(r.db.QueryRowContext(ctx, aliasQuery, sku))
	}
	if err == sql.ErrNoRows {
		// Legacy IDs вида "<article> <color>" нужны только когда exact alias не
		// найден. Редкий fallback не ухудшает основной indexed path.
		legacyQuery := similarProductSelect + `
			FROM products p
			WHERE p.group_id LIKE $1 OR p.sales_sku LIKE $1
			ORDER BY p.sku ASC
			LIMIT 1
		`
		p, err = scanSimilarProductRow(r.db.QueryRowContext(ctx, legacyQuery, sku+" %"))
	}
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}
	if merged, mergeErr := r.loadMergedGroupAttributes(
		ctx,
		[]string{productCardKey(p.GroupID, p.SalesSKU, p.SKU)},
	); mergeErr == nil {
		applyGroupAttributeMerge(p, merged)
	}
	return p, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSimilarProductRow(row rowScanner) (*similar.ProductRow, error) {
	var p similar.ProductRow
	var price, oldPrice float64
	if err := row.Scan(
		&p.SKU, &p.SalesSKU, &p.GroupID, &p.Name,
		&p.CategoryPath, &p.LeafCategory, &p.NormalizedColor, &p.ProductType,
		&p.Gender, &price, &oldPrice, &p.ImageURL, &p.RootSection, &p.Attributes, &p.CategoriesFeed,
	); err != nil {
		return nil, err
	}
	p.ID = p.SKU
	p.Price = price
	p.OldPrice = oldPrice
	if strings.TrimSpace(p.RootSection) == "" || p.RootSection == "unknown" {
		p.RootSection = discovery.MapGenderFilterToSection(p.Gender)
	}
	return &p, nil
}
