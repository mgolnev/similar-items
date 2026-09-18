package handlers

import (
	"fmt"
	"github.com/mgolnev/similar-items/backend/internal/services"
	"time"

	"github.com/gofiber/fiber/v2"
)

type DataUploadHandler struct {
	dataUploadSvc *services.DataUploadService
	strategySvc   *services.StrategyService
}

func NewDataUploadHandler(dataUploadSvc *services.DataUploadService, strategySvc *services.StrategyService) *DataUploadHandler {
	return &DataUploadHandler{
		dataUploadSvc: dataUploadSvc,
		strategySvc:   strategySvc,
	}
}

func (h *DataUploadHandler) ProcessFeed(c *fiber.Ctx) error {
	feedURL := c.Query("url", h.dataUploadSvc.DefaultFeedURL())

	// Скачиваем и обрабатываем фид
	productsCount, err := h.dataUploadSvc.ProcessFeed(feedURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := fiber.Map{
		"status":         "success",
		"message":        fmt.Sprintf("Фид успешно обработан. Создано %d товаров.", productsCount),
		"products_count": productsCount,
	}
	return c.JSON(response)
}

func (h *DataUploadHandler) UploadFeed(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Файл фида не найден (поле file)",
		})
	}

	productsCount, err := h.dataUploadSvc.ProcessFeedUpload(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := fiber.Map{
		"status":         "success",
		"message":        fmt.Sprintf("Фид из файла обработан. Импортировано %d товаров.", productsCount),
		"products_count": productsCount,
	}
	return c.JSON(response)
}

func (h *DataUploadHandler) DownloadFeed(c *fiber.Ctx) error {
	feedURL := c.Query("url", h.dataUploadSvc.DefaultFeedURL())

	// Скачиваем и обрабатываем фид
	productsCount, err := h.dataUploadSvc.ProcessFeed(feedURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	response := fiber.Map{
		"status":         "success",
		"message":        fmt.Sprintf("Фид успешно обработан. Создано %d товаров.", productsCount),
		"products_count": productsCount,
	}
	return c.JSON(response)
}

func (h *DataUploadHandler) GetDataStatus(c *fiber.Ctx) error {
	status, err := h.dataUploadSvc.GetDataStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}

func (h *DataUploadHandler) RefreshFeed(c *fiber.Ctx) error {
	count, err := h.dataUploadSvc.RefreshFeed()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("failed to refresh feed: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":         "success",
		"message":        fmt.Sprintf("Фид обновлен. Обработано %d товаров.", count),
		"products_count": count,
		"last_synced_at": time.Now().Format(time.RFC3339),
	})
}

// GetFeedSettings — GET /api/admin/feed-settings
func (h *DataUploadHandler) GetFeedSettings(c *fiber.Ctx) error {
	intervalMinutes := int(h.dataUploadSvc.FeedSyncInterval().Minutes())
	persistedInterval, _ := h.strategySvc.IsFeedSyncIntervalPersisted()
	return c.JSON(fiber.Map{
		"feed_url":                h.dataUploadSvc.DefaultFeedURL(),
		"sync_interval_minutes":   intervalMinutes,
		"scheduler_enabled":       intervalMinutes > 0 && h.dataUploadSvc.DefaultFeedURL() != "",
		"sync_interval_persisted": persistedInterval,
	})
}

// SaveFeedSettings — PUT /api/admin/feed-settings
func (h *DataUploadHandler) SaveFeedSettings(c *fiber.Ctx) error {
	var body struct {
		FeedURL             string `json:"feed_url"`
		SyncIntervalMinutes int    `json:"sync_interval_minutes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_body", "message": "invalid JSON"},
		})
	}
	if err := h.dataUploadSvc.SaveFeedSettings(h.strategySvc, body.FeedURL, body.SyncIntervalMinutes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{"code": "save_failed", "message": err.Error()},
		})
	}
	return c.JSON(fiber.Map{
		"ok":                    true,
		"feed_url":              h.dataUploadSvc.DefaultFeedURL(),
		"sync_interval_minutes": int(h.dataUploadSvc.FeedSyncInterval().Minutes()),
		"scheduler_enabled":     h.dataUploadSvc.FeedSyncInterval() > 0,
	})
}
