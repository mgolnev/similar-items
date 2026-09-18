package main

import (
	"context"
	"github.com/mgolnev/similar-items/backend/internal/config"
	"github.com/mgolnev/similar-items/backend/internal/container"
	"github.com/mgolnev/similar-items/backend/internal/database"
	"github.com/mgolnev/similar-items/backend/internal/routes"
	"log"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем базу данных
	db, err := database.Init(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Создаем миграции
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Создаем контейнер зависимостей
	container := container.NewContainer(db, cfg)

	// Запускаем планировщик для обновления фида
	container.Services.DataUpload.StartFeedScheduler(context.Background())

	// Прогрев id-кэша похожих в фоне: сервис не блокирует старт каталога.
	container.Services.Similar.StartWarm()
	container.Services.Outfits.StartWarm()

	// Настраиваем все маршруты
	routes.SetupRoutes(container.App, container)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := container.App.Listen(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
