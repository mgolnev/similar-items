package config

import (
	"os"
	"strconv"
	"time"
)

// Config содержит конфигурацию приложения
type Config struct {
	Port             string
	DatabaseURL      string
	TemplatePath     string
	StaticPath       string
	FeedURL          string
	FeedSyncInterval time.Duration
	// WarmTimeout — бюджет одного прогрева idCache (на старте и после фид-синка).
	// env WARM_TIMEOUT (Go duration: "90m", "2h"); по умолчанию 30m.
	WarmTimeout time.Duration
	// WarmWorkers — число воркеров прогрева. env WARM_WORKERS; 0 (default) = авто:
	// GOMAXPROCS-1 (минимум 1) — прогрев CPU-bound, одно «ядро» оставляем под
	// обслуживание запросов и GC.
	WarmWorkers int
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://similar:similar@localhost:5435/similar?sslmode=disable"
	}

	templatePath := os.Getenv("TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "./web/templates"
	}

	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		staticPath = "./web/static"
	}

	feedURL := os.Getenv("FEED_URL")
	if feedURL == "" {
		feedURL = DefaultFeedURL
	} else {
		feedURL = NormalizeFeedURL(feedURL)
	}

	feedInterval := time.Hour
	if intervalStr := os.Getenv("FEED_SYNC_INTERVAL"); intervalStr != "" {
		if intervalStr == "0" {
			feedInterval = 0
		} else if parsed, err := time.ParseDuration(intervalStr); err == nil && parsed > 0 {
			feedInterval = parsed
		}
	}

	warmTimeout := 30 * time.Minute
	if warmStr := os.Getenv("WARM_TIMEOUT"); warmStr != "" {
		if parsed, err := time.ParseDuration(warmStr); err == nil && parsed > 0 {
			warmTimeout = parsed
		}
	}

	warmWorkers := 0 // авто (GOMAXPROCS-1)
	if wwStr := os.Getenv("WARM_WORKERS"); wwStr != "" {
		if parsed, err := strconv.Atoi(wwStr); err == nil && parsed > 0 {
			warmWorkers = parsed
		}
	}

	return &Config{
		Port:             port,
		DatabaseURL:      databaseURL,
		TemplatePath:     templatePath,
		StaticPath:       staticPath,
		FeedURL:          feedURL,
		FeedSyncInterval: feedInterval,
		WarmTimeout:      warmTimeout,
		WarmWorkers:      warmWorkers,
	}
}
