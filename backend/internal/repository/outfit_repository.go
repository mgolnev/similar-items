package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
)

type OutfitRepositoryInterface interface {
	outfits.ContextLoader
}

type OutfitRepository struct {
	db *database.DB
}

func NewOutfitRepository(db *database.DB) OutfitRepositoryInterface {
	return &OutfitRepository{db: db}
}

// LoadOutfitPool загружает доступные варианты без коллапса: service объединяет
// их в карточку и сохраняет все встреченные ссылки «Сочетается с товаром».
func (r *OutfitRepository) LoadOutfitPool(ctx context.Context) ([]outfits.ProductRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			sku,
			COALESCE(sales_sku, ''),
			COALESCE(group_id, ''),
			COALESCE(name, ''),
			COALESCE(image_url, ''),
			COALESCE(attributes, '{}')
		FROM products
		WHERE publication_status = 'published'
		  AND available = TRUE
		  AND image_url IS NOT NULL AND TRIM(image_url) != ''
		ORDER BY sku ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]outfits.ProductRow, 0)
	for rows.Next() {
		var row outfits.ProductRow
		if err := rows.Scan(&row.SKU, &row.SalesSKU, &row.GroupID, &row.Name, &row.ImageURL, &row.Attributes); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *OutfitRepository) LoadOutfitIdentities(ctx context.Context, ids []string) (map[string]outfits.ProductRow, error) {
	out := make(map[string]outfits.ProductRow, len(ids))
	values := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		args = append(args, id)
		values = append(values, fmt.Sprintf("($%d::text)", len(args)))
	}
	if len(values) == 0 {
		return out, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		WITH requested(request_id) AS (
			VALUES `+strings.Join(values, ",")+`
		), matches AS (
			SELECT req.request_id, p.sku, p.sales_sku, p.group_id, p.name, p.image_url, p.attributes
			FROM requested req JOIN products p ON p.sku = req.request_id
			UNION ALL
			SELECT req.request_id, p.sku, p.sales_sku, p.group_id, p.name, p.image_url, p.attributes
			FROM requested req JOIN products p ON p.group_id = req.request_id
			UNION ALL
			SELECT req.request_id, p.sku, p.sales_sku, p.group_id, p.name, p.image_url, p.attributes
			FROM requested req JOIN products p ON p.sales_sku = req.request_id
		)
		SELECT DISTINCT ON (request_id)
			request_id,
			sku,
			COALESCE(sales_sku, ''),
			COALESCE(group_id, ''),
			COALESCE(name, ''),
			COALESCE(image_url, ''),
			COALESCE(attributes, '{}')
		FROM matches
		ORDER BY request_id, sku ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var requestID string
		var row outfits.ProductRow
		if err := rows.Scan(&requestID, &row.SKU, &row.SalesSKU, &row.GroupID, &row.Name, &row.ImageURL, &row.Attributes); err != nil {
			return nil, err
		}
		out[requestID] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := out[id]; ok {
			continue
		}
		row, err := scanOutfitProduct(r.db.QueryRowContext(ctx, `
			SELECT sku, COALESCE(sales_sku, ''), COALESCE(group_id, ''),
				COALESCE(name, ''), COALESCE(image_url, ''), COALESCE(attributes, '{}')
			FROM products
			WHERE group_id LIKE $1 OR sales_sku LIKE $1
			ORDER BY sku ASC
			LIMIT 1
		`, id+" %"))
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[id] = *row
	}
	return out, nil
}

func scanOutfitProduct(row rowScanner) (*outfits.ProductRow, error) {
	var product outfits.ProductRow
	if err := row.Scan(
		&product.SKU,
		&product.SalesSKU,
		&product.GroupID,
		&product.Name,
		&product.ImageURL,
		&product.Attributes,
	); err != nil {
		return nil, err
	}
	return &product, nil
}
