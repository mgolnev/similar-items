package utils

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mgolnev/similar-items/backend/internal/catalog"
)

type Feed struct {
	XMLName xml.Name `xml:"yml_catalog"`
	Shop    Shop     `xml:"shop"`
}

type Shop struct {
	XMLName    xml.Name   `xml:"shop"`
	Name       string     `xml:"name"`
	Categories []Category `xml:"categories>category"`
	Offers     []Offer    `xml:"offers>offer"`
}

type Category struct {
	ID       string `xml:"id,attr"`
	ParentID string `xml:"parentId,attr"`
	Name     string `xml:",chardata"`
}

type Offer struct {
	ID          string   `xml:"id,attr"`
	GroupID     string   `xml:"group_id,attr"`
	Name        string   `xml:"name"`
	Description string   `xml:"description"`
	Price       string   `xml:"price"`
	OldPrice    string   `xml:"oldprice"`
	Discount    string   `xml:"discount"`
	URL         string   `xml:"url"`
	Pictures    []string `xml:"picture"` // Массив всех тегов picture
	Available   bool     `xml:"available,attr"`
	CategoryIDs []string `xml:"categoryId"`
	Params      []Param  `xml:"param"`
}

type Param struct {
	Name  string `xml:"name,attr"`
	Unit  string `xml:"unit,attr"`
	Value string `xml:",chardata"`
}

type ProcessedProduct struct {
	SKU                   string  `json:"sku"`
	GroupID               string  `json:"group_id"`
	Color                 string  `json:"color"`
	SalesSKU              string  `json:"sales_sku"`
	Name                  string  `json:"name"`
	Price                 float64 `json:"price"`
	OldPrice              float64 `json:"oldprice"`
	Discount              float64 `json:"discount"`
	Gender                string  `json:"gender"`
	ImageURL              string  `json:"image_url"`
	Categories            string  `json:"categories"`
	CategoryIDs           string  `json:"category_ids"`
	URL                   string  `json:"url"`
	SizeCompleteness      string  `json:"size_completeness"`
	SizeOutOfStockPercent float64 `json:"size_out_of_stock_percent"`
	SizeName              string  `json:"size_name"`
	StoreStock            int     `json:"store_stock"`
	WarehouseStock        int     `json:"warehouse_stock"`
	OnlineSales           int     `json:"online_sales"`
	OnlinePurchases       int     `json:"online_purchases"`
	OfflineSales          int     `json:"offline_sales"`
	SaleStartDate         string  `json:"sale_start_date"`
	Available             bool    `json:"available"`
	// Discovery catalog (Phase 1)
	CategoryPathJSON  string `json:"category_path"`
	LeafCategory      string `json:"leaf_category"`
	RootSection       string `json:"root_section"`
	Subsection        string `json:"subsection"`
	AttributesJSON    string `json:"attributes"`
	NormalizedColor   string `json:"normalized_color"`
	ProductType       string `json:"product_type"`
	PublicationStatus string `json:"publication_status"`
	FeedVersion       string `json:"feed_version"`
}

func ParseFeedXML(filePath string) (*Feed, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var feed Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return &feed, nil
}

func ProcessFeedData(feed *Feed) ([]ProcessedProduct, error) {
	// Создаем карту категорий для быстрого поиска
	categories := make(map[string]Category)
	for _, cat := range feed.Shop.Categories {
		categories[cat.ID] = cat
	}

	var products []ProcessedProduct
	totalOffers := len(feed.Shop.Offers)
	processedOffers := 0
	filteredCount := 0
	invalidOffers := 0

	for _, offer := range feed.Shop.Offers {
		processedOffers++

		// Проверяем обязательные поля
		if strings.TrimSpace(offer.ID) == "" {
			fmt.Printf("Пропускаем товар с пустым ID: %+v\n", offer)
			invalidOffers++
			continue
		}

		// Фильтруем товары без изображения и берем первую картинку
		var pictureURL string
		if len(offer.Pictures) > 0 {
			pictureURL = strings.TrimSpace(offer.Pictures[0]) // Берем первую картинку
		}

		if pictureURL == "" {
			filteredCount++
			continue
		}

		// Получаем цвет из параметров
		color := ""
		gender := ""
		sizeCompleteness := ""
		sizeOutOfStockPercent := 0.0
		sizeName := ""
		storeStock := 0
		warehouseStock := 0
		saleStartDate := ""

		for _, param := range offer.Params {
			paramName := strings.TrimSpace(param.Name)
			lowerName := strings.ToLower(paramName)
			normalizedName := normalizeFeedParamName(paramName)
			paramValue := strings.TrimSpace(param.Value)

			switch {
			case lowerName == "цвет":
				color = paramValue
			case lowerName == "пол":
				gender = paramValue
			case normalizedName == "датаначалапродаж":
				saleStartDate = catalog.NormalizeSaleStartDate(paramValue)
			case lowerName == "полнотаразмеровмодели":
				sizeCompleteness = paramValue
			case normalizedName == "выбитостьразмеровпроцент":
				if paramValue != "" {
					value := strings.Replace(paramValue, "%", "", -1)
					if val, err := strconv.ParseFloat(value, 64); err == nil {
						sizeOutOfStockPercent = val
					}
				}
			case lowerName == "размер", lowerName == "size":
				if paramValue != "" {
					sizeName = paramValue
				}
			case strings.Contains(lowerName, "размер") &&
				!strings.Contains(lowerName, "полнота") &&
				!strings.Contains(lowerName, "выбитость") &&
				!strings.Contains(lowerName, "модел"):
				// «Размер товара на модели»/«Рост модели» — это фото-метаданные модели,
				// а не размер товара, поэтому их исключаем.
				if paramValue != "" && sizeName == "" {
					sizeName = paramValue
				}
			case lowerName == "storestock":
				if val, err := strconv.Atoi(paramValue); err == nil {
					storeStock = val
				}
			case lowerName == "warehousestock":
				if val, err := strconv.Atoi(paramValue); err == nil {
					warehouseStock = val
				}
			}
		}

		if strings.TrimSpace(sizeName) == "" {
			sizeName = "-"
		}

		// Создаем маппинг SKU для продаж
		salesSKU := offer.ID
		if offer.GroupID != "" && color != "" {
			salesSKU = fmt.Sprintf("%s %s", offer.GroupID, color)
		}

		// Параметры фида → map для attributes JSON
		paramMap := make(map[string]string, len(offer.Params))
		for _, param := range offer.Params {
			paramMap[strings.TrimSpace(param.Name)] = strings.TrimSpace(param.Value)
		}
		if desc := PlainTextFromFeedDescription(offer.Description); desc != "" {
			if existing := strings.TrimSpace(paramMap[DescriptionAttributeKey]); existing == "" || len(desc) > len(existing) {
				paramMap[DescriptionAttributeKey] = desc
			}
		}

		// Лучшая цепочка категорий (самая длинная) для discovery
		var bestCategoryChain []string
		var categoryPairs []string
		var allCategoryIDs []int

		for _, catID := range offer.CategoryIDs {
			chain := getCategoryChain(catID, categories)
			if len(chain) > len(bestCategoryChain) {
				bestCategoryChain = chain
			}
			if len(chain) > 0 {
				// Получаем все ID категорий в иерархии
				hierarchyIDs := getCategoryHierarchyIDs(catID, categories)
				allCategoryIDs = append(allCategoryIDs, hierarchyIDs...)

				// Создаем пары "ID:Название" для каждого уровня иерархии
				for i, categoryName := range chain {
					if i < len(hierarchyIDs) {
						categoryPairs = append(categoryPairs, fmt.Sprintf("%d:%s", hierarchyIDs[i], categoryName))
					}
				}
			}
		}

		// Убираем дубликаты из ID категорий
		uniqueCategoryIDs := make([]int, 0)
		seen := make(map[int]bool)
		for _, id := range allCategoryIDs {
			if !seen[id] {
				seen[id] = true
				uniqueCategoryIDs = append(uniqueCategoryIDs, id)
			}
		}

		// Парсим цены с обработкой пустых значений (как в Python версии)
		price := 0.0
		if strings.TrimSpace(offer.Price) != "" {
			if val, err := strconv.ParseFloat(offer.Price, 64); err == nil {
				price = val
			}
		}

		oldPrice := 0.0
		if strings.TrimSpace(offer.OldPrice) != "" {
			if val, err := strconv.ParseFloat(offer.OldPrice, 64); err == nil {
				oldPrice = val
			}
		}

		discount := 0.0
		if strings.TrimSpace(offer.Discount) != "" {
			if val, err := strconv.ParseFloat(offer.Discount, 64); err == nil {
				discount = val
			}
		}

		// Создаем строку с ID категорий
		categoryIDsStr := "["
		for i, id := range uniqueCategoryIDs {
			if i > 0 {
				categoryIDsStr += ", "
			}
			categoryIDsStr += strconv.Itoa(id)
		}
		categoryIDsStr += "]"

		categoriesStr := strings.Join(categoryPairs, " | ")
		discoveryPath := catalog.DiscoveryCategoryPath(categoriesStr, bestCategoryChain)
		catalogMeta := catalog.BuildProductMeta(
			discoveryPath,
			gender,
			color,
			pictureURL,
			paramMap,
			"",
			categoriesStr,
		)

		normalizedColor := catalogMeta.NormalizedColor
		if normalizedColor == "" && color != "" {
			normalizedColor = catalog.NormalizeColor(color)
		}
		if normalizedColor == "" {
			if suffix := catalog.ExtractColorFromGroupID(strings.TrimSpace(offer.GroupID)); suffix != "" {
				color = suffix
				normalizedColor = catalog.NormalizeColor(suffix)
			}
		}

		// Ключ карточки = цветомодель. В потоварном (cc) фиде group_id — это базовый
		// артикул без цвета («BAC013491»), а id — цветомодель («BAC013491-2»), поэтому
		// ключом берём id, иначе цвета одной модели схлопнулись бы в одну карточку.
		// В SKU-фиде group_id уже несёт цвет (с пробелом) — оставляем его как есть.
		cardGroupID := strings.TrimSpace(offer.GroupID)
		trimmedID := strings.TrimSpace(offer.ID)
		if cardGroupID != "" && !strings.Contains(cardGroupID, " ") &&
			strings.HasPrefix(trimmedID, cardGroupID+"-") {
			cardGroupID = trimmedID
		}

		product := ProcessedProduct{
			SKU:                   strings.TrimSpace(offer.ID),
			GroupID:               cardGroupID,
			Color:                 color,
			SalesSKU:              salesSKU,
			Name:                  strings.TrimSpace(offer.Name),
			Price:                 price,
			OldPrice:              oldPrice,
			Discount:              discount,
			Gender:                gender,
			ImageURL:              pictureURL, // Уже обработано выше
			Categories:            categoriesStr,
			CategoryIDs:           categoryIDsStr,
			URL:                   strings.TrimSpace(offer.URL),
			SizeCompleteness:      sizeCompleteness,
			SizeOutOfStockPercent: sizeOutOfStockPercent,
			SizeName:              sizeName,
			StoreStock:            storeStock,
			WarehouseStock:        warehouseStock,
			OnlineSales:           0,
			OnlinePurchases:       0,
			OfflineSales:          0,
			SaleStartDate:         saleStartDate,
			Available:             true,
			CategoryPathJSON:      catalogMeta.CategoryPathJSON,
			LeafCategory:          catalogMeta.LeafCategory,
			RootSection:           catalogMeta.RootSection,
			Subsection:            catalogMeta.Subsection,
			AttributesJSON:        catalogMeta.AttributesJSON,
			NormalizedColor:       normalizedColor,
			ProductType:           catalogMeta.ProductType,
			PublicationStatus:     catalogMeta.PublicationStatus,
		}

		products = append(products, product)
	}

	// Логируем статистику импорта
	fmt.Printf("Статистика импорта:\n")
	fmt.Printf("- Всего товаров в фиде: %d\n", totalOffers)
	fmt.Printf("- Обработано товаров: %d\n", processedOffers)
	fmt.Printf("- Отфильтровано (без изображения): %d\n", filteredCount)
	fmt.Printf("- Отфильтровано (некорректные данные): %d\n", invalidOffers)
	fmt.Printf("- Успешно импортировано: %d\n", len(products))
	fmt.Printf("- Итого отфильтровано: %d\n", filteredCount+invalidOffers)

	return products, nil
}

func normalizeFeedParamName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

func getCategoryChain(catID string, categories map[string]Category) []string {
	var chain []string
	current := categories[catID]

	for current.ID != "" {
		chain = append([]string{current.Name}, chain...)
		if current.ParentID == "" {
			break
		}
		current = categories[current.ParentID]
	}

	return chain
}

func getCategoryHierarchyIDs(catID string, categories map[string]Category) []int {
	var ids []int
	current := categories[catID]

	for current.ID != "" {
		if id, err := strconv.Atoi(current.ID); err == nil {
			ids = append([]int{id}, ids...)
		}
		if current.ParentID == "" {
			break
		}
		current = categories[current.ParentID]
	}

	return ids
}
