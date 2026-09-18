package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Logger настройка логирования
func Logger() fiber.Handler {
	return logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
	})
}

// CORS настройка CORS
func CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	})
}

// Recover middleware для обработки паник
func Recover() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
	})
}

// RequestID добавляет уникальный ID к каждому запросу
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Set("X-Request-ID", requestID)
		}
		return c.Next()
	}
}

// RateLimit базовый rate limiting
func RateLimit(limit int, window time.Duration) fiber.Handler {
	// Простая реализация rate limiting
	// В продакшене лучше использовать Redis или другой внешний store
	return func(c *fiber.Ctx) error {
		// Здесь должна быть логика rate limiting
		return c.Next()
	}
}

// Auth middleware для аутентификации (заглушка)
func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Здесь должна быть логика аутентификации
		// Для простоты пропускаем все запросы
		return c.Next()
	}
}

// Validation middleware для валидации запросов
func Validation() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Здесь может быть общая валидация запросов
		return c.Next()
	}
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
