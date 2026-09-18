.PHONY: help build run dev test test-coverage clean deps-tidy pg-up pg-down docker-build docker-run

BINARY_NAME=similar-items
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Локальные дефолты (своя независимая БД на 5434, фид по расписанию раз в час)
DATABASE_URL ?= postgresql://similar:similar@localhost:5435/similar?sslmode=disable
FEED_SYNC_INTERVAL ?= 1h
PORT ?= 8080

GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-18s$(NC) %s\n", $$1, $$2}'

build: ## Собрать бинарь сервиса
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/similar

run: ## Запустить сервис (фид по расписанию включён)
	DATABASE_URL=$(DATABASE_URL) FEED_SYNC_INTERVAL=$(FEED_SYNC_INTERVAL) PORT=$(PORT) go run ./cmd/similar

dev: ## Запустить с hot reload (air) или fallback на run
	@if command -v air > /dev/null; then air; else echo "$(YELLOW)air не установлен, go install github.com/air-verse/air@latest$(NC)"; $(MAKE) run; fi

test: ## Запустить тесты
	go test ./...

test-coverage: ## Тесты с покрытием
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Очистить артефакты сборки
	rm -rf $(BUILD_DIR) coverage.out coverage.html

deps-tidy: ## go mod tidy
	go mod tidy

pg-up: ## Поднять PostgreSQL для сервиса
	docker compose up -d postgres

pg-down: ## Остановить PostgreSQL
	docker compose stop postgres

docker-build: ## Собрать Docker образ
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .

docker-run: ## Запустить через docker compose
	docker compose up -d
