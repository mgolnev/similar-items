package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
)

func mergeAttributesJSON(primary string, extras ...string) string {
	merged := make(map[string]string)
	apply := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" || raw == "{}" {
			return
		}
		var attrs map[string]string
		if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
			return
		}
		for k, v := range attrs {
			key := strings.TrimSpace(k)
			val := strings.TrimSpace(v)
			if key == "" || val == "" {
				continue
			}
			if existing, ok := merged[key]; !ok || existing == "" {
				merged[key] = val
			}
		}
	}
	apply(primary)
	for _, raw := range extras {
		apply(raw)
	}
	if len(merged) == 0 {
		return primary
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return primary
	}
	return string(out)
}

type groupAttributeMerge struct {
	Attributes  string
	ProductType string
}

func productCardKey(groupID, salesSKU, sku string) string {
	if key := strings.TrimSpace(groupID); key != "" {
		return key
	}
	if key := strings.TrimSpace(salesSKU); key != "" {
		return key
	}
	return strings.TrimSpace(sku)
}

func (r *SimilarRepository) loadMergedGroupAttributes(ctx context.Context, keys []string) (map[string]groupAttributeMerge, error) {
	out := make(map[string]groupAttributeMerge, len(keys))
	if len(keys) == 0 {
		return out, nil
	}

	unique := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}
	if len(unique) == 0 {
		return out, nil
	}

	parts := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, key := range unique {
		parts[i] = fmt.Sprintf("$%d", i+1)
		args[i] = key
	}
	placeholders := strings.Join(parts, ",")

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(NULLIF(TRIM(group_id), ''), NULLIF(TRIM(sales_sku), ''), TRIM(sku)) AS card_key,
			COALESCE(attributes, '{}'),
			COALESCE(product_type, '')
		FROM products
		WHERE COALESCE(NULLIF(TRIM(group_id), ''), NULLIF(TRIM(sales_sku), ''), TRIM(sku)) IN (`+placeholders+`)
		  AND attributes IS NOT NULL AND TRIM(attributes) != '' AND TRIM(attributes) != '{}'
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	typeCounts := make(map[string]map[string]int)
	attrParts := make(map[string][]string)
	for rows.Next() {
		var cardKey, attrs, productType string
		if err := rows.Scan(&cardKey, &attrs, &productType); err != nil {
			return nil, err
		}
		cardKey = strings.TrimSpace(cardKey)
		if cardKey == "" {
			continue
		}
		attrParts[cardKey] = append(attrParts[cardKey], attrs)
		pt := strings.TrimSpace(productType)
		if pt != "" {
			if typeCounts[cardKey] == nil {
				typeCounts[cardKey] = make(map[string]int)
			}
			typeCounts[cardKey][pt]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for cardKey, parts := range attrParts {
		merged := parts[0]
		for _, part := range parts[1:] {
			merged = mergeAttributesJSON(merged, part)
		}
		bestType := ""
		bestCount := 0
		for pt, count := range typeCounts[cardKey] {
			if count > bestCount {
				bestCount = count
				bestType = pt
			}
		}
		out[cardKey] = groupAttributeMerge{
			Attributes:  catalog.RebuildAttributeSlotSetsInAttributesJSON(merged),
			ProductType: bestType,
		}
	}
	return out, nil
}

func applyGroupAttributeMerge(row *similar.ProductRow, merged map[string]groupAttributeMerge) {
	if row == nil {
		return
	}
	key := productCardKey(row.GroupID, row.SalesSKU, row.SKU)
	entry, ok := merged[key]
	if !ok {
		return
	}
	if entry.Attributes != "" && entry.Attributes != "{}" {
		row.Attributes = entry.Attributes
	}
	if entry.ProductType != "" {
		row.ProductType = entry.ProductType
	}
}

func collectSimilarGroupKeys(rows []similar.ProductRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		key := productCardKey(row.GroupID, row.SalesSKU, row.SKU)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}
