package services

import (
	"context"
	"fmt"
	"github.com/mgolnev/similar-items/backend/internal/catalog"
	"github.com/mgolnev/similar-items/backend/internal/config"
	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/repository"
	"github.com/mgolnev/similar-items/backend/internal/utils"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"mime/multipart"

	"log"
)

type DataUploadService struct {
	// feedMu не допускает наложения scheduler/manual/upload импортов. Каждый
	// успешный импорт создаёт ровно одно событие нового поколения кэша.
	feedMu           sync.Mutex
	productRepo      repository.ProductRepositoryInterface
	syncRepo         repository.SyncRepositoryInterface
	defaultFeedURL   string
	feedSyncInterval time.Duration
	changeHook       func()
}

func NewDataUploadService(
	productRepo repository.ProductRepositoryInterface,
	syncRepo repository.SyncRepositoryInterface,
	defaultFeedURL string,
	feedSyncInterval time.Duration,
) *DataUploadService {
	return &DataUploadService{
		productRepo:      productRepo,
		syncRepo:         syncRepo,
		defaultFeedURL:   strings.TrimSpace(defaultFeedURL),
		feedSyncInterval: feedSyncInterval,
	}
}

func (s *DataUploadService) WithChangeHook(fn func()) *DataUploadService {
	s.changeHook = fn
	return s
}

func (s *DataUploadService) notifyChanged() {
	if s.changeHook != nil {
		s.changeHook()
	}
}

func (s *DataUploadService) ProcessFeed(feedURL string) (int, error) {
	s.feedMu.Lock()
	defer s.feedMu.Unlock()

	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		if s.defaultFeedURL == "" {
			return 0, fmt.Errorf("feed URL is not configured")
		}
		feedURL = s.defaultFeedURL
	}

	feedPath, err := s.downloadFeed(feedURL)
	if err != nil {
		return 0, err
	}
	defer os.Remove(feedPath)

	return s.processFeedFromPath(feedPath)
}

// ProcessFeedUpload импортирует XML-фид из загруженного файла (мини-фид, тестовые выгрузки).
func (s *DataUploadService) ProcessFeedUpload(file *multipart.FileHeader) (int, error) {
	s.feedMu.Lock()
	defer s.feedMu.Unlock()

	if file == nil {
		return 0, fmt.Errorf("файл фида не передан")
	}
	name := strings.ToLower(strings.TrimSpace(file.Filename))
	if !strings.HasSuffix(name, ".xml") {
		return 0, fmt.Errorf("поддерживаются только XML-файлы (.xml)")
	}

	feedPath, err := saveMultipartFileToTemp(file, "feed-upload-*.xml")
	if err != nil {
		return 0, err
	}
	defer os.Remove(feedPath)

	return s.processFeedFromPath(feedPath)
}

func (s *DataUploadService) processFeedFromPath(feedPath string) (int, error) {
	productsCount, err := s.processXMLFeed(feedPath)
	if err != nil {
		return 0, err
	}
	if s.syncRepo != nil {
		if err := s.syncRepo.Upsert("feed", time.Now()); err != nil {
			fmt.Printf("Warning: failed to update feed sync status: %v\n", err)
		}
	}
	s.notifyChanged()
	return productsCount, nil
}

func saveMultipartFileToTemp(file *multipart.FileHeader, pattern string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("не удалось открыть файл фида: %w", err)
	}
	defer src.Close()

	tempFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, src); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("не удалось сохранить файл фида: %w", err)
	}
	return tempFile.Name(), nil
}

func (s *DataUploadService) RefreshFeed() (int, error) {
	return s.ProcessFeed(s.defaultFeedURL)
}

func (s *DataUploadService) DefaultFeedURL() string {
	return s.defaultFeedURL
}

func (s *DataUploadService) FeedSyncInterval() time.Duration {
	return s.feedSyncInterval
}

// ConfigureFeed обновляет URL и интервал автозагрузки в памяти сервиса.
func (s *DataUploadService) ConfigureFeed(defaultURL string, syncInterval time.Duration) {
	if u := config.NormalizeFeedURL(defaultURL); u != "" {
		s.defaultFeedURL = u
	}
	if syncInterval >= 0 {
		s.feedSyncInterval = syncInterval
	}
}

// ApplyPersistedFeedSettings загружает сохранённые настройки фида из runtime_settings.
func (s *DataUploadService) ApplyPersistedFeedSettings(strategy *StrategyService, envURL string, envInterval time.Duration) {
	url := strategy.GetDefaultFeedURL(envURL)
	fallbackMinutes := int(envInterval.Minutes())
	if envInterval <= 0 {
		fallbackMinutes = 0
	}
	minutes := strategy.GetFeedSyncIntervalMinutes(fallbackMinutes)
	s.ConfigureFeed(url, time.Duration(minutes)*time.Minute)
}

// SaveFeedSettings сохраняет URL и интервал в runtime_settings и применяет в сервисе.
func (s *DataUploadService) SaveFeedSettings(strategy *StrategyService, feedURL string, intervalMinutes int) error {
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return fmt.Errorf("feed url is required")
	}
	if intervalMinutes < 0 {
		intervalMinutes = 0
	}
	if err := strategy.SetDefaultFeedURL(feedURL); err != nil {
		return err
	}
	if err := strategy.SetFeedSyncIntervalMinutes(intervalMinutes); err != nil {
		return err
	}
	s.ConfigureFeed(feedURL, time.Duration(intervalMinutes)*time.Minute)
	return nil
}

func (s *DataUploadService) StartFeedScheduler(ctx context.Context) {
	if s.feedSyncInterval <= 0 || s.defaultFeedURL == "" || s.syncRepo == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(s.feedSyncInterval)
		defer ticker.Stop()

		// Откладываем первую синхронизацию, чтобы не блокировать UI при старте сервера.
		startupDelay := 3 * time.Minute
		startupTimer := time.NewTimer(startupDelay)
		defer startupTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Feed scheduler stopped")
				return
			case <-startupTimer.C:
				startupTimer.Stop()
				s.runScheduledFeedSync()
			case <-ticker.C:
				s.runScheduledFeedSync()
			}
		}
	}()
}

func (s *DataUploadService) runScheduledFeedSync() {
	if s.syncRepo == nil {
		return
	}

	lastSync, err := s.syncRepo.GetLastSync("feed")
	if err != nil {
		log.Printf("Feed scheduler: failed to get last sync: %v", err)
	}

	needsSync := lastSync == nil
	if lastSync != nil && s.feedSyncInterval > 0 {
		needsSync = time.Since(*lastSync) >= s.feedSyncInterval
	}

	if !needsSync {
		return
	}

	log.Println("Feed scheduler: starting automatic feed refresh...")
	if _, err := s.RefreshFeed(); err != nil {
		log.Printf("Feed scheduler: failed to refresh feed: %v", err)
	} else {
		log.Println("Feed scheduler: feed refreshed successfully")
	}
}

func (s *DataUploadService) downloadFeed(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("feed download failed with status: %d", resp.StatusCode)
	}

	// Создаем временный файл
	tempFile, err := os.CreateTemp("", "feed-*.xml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Копируем содержимое
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to copy feed content: %w", err)
	}

	return tempFile.Name(), nil
}

func (s *DataUploadService) processXMLFeed(feedPath string) (int, error) {
	// Парсим XML фид
	feed, err := utils.ParseFeedXML(feedPath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse XML feed: %w", err)
	}

	// Обрабатываем данные фида
	products, err := utils.ProcessFeedData(feed)
	if err != nil {
		return 0, fmt.Errorf("failed to process feed data: %w", err)
	}

	// Сразу импортируем в БД
	err = s.importProductsToDatabase(products)
	if err != nil {
		return 0, fmt.Errorf("failed to import products to database: %w", err)
	}

	return len(products), nil
}

// importProductsToDatabase импортирует товары через репозиторий
func (s *DataUploadService) importProductsToDatabase(products []utils.ProcessedProduct) error {
	fmt.Println("Импорт товаров в БД...")

	feedVersion := time.Now().UTC().Format(time.RFC3339)

	// Конвертируем в структуры БД
	var dbProducts []database.Product
	for _, product := range products {
		dbProduct := database.Product{
			SKU:                   product.SKU,
			SalesSKU:              product.SalesSKU,
			GroupID:               product.GroupID,
			Name:                  product.Name,
			Price:                 product.Price,
			OldPrice:              product.OldPrice,
			Discount:              product.Discount,
			Gender:                product.Gender,
			ImageURL:              product.ImageURL,
			SaleStartDate:         catalog.NormalizeSaleStartDate(product.SaleStartDate),
			Available:             product.Available,
			Categories:            product.Categories,
			URL:                   product.URL,
			SizeCompleteness:      product.SizeCompleteness,
			SizeOutOfStockPercent: product.SizeOutOfStockPercent,
			SizeName:              product.SizeName,
			StoreStock:            product.StoreStock,
			WarehouseStock:        product.WarehouseStock,
			CategoryPathJSON:      product.CategoryPathJSON,
			LeafCategory:          product.LeafCategory,
			RootSection:           product.RootSection,
			Subsection:            product.Subsection,
			AttributesJSON:        product.AttributesJSON,
			NormalizedColor:       product.NormalizedColor,
			ProductType:           product.ProductType,
			PublicationStatus:     product.PublicationStatus,
			FeedVersion:           feedVersion,
		}
		if pathJSON, leaf, root := catalog.DiscoveryFieldsFromFeed(product.Categories, nil, product.Gender); pathJSON != "" {
			dbProduct.CategoryPathJSON = pathJSON
			dbProduct.LeafCategory = leaf
			if root != "" && root != "unknown" {
				dbProduct.RootSection = root
			}
		}
		dbProducts = append(dbProducts, dbProduct)
	}

	// Импортируем через репозиторий
	err := s.productRepo.ImportProducts(dbProducts)
	if err != nil {
		return fmt.Errorf("failed to import products: %w", err)
	}

	if updated, err := s.productRepo.BackfillSaleStartDatesFromAttributes(); err != nil {
		return fmt.Errorf("failed to backfill sale start dates from feed attributes: %w", err)
	} else if updated > 0 {
		fmt.Printf("Из attributes фида восстановлено sale_start_date у %d товаров\n", updated)
	}

	fmt.Printf("Успешно импортировано %d товаров в БД\n", len(products))
	return nil
}

func (s *DataUploadService) GetDataStatus() (map[string]interface{}, error) {
	status := map[string]interface{}{
		"database_imported":           false,
		"db_products_count":           0,
		"feed_uploaded":               false,
		"feed_last_sync":              "",
		"feed_next_sync":              "",
		"feed_sync_interval_minutes":  int(s.feedSyncInterval.Minutes()),
		"default_feed_url":            s.defaultFeedURL,
		"feed_sync_scheduler_enabled": s.feedSyncInterval > 0 && s.defaultFeedURL != "",
	}

	// Проверяем базу данных через репозиторий
	count, err := s.productRepo.GetProductsCount()
	if err != nil {
		// Если таблица не существует или база пустая, это нормально
		status["database_imported"] = false
		status["db_products_count"] = 0
	} else if count > 0 {
		// Если в БД есть товары, значит фид обработан и импортирован
		status["database_imported"] = true
		status["db_products_count"] = count
	}

	if s.syncRepo != nil {
		if lastSync, err := s.syncRepo.GetLastSync("feed"); err == nil {
			if lastSync != nil {
				status["feed_uploaded"] = true
				status["feed_last_sync"] = lastSync.Format(time.RFC3339)
				if s.feedSyncInterval > 0 {
					next := lastSync.Add(s.feedSyncInterval)
					status["feed_next_sync"] = next.Format(time.RFC3339)
				}
			}
		} else {
			fmt.Printf("Warning: failed to fetch feed sync status: %v\n", err)
		}
	}

	return status, nil
}
