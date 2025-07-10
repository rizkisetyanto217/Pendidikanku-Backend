package logger

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// LoggerMiddleware untuk mencatat semua request
func LoggerMiddleware() fiber.Handler {
	return logger.New(logger.Config{
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Asia/Jakarta",
		Format:     "[${time}] ${ip} - ${method} ${path} - ${status} - ${latency}\n",
	})
}
