package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/discovery"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

// LoadProductIdentities разрешает входные article IDs одним indexed SQL.
// Cache-hit path нужны только поля ключа/секции; attributes/title/rules и
// group-merge загружаются позднее только на реальном cache miss.
func (r *SimilarRepository) LoadProductIdentities(
	ctx context.Context,
	ids []string,
) (map[string]similar.ProductRow, error) {
	out := make(map[string]similar.ProductRow, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	values := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		args = append(args, id)
		values = append(values, fmt.Sprintf("($%d::text)", len(args)))
	}
	if len(values) == 0 {
		return out, nil
	}
	if len(args) == 1 {
		id := args[0].(string)
		row, err := r.loadSingleProductIdentity(ctx, id)
		if err == sql.ErrNoRows {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out[id] = *row
		return out, nil
	}

	query := `
		WITH requested(request_id) AS (
			VALUES ` + strings.Join(values, ",") + `
		),
		matches AS (
			SELECT req.request_id, p.sku, p.sales_sku, p.group_id, p.root_section, p.gender
			FROM requested req
			JOIN products p ON p.sku = req.request_id
			UNION ALL
			SELECT req.request_id, p.sku, p.sales_sku, p.group_id, p.root_section, p.gender
			FROM requested req
			JOIN products p ON p.group_id = req.request_id
			UNION ALL
			SELECT req.request_id, p.sku, p.sales_sku, p.group_id, p.root_section, p.gender
			FROM requested req
			JOIN products p ON p.sales_sku = req.request_id
		)
		SELECT DISTINCT ON (request_id)
			request_id,
			sku,
			COALESCE(sales_sku, ''),
			COALESCE(group_id, ''),
			COALESCE(root_section, ''),
			COALESCE(gender, '')
		FROM matches
		ORDER BY request_id, sku ASC
	`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var requestID string
		var row similar.ProductRow
		if err := rows.Scan(
			&requestID,
			&row.SKU,
			&row.SalesSKU,
			&row.GroupID,
			&row.RootSection,
			&row.Gender,
		); err != nil {
			return nil, err
		}
		prepareProductIdentity(&row)
		out[requestID] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Legacy prefix resolution применяется только к действительно неразрешённым
	// ID. Для обычного batch всех SKU/group_id/sales_sku это остаётся одним SQL.
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, found := out[id]; found {
			continue
		}
		row, err := r.loadLegacyProductIdentity(ctx, id)
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

func (r *SimilarRepository) loadLegacyProductIdentity(
	ctx context.Context,
	id string,
) (*similar.ProductRow, error) {
	return scanProductIdentity(r.db.QueryRowContext(ctx, `
		SELECT
			sku,
			COALESCE(sales_sku, ''),
			COALESCE(group_id, ''),
			COALESCE(root_section, ''),
			COALESCE(gender, '')
		FROM products
		WHERE group_id LIKE $1 OR sales_sku LIKE $1
		ORDER BY sku ASC
		LIMIT 1
	`, id+" %"))
}

func (r *SimilarRepository) loadSingleProductIdentity(
	ctx context.Context,
	id string,
) (*similar.ProductRow, error) {
	product, err := scanProductIdentity(r.db.QueryRowContext(ctx, `
		SELECT
			sku,
			COALESCE(sales_sku, ''),
			COALESCE(group_id, ''),
			COALESCE(root_section, ''),
			COALESCE(gender, '')
		FROM products
		WHERE sku = $1
		LIMIT 1
	`, id))
	if err == nil {
		return product, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	product, err = scanProductIdentity(r.db.QueryRowContext(ctx, `
		WITH candidate_skus AS (
			SELECT sku FROM products WHERE group_id = $1
			UNION
			SELECT sku FROM products WHERE sales_sku = $1
		),
		picked AS (
			SELECT sku FROM candidate_skus ORDER BY sku ASC LIMIT 1
		)
		SELECT
			p.sku,
			COALESCE(p.sales_sku, ''),
			COALESCE(p.group_id, ''),
			COALESCE(p.root_section, ''),
			COALESCE(p.gender, '')
		FROM products p
		JOIN picked ON picked.sku = p.sku
	`, id))
	if err == nil {
		return product, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	return r.loadLegacyProductIdentity(ctx, id)
}

func scanProductIdentity(row rowScanner) (*similar.ProductRow, error) {
	var product similar.ProductRow
	if err := row.Scan(
		&product.SKU,
		&product.SalesSKU,
		&product.GroupID,
		&product.RootSection,
		&product.Gender,
	); err != nil {
		return nil, err
	}
	prepareProductIdentity(&product)
	return &product, nil
}

func prepareProductIdentity(row *similar.ProductRow) {
	row.ID = row.SKU
	if strings.TrimSpace(row.RootSection) == "" || row.RootSection == "unknown" {
		row.RootSection = discovery.MapGenderFilterToSection(row.Gender)
	}
}
