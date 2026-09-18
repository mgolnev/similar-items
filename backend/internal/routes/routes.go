package routes

import (
	"github.com/mgolnev/similar-items/backend/internal/container"

	fiber "github.com/gofiber/fiber/v2"
)

// SetupRoutes настраивает все маршруты сервиса «Похожие».
func SetupRoutes(app *fiber.App, container *container.Container) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/ready", func(c *fiber.Ctx) error {
		similarStatus := container.Services.Similar.GetWarmStatus()
		outfitStatus := container.Services.Outfits.GetWarmStatus()
		ready := similarStatus.Ready && outfitStatus.Ready
		response := fiber.Map{"ready": ready, "similar": similarStatus, "outfits": outfitStatus}
		if !ready {
			return c.Status(fiber.StatusServiceUnavailable).JSON(response)
		}
		return c.JSON(response)
	})

	setupFeedRoutes(app, container)
	setupAdminRoutes(app, container)
	setupClientAPIRoutes(app, container)
}

// setupClientAPIRoutes — публичный API похожих товаров.
func setupClientAPIRoutes(app *fiber.App, container *container.Container) {
	v1 := app.Group("/api/v1/client")
	v1.Post("/products/similar", container.Handlers.SimilarRollout.ByArticles)
	v1.Get("/products/similar", container.Handlers.SimilarRollout.ByArticles)
	v1.Get("/products/:id/similar", container.Handlers.SimilarRollout.ByArticle)
	v1.Post("/products/recommendations", container.Handlers.Recommendations.ByArticles)
	v1.Get("/products/recommendations", container.Handlers.Recommendations.ByArticles)
	v1.Get("/products/:id/recommendations", container.Handlers.Recommendations.ByArticle)
	v1.Post("/products/outfits", container.Handlers.Outfits.ByArticles)
	v1.Get("/products/outfits", container.Handlers.Outfits.ByArticles)
	v1.Get("/products/:id/outfits", container.Handlers.Outfits.ByArticle)
}

// setupAdminRoutes — админка стратегии похожих.
func setupAdminRoutes(app *fiber.App, container *container.Container) {
	admin := app.Group("/api/admin")
	admin.Get("/similar-strategy", container.Handlers.AdminStrategy.GetStrategy)
	admin.Put("/similar-strategy", container.Handlers.AdminStrategy.SaveStrategy)
	admin.Post("/similar-strategy/reset", container.Handlers.AdminStrategy.ResetStrategy)
	admin.Post("/similar-strategy/preview", container.Handlers.AdminStrategy.PreviewSimilar)
	admin.Post("/similar-strategy/visual-preview", container.Handlers.AdminStrategy.VisualPreview)
	admin.Get("/similar-strategy/category-audit.csv", container.Handlers.AdminStrategy.CategoryAuditCSV)
	admin.Get("/similar-strategy/feature-catalog", container.Handlers.AdminStrategy.GetFeatureCatalog)
	admin.Get("/similar-strategy/copy-references", container.Handlers.AdminStrategy.CopyReferences)
	admin.Get("/similar-strategy/warm-status", container.Handlers.AdminStrategy.GetWarmStatus)
	admin.Get("/similar-strategy/presets", container.Handlers.AdminStrategy.ListPresets)
	admin.Post("/similar-strategy/presets", container.Handlers.AdminStrategy.CreatePreset)
	admin.Get("/similar-strategy/presets/:id", container.Handlers.AdminStrategy.GetPreset)
	admin.Put("/similar-strategy/presets/:id", container.Handlers.AdminStrategy.UpdatePreset)
	admin.Delete("/similar-strategy/presets/:id", container.Handlers.AdminStrategy.DeletePreset)
	admin.Post("/similar-strategy/presets/:id/promote", container.Handlers.AdminStrategy.PromotePreset)
	admin.Get("/outfit-strategy", container.Handlers.AdminOutfits.GetStrategy)
	admin.Put("/outfit-strategy", container.Handlers.AdminOutfits.SaveStrategy)
	admin.Post("/outfit-strategy/reset", container.Handlers.AdminOutfits.ResetStrategy)
	admin.Post("/outfit-strategy/preview", container.Handlers.AdminOutfits.Preview)
	admin.Get("/outfit-strategy/warm-status", container.Handlers.AdminOutfits.GetWarmStatus)
	admin.Get("/recommendation-composition", container.Handlers.AdminRecommendations.GetStrategy)
	admin.Put("/recommendation-composition", container.Handlers.AdminRecommendations.SaveStrategy)
	admin.Post("/recommendation-composition/reset", container.Handlers.AdminRecommendations.ResetStrategy)
	admin.Post("/recommendation-composition/preview", container.Handlers.AdminRecommendations.Preview)
	admin.Get("/feed-settings", container.Handlers.DataUpload.GetFeedSettings)
	admin.Put("/feed-settings", container.Handlers.DataUpload.SaveFeedSettings)
}

// setupFeedRoutes — управление загрузкой фида (ручной триггер + статус).
func setupFeedRoutes(app *fiber.App, container *container.Container) {
	api := app.Group("/api")
	api.Get("/process-feed", container.Handlers.DataUpload.ProcessFeed)
	api.Post("/download-feed", container.Handlers.DataUpload.DownloadFeed)
	api.Post("/upload-feed", container.Handlers.DataUpload.UploadFeed)
	api.Post("/feed/refresh", container.Handlers.DataUpload.RefreshFeed)
	api.Get("/data-status", container.Handlers.DataUpload.GetDataStatus)
}
