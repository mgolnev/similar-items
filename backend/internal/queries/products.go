package queries

// SQL запросы для работы с товарами (расширенные)
const (
	// Получение всех товаров (базовый)
	GetAllProductsBasic = `
		SELECT sku, sales_sku, group_id, name, price, oldprice, discount, gender,
			   image_url, sale_start_date, available, categories, url,
			   size_name, size_completeness, size_out_of_stock_percent,
			   store_stock, warehouse_stock
		FROM products
	`

	// Подсчет товаров
	GetProductsCount = `SELECT COUNT(*) FROM products`

	// UpsertProduct обновляет товар из фида без удаления строки products.
	UpsertProduct = `
		INSERT INTO products (
			sku, sales_sku, group_id, name, price, oldprice, discount, gender, image_url,
			sale_start_date, available, categories, url, size_name,
			size_completeness, size_out_of_stock_percent, store_stock, warehouse_stock,
			category_path, leaf_category, root_section, subsection, attributes,
			normalized_color, product_type, publication_status, feed_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24, $25, $26, $27)
		ON CONFLICT (sku) DO UPDATE SET
			sales_sku = excluded.sales_sku,
			group_id = excluded.group_id,
			name = excluded.name,
			price = excluded.price,
			oldprice = excluded.oldprice,
			discount = excluded.discount,
			gender = excluded.gender,
			image_url = excluded.image_url,
			sale_start_date = excluded.sale_start_date,
			available = excluded.available,
			categories = excluded.categories,
			url = excluded.url,
			size_name = excluded.size_name,
			size_completeness = excluded.size_completeness,
			size_out_of_stock_percent = excluded.size_out_of_stock_percent,
			store_stock = excluded.store_stock,
			warehouse_stock = excluded.warehouse_stock,
			category_path = excluded.category_path,
			leaf_category = excluded.leaf_category,
			root_section = excluded.root_section,
			subsection = excluded.subsection,
			attributes = excluded.attributes,
			normalized_color = excluded.normalized_color,
			product_type = excluded.product_type,
			publication_status = excluded.publication_status,
			feed_version = excluded.feed_version
		WHERE
			products.sales_sku IS DISTINCT FROM excluded.sales_sku OR
			products.group_id IS DISTINCT FROM excluded.group_id OR
			products.name IS DISTINCT FROM excluded.name OR
			products.price IS DISTINCT FROM excluded.price OR
			products.oldprice IS DISTINCT FROM excluded.oldprice OR
			products.discount IS DISTINCT FROM excluded.discount OR
			products.gender IS DISTINCT FROM excluded.gender OR
			products.image_url IS DISTINCT FROM excluded.image_url OR
			products.sale_start_date IS DISTINCT FROM excluded.sale_start_date OR
			products.available IS DISTINCT FROM excluded.available OR
			products.categories IS DISTINCT FROM excluded.categories OR
			products.url IS DISTINCT FROM excluded.url OR
			products.size_name IS DISTINCT FROM excluded.size_name OR
			products.size_completeness IS DISTINCT FROM excluded.size_completeness OR
			products.size_out_of_stock_percent IS DISTINCT FROM excluded.size_out_of_stock_percent OR
			products.store_stock IS DISTINCT FROM excluded.store_stock OR
			products.warehouse_stock IS DISTINCT FROM excluded.warehouse_stock OR
			products.category_path IS DISTINCT FROM excluded.category_path OR
			products.leaf_category IS DISTINCT FROM excluded.leaf_category OR
			products.root_section IS DISTINCT FROM excluded.root_section OR
			products.subsection IS DISTINCT FROM excluded.subsection OR
			products.attributes IS DISTINCT FROM excluded.attributes OR
			products.normalized_color IS DISTINCT FROM excluded.normalized_color OR
			products.product_type IS DISTINCT FROM excluded.product_type OR
			products.publication_status IS DISTINCT FROM excluded.publication_status OR
			products.feed_version IS DISTINCT FROM excluded.feed_version
	`
)
