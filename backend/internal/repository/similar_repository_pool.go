package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

func (r *SimilarRepository) LoadSimilarPool(ctx context.Context, section string, limit int) ([]similar.ProductRow, error) {
	if limit <= 0 {
		limit = 1000000
	}

	sectionFilter, args := similarPoolSectionSQL(section)
	query := `
		WITH pool AS (
			SELECT
				p.sku AS sku,
				p.sales_sku AS sales_sku,
				p.group_id AS group_id,
				p.name AS name,
				COALESCE(p.category_path, '[]') AS category_path,
				COALESCE(p.leaf_category, '') AS leaf_category,
				COALESCE(p.normalized_color, '') AS normalized_color,
				COALESCE(p.product_type, '') AS product_type,
				COALESCE(p.gender, '') AS gender,
				COALESCE(p.price, 0) AS price,
				COALESCE(p.oldprice, 0) AS oldprice,
				COALESCE(p.image_url, '') AS image_url,
				COALESCE(p.root_section, '') AS root_section,
				COALESCE(p.attributes, '{}') AS attributes,
				COALESCE(p.categories, '') AS categories,
				ROW_NUMBER() OVER (
					PARTITION BY COALESCE(NULLIF(TRIM(p.group_id), ''), TRIM(p.sales_sku), TRIM(p.sku))
					ORDER BY p.sku ASC
				) AS rn
			FROM products p
			WHERE p.publication_status = 'published'
			  AND p.available = TRUE
			  AND p.image_url IS NOT NULL AND TRIM(p.image_url) != ''
			  ` + sectionFilter + `
		)
		SELECT
			sku, sales_sku, group_id, name,
			category_path, leaf_category, normalized_color, product_type,
			gender, price, oldprice, image_url, root_section, attributes, categories
		FROM pool
		WHERE rn = 1
		ORDER BY sku ASC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `
	`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out, err := scanSimilarProductRows(rows)
	if err != nil {
		return nil, err
	}
	merged, err := r.loadMergedGroupAttributes(ctx, collectSimilarGroupKeys(out))
	if err != nil {
		return out, nil
	}
	for i := range out {
		applyGroupAttributeMerge(&out[i], merged)
	}
	return out, nil
}

func similarPoolSectionSQL(section string) (string, []interface{}) {
	section = strings.TrimSpace(strings.ToLower(section))
	if section == "" || section == "all" {
		return "", nil
	}
	if gender, fallbackRoot := discovery.MapSectionToMerchFilters(section); gender != "all" && fallbackRoot == "" {
		return ` AND (p.root_section = $1 OR (p.root_section = 'unknown' AND p.gender = $2))`, []interface{}{section, gender}
	}
	if section == "kids" {
		return ` AND (p.root_section = $1 OR (p.root_section = 'unknown' AND p.gender IN ('Девочки', 'Мальчики')))`, []interface{}{section}
	}
	return ` AND p.root_section = $1`, []interface{}{section}
}

func scanSimilarProductRows(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]similar.ProductRow, error) {
	var out []similar.ProductRow
	for rows.Next() {
		var p similar.ProductRow
		var price, oldPrice float64
		if err := rows.Scan(
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
		out = append(out, p)
	}
	return out, rows.Err()
}
