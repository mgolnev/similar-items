package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/queries"
)

type ProductRepository struct {
	db *database.DB
}

// SampledProduct — краткая карточка для визуального preview.
type SampledProduct struct {
	SKU      string
	Name     string
	ImageURL string
}

// CopyReferenceProduct — минимальная карточка для сервиса генерации текстов.
// Описание уже приходит в товарном фиде и хранится внутри attributes JSON.
type CopyReferenceProduct struct {
	Article       string `json:"article"`
	NameRU        string `json:"name_ru"`
	DescriptionRU string `json:"description_ru"`
}

// ProductRepositoryInterface — товары для импорта фида и мета-каталога атрибутов.
type ProductRepositoryInterface interface {
	ImportProducts(products []database.Product) error
	BackfillSaleStartDatesFromAttributes() (int64, error)
	GetProductsCount() (int, error)
	CollectAttributeKeyCounts(sampleLimit int) (map[string]int, error)
	SampleRandomProducts(section string, limit int) ([]SampledProduct, error)
	LoadVisualProducts(articles []string) (map[string]SampledProduct, error)
	CopyReferenceProducts(skus []string) ([]CopyReferenceProduct, error)
}

func NewProductRepository(db *database.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// CopyReferenceProducts возвращает тексты карточек в порядке входных SKU.
// Метод намеренно не является частью публичного catalog API: он используется
// только read-only ручкой для внутреннего сервиса копирайтинга.
func (r *ProductRepository) CopyReferenceProducts(skus []string) ([]CopyReferenceProduct, error) {
	ordered := make([]string, 0, len(skus))
	seen := make(map[string]struct{}, len(skus))
	for _, sku := range skus {
		sku = strings.TrimSpace(sku)
		if sku == "" {
			continue
		}
		if _, exists := seen[sku]; exists {
			continue
		}
		seen[sku] = struct{}{}
		ordered = append(ordered, sku)
	}
	if len(ordered) == 0 {
		return []CopyReferenceProduct{}, nil
	}

	placeholders := make([]string, len(ordered))
	args := make([]interface{}, len(ordered))
	for index, sku := range ordered {
		placeholders[index] = fmt.Sprintf("$%d", index+1)
		args[index] = sku
	}
	rows, err := r.db.Query(
		`SELECT sku, COALESCE(name, ''), COALESCE(attributes, '{}')
		 FROM products
		 WHERE sku IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("query copy reference products: %w", err)
	}
	defer rows.Close()

	bySKU := make(map[string]CopyReferenceProduct, len(ordered))
	for rows.Next() {
		var sku, name, attributes string
		if err := rows.Scan(&sku, &name, &attributes); err != nil {
			return nil, fmt.Errorf("scan copy reference product: %w", err)
		}
		bySKU[sku] = CopyReferenceProduct{
			Article:       sku,
			NameRU:        strings.TrimSpace(name),
			DescriptionRU: copyDescriptionFromAttributes(attributes),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate copy reference products: %w", err)
	}

	result := make([]CopyReferenceProduct, 0, len(ordered))
	for _, sku := range ordered {
		if product, ok := bySKU[sku]; ok {
			result = append(result, product)
		}
	}
	return result, nil
}

func copyDescriptionFromAttributes(attributes string) string {
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(attributes), &values); err != nil {
		return ""
	}
	for key, value := range values {
		if normalizeAttributeKey(key) == "описание" {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

// ImportProducts импортирует товары из фида (upsert + удаление пропавших SKU).
func (r *ProductRepository) ImportProducts(products []database.Product) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(queries.UpsertProduct)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	currentSKUs := make([]string, 0, len(products))
	for _, product := range products {
		currentSKUs = append(currentSKUs, strings.TrimSpace(product.SKU))
		_, err := stmt.Exec(
			product.SKU, product.SalesSKU, product.GroupID, product.Name, product.Price, product.OldPrice,
			product.Discount, product.Gender, product.ImageURL, product.SaleStartDate,
			product.Available, product.Categories, product.URL, product.SizeName,
			product.SizeCompleteness, product.SizeOutOfStockPercent, product.StoreStock, product.WarehouseStock,
			product.CategoryPathJSON, product.LeafCategory, product.RootSection, product.Subsection,
			product.AttributesJSON, product.NormalizedColor, product.ProductType,
			product.PublicationStatus, product.FeedVersion,
		)
		if err != nil {
			return fmt.Errorf("failed to insert product %s: %w", product.SKU, err)
		}
	}

	if err := deleteProductsMissingFromFeed(tx, currentSKUs); err != nil {
		return err
	}

	return tx.Commit()
}

func deleteProductsMissingFromFeed(tx *database.Tx, currentSKUs []string) error {
	if len(currentSKUs) == 0 {
		return nil
	}

	if _, err := tx.Exec("CREATE TEMP TABLE IF NOT EXISTS current_feed_skus (sku TEXT PRIMARY KEY)"); err != nil {
		return fmt.Errorf("failed to create feed sku temp table: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM current_feed_skus"); err != nil {
		return fmt.Errorf("failed to clear feed sku temp table: %w", err)
	}

	stmt, err := tx.Prepare("INSERT INTO current_feed_skus (sku) VALUES ($1) ON CONFLICT (sku) DO NOTHING")
	if err != nil {
		return fmt.Errorf("failed to prepare feed sku insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, sku := range currentSKUs {
		if strings.TrimSpace(sku) == "" {
			continue
		}
		if _, err := stmt.Exec(strings.TrimSpace(sku)); err != nil {
			return fmt.Errorf("failed to insert feed sku %s: %w", sku, err)
		}
		inserted++
	}
	if inserted == 0 {
		return nil
	}

	if _, err := tx.Exec(`
		DELETE FROM products
		WHERE NOT EXISTS (
			SELECT 1 FROM current_feed_skus cfs WHERE cfs.sku = products.sku
		)
	`); err != nil {
		return fmt.Errorf("failed to delete stale products: %w", err)
	}
	return nil
}

// BackfillSaleStartDatesFromAttributes восстанавливает sale_start_date из attributes фида.
func (r *ProductRepository) BackfillSaleStartDatesFromAttributes() (int64, error) {
	rows, err := r.db.Query(`
		SELECT sku, attributes
		FROM products
		WHERE NULLIF(NULLIF(TRIM(sale_start_date), ''), '01.01.2000') IS NULL
			AND COALESCE(TRIM(attributes), '') <> ''
	`)
	if err != nil {
		return 0, fmt.Errorf("query products for sale start date backfill: %w", err)
	}
	defer rows.Close()

	type update struct {
		sku  string
		date string
	}
	var updates []update
	for rows.Next() {
		var sku, attrs string
		if err := rows.Scan(&sku, &attrs); err != nil {
			return 0, fmt.Errorf("scan product for sale start date backfill: %w", err)
		}
		if date := saleStartDateFromAttributesJSON(attrs); date != "" {
			updates = append(updates, update{sku: sku, date: date})
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate products for sale start date backfill: %w", err)
	}
	if len(updates) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin sale start date backfill: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		UPDATE products
		SET sale_start_date = $1
		WHERE sku = $2
			AND NULLIF(NULLIF(TRIM(sale_start_date), ''), '01.01.2000') IS NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("prepare sale start date backfill: %w", err)
	}
	defer stmt.Close()

	var affected int64
	for _, item := range updates {
		res, err := stmt.Exec(item.date, item.sku)
		if err != nil {
			return 0, fmt.Errorf("backfill sale start date for %s: %w", item.sku, err)
		}
		n, _ := res.RowsAffected()
		affected += n
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit sale start date backfill: %w", err)
	}
	return affected, nil
}

func saleStartDateFromAttributesJSON(attrs string) string {
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(attrs), &values); err != nil {
		return ""
	}
	for key, value := range values {
		if normalizeAttributeKey(key) != "датаначалапродаж" {
			continue
		}
		return catalog.NormalizeSaleStartDate(fmt.Sprint(value))
	}
	return ""
}

func normalizeAttributeKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	return key
}

// GetProductsCount получает количество товаров в БД.
func (r *ProductRepository) GetProductsCount() (int, error) {
	var count int
	err := r.db.QueryRow(queries.GetProductsCount).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get products count: %w", err)
	}
	return count, nil
}

// SampleRandomProducts — случайные товары с фото (для вкладки «Визуал»).
func (r *ProductRepository) SampleRandomProducts(section string, limit int) ([]SampledProduct, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 30 {
		limit = 30
	}
	section = strings.TrimSpace(strings.ToLower(section))
	q := `
		SELECT sku, COALESCE(name, ''), COALESCE(image_url, '')
		FROM products
		WHERE COALESCE(image_url, '') != ''
	`
	args := make([]interface{}, 0, 2)
	if section != "" && section != "all" {
		q += ` AND COALESCE(root_section, '') = $1`
		args = append(args, section)
	}
	if len(args) == 0 {
		q += ` ORDER BY RANDOM() LIMIT $1`
		args = append(args, limit)
	} else {
		q += ` ORDER BY RANDOM() LIMIT $2`
		args = append(args, limit)
	}

	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("sample random products: %w", err)
	}
	defer rows.Close()

	out := make([]SampledProduct, 0, limit)
	for rows.Next() {
		var p SampledProduct
		if err := rows.Scan(&p.SKU, &p.Name, &p.ImageURL); err != nil {
			continue
		}
		if strings.TrimSpace(p.SKU) == "" {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadVisualProducts одним запросом обогащает article/group IDs из ranked
// cache данными карточек. Ключ результата — исходный article из запроса, а SKU
// внутри карточки — конкретный offer, выбранный теми же приоритетами
// sku → group_id → sales_sku, что и публичное разрешение article IDs.
func (r *ProductRepository) LoadVisualProducts(articles []string) (map[string]SampledProduct, error) {
	out := make(map[string]SampledProduct, len(articles))
	values := make([]string, 0, len(articles))
	args := make([]interface{}, 0, len(articles))
	seen := make(map[string]struct{}, len(articles))
	for _, raw := range articles {
		article := strings.TrimSpace(raw)
		if article == "" {
			continue
		}
		if _, ok := seen[article]; ok {
			continue
		}
		seen[article] = struct{}{}
		args = append(args, article)
		values = append(values, fmt.Sprintf("($%d::text)", len(args)))
	}
	if len(values) == 0 {
		return out, nil
	}

	query := `
		WITH requested(article) AS (
			VALUES ` + strings.Join(values, ",") + `
		),
		matches AS (
			SELECT req.article, p.sku, p.name, p.image_url, 1 AS priority
			FROM requested req
			JOIN products p ON p.sku = req.article
			WHERE p.publication_status = 'published'
			UNION ALL
			SELECT req.article, p.sku, p.name, p.image_url, 2 AS priority
			FROM requested req
			JOIN products p ON p.group_id = req.article
			WHERE p.publication_status = 'published'
			UNION ALL
			SELECT req.article, p.sku, p.name, p.image_url, 3 AS priority
			FROM requested req
			JOIN products p ON p.sales_sku = req.article
			WHERE p.publication_status = 'published'
		)
		SELECT DISTINCT ON (article)
			article,
			sku,
			COALESCE(name, ''),
			COALESCE(image_url, '')
		FROM matches
		ORDER BY article, priority, sku
	`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("load visual products: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var article string
		var product SampledProduct
		if err := rows.Scan(&article, &product.SKU, &product.Name, &product.ImageURL); err != nil {
			return nil, fmt.Errorf("scan visual product: %w", err)
		}
		out[article] = product
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visual products: %w", err)
	}
	return out, nil
}

// CollectAttributeKeyCounts собирает частоту ключей attributes (для каталога признаков в админке).
// sampleLimit <= 0 — полный проход по каталогу без LIMIT.
func (r *ProductRepository) CollectAttributeKeyCounts(sampleLimit int) (map[string]int, error) {
	baseQuery := `
		SELECT attributes FROM products
		WHERE attributes IS NOT NULL AND TRIM(attributes) != '' AND TRIM(attributes) != '{}'
	`
	var rows *sql.Rows
	var err error
	if sampleLimit > 0 {
		rows, err = r.db.Query(baseQuery+" LIMIT $1", sampleLimit)
	} else {
		rows, err = r.db.Query(baseQuery)
	}
	if err != nil {
		return nil, fmt.Errorf("scan attribute keys: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var attrs map[string]string
		if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
			continue
		}
		seen := make(map[string]struct{})
		for k := range attrs {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if _, dup := seen[k]; dup {
				continue
			}
			seen[k] = struct{}{}
			counts[k]++
		}
	}
	return counts, rows.Err()
}
