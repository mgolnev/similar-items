package container

import (
	"github.com/mgolnev/similar-items/backend/internal/config"
	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/discovery/outfits"
	"github.com/mgolnev/similar-items/backend/internal/discovery/recommendations"
	"github.com/mgolnev/similar-items/backend/internal/discovery/similar"
	"github.com/mgolnev/similar-items/backend/internal/handlers"
	"github.com/mgolnev/similar-items/backend/internal/repository"
	"github.com/mgolnev/similar-items/backend/internal/services"
	"log"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Container контейнер зависимостей сервиса «Похожие».
type Container struct {
	Repositories *Repositories
	Services     *Services
	Handlers     *Handlers
	App          *fiber.App
}

// Repositories — репозитории, необходимые для загрузки фида и расчёта похожих.
type Repositories struct {
	Product         repository.ProductRepositoryInterface
	Sync            repository.SyncRepositoryInterface
	Similar         repository.SimilarRepositoryInterface
	Outfits         repository.OutfitRepositoryInterface
	RuntimeSettings repository.RuntimeSettingsRepositoryInterface
	StrategyPresets repository.StrategyPresetRepositoryInterface
}

// Services — сервисы.
type Services struct {
	DataUpload                *services.DataUploadService
	Similar                   *similar.Service
	Strategy                  *services.StrategyService
	Outfits                   *outfits.Service
	OutfitStrategy            *services.OutfitStrategyService
	Recommendations           *recommendations.Service
	RecommendationComposition *services.RecommendationCompositionService
}

// Handlers — HTTP-обработчики.
type Handlers struct {
	DataUpload           *handlers.DataUploadHandler
	Client               *handlers.ClientHandler
	AdminStrategy        *handlers.AdminStrategyHandler
	Outfits              *handlers.OutfitHandler
	AdminOutfits         *handlers.AdminOutfitStrategyHandler
	Recommendations      *handlers.RecommendationHandler
	AdminRecommendations *handlers.AdminRecommendationCompositionHandler
	SimilarRollout       *handlers.SimilarRolloutHandler
}

// NewContainer создает новый контейнер зависимостей.
func NewContainer(db *database.DB, cfg *config.Config) *Container {
	repos := &Repositories{
		Product:         repository.NewProductRepository(db),
		Sync:            repository.NewSyncRepository(db),
		Similar:         repository.NewSimilarRepository(db),
		Outfits:         repository.NewOutfitRepository(db),
		RuntimeSettings: repository.NewRuntimeSettingsRepository(db),
		StrategyPresets: repository.NewStrategyPresetRepository(db),
	}

	strategySvc := services.NewStrategyService(repos.RuntimeSettings, repos.StrategyPresets)
	if err := strategySvc.EnsurePresetsBootstrapped(); err != nil {
		log.Printf("Warning: preset bootstrap failed: %v", err)
	}
	similarSvc := similar.NewService(repos.Similar, strategySvc)
	similarSvc.SetWarmWorkers(cfg.WarmWorkers)
	similarSvc.SetWarmBudget(cfg.WarmTimeout)
	outfitStrategySvc := services.NewOutfitStrategyService(repos.RuntimeSettings)
	outfitSvc := outfits.NewService(repos.Outfits, outfitStrategySvc)
	outfitSvc.SetWarmBudget(cfg.WarmTimeout)
	recommendationCompositionSvc := services.NewRecommendationCompositionService(repos.RuntimeSettings)
	recommendationSvc := recommendations.NewService(similarSvc, outfitSvc, recommendationCompositionSvc)

	dataUploadSvc := services.NewDataUploadService(
		repos.Product,
		repos.Sync,
		cfg.FeedURL,
		cfg.FeedSyncInterval,
	).WithChangeHook(func() {
		// Serving snapshot остаётся доступен до атомарной публикации полностью
		// прогретого поколения для нового каталога. StartWarm coalesce-ит события,
		// но не теряет refresh, пришедший во время уже идущего прогрева.
		similarSvc.StartWarm()
		outfitSvc.StartWarm()
	})
	dataUploadSvc.ApplyPersistedFeedSettings(strategySvc, cfg.FeedURL, cfg.FeedSyncInterval)

	svcs := &Services{
		DataUpload:                dataUploadSvc,
		Similar:                   similarSvc,
		Strategy:                  strategySvc,
		Outfits:                   outfitSvc,
		OutfitStrategy:            outfitStrategySvc,
		Recommendations:           recommendationSvc,
		RecommendationComposition: recommendationCompositionSvc,
	}

	clientHandler := handlers.NewClientHandler(similarSvc)
	recommendationHandler := handlers.NewRecommendationHandler(recommendationSvc)
	hs := &Handlers{
		DataUpload:           handlers.NewDataUploadHandler(svcs.DataUpload, strategySvc),
		Client:               clientHandler,
		AdminStrategy:        handlers.NewAdminStrategyHandler(strategySvc, similarSvc, repos.Product),
		Outfits:              handlers.NewOutfitHandler(outfitSvc),
		AdminOutfits:         handlers.NewAdminOutfitStrategyHandler(outfitStrategySvc, outfitSvc),
		Recommendations:      recommendationHandler,
		AdminRecommendations: handlers.NewAdminRecommendationCompositionHandler(recommendationCompositionSvc, recommendationSvc),
		SimilarRollout:       handlers.NewSimilarRolloutHandler(recommendationCompositionSvc, clientHandler, recommendationHandler),
	}

	c := &Container{
		Repositories: repos,
		Services:     svcs,
		Handlers:     hs,
	}

	c.App = c.createApp()
	return c
}

// createApp создает и настраивает Fiber приложение.
func (c *Container) createApp() *fiber.App {
	app := fiber.New()

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))

	return app
}
