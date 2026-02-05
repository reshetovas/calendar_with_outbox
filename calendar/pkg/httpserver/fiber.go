package httpserver

import (
	"calendar/pkg/config"
	"calendar/pkg/metrics"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"strconv"
	"strings"
	"time"
)

func NewFiber(conf config.Config, m *metrics.Metrics) *fiber.App {
	app := fiber.New(
		fiber.Config{
			ReadBufferSize: 1024 * 100,
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				code := fiber.StatusInternalServerError
				return c.Status(code).JSON(fiber.Map{
					"status":  false,
					"message": err.Error(),
				})
			},
		},
	)

	app.Use(
		cors.New(cors.Config{
			AllowOrigins:  "*", // Разрешаем все источники по умолчанию
			ExposeHeaders: "Authorization",
		}),
		recover.New(),
		logger.New(),
	)

	// Prometheus middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// Получаем путь из роута, если доступен, иначе используем фактический путь
		path := c.Path()
		if r := c.Route(); r != nil && r.Path != "" {
			path = r.Path
		}

		// Получаем метод из роута, если доступен, иначе из заголовка запроса
		method := strings.ToUpper(c.Method())
		if r := c.Route(); r != nil && r.Method != "" {
			method = strings.ToUpper(r.Method)
		}

		// Нормализуем метод (убираем возможные опечатки и приводим к стандартному виду)
		method = normalizeHTTPMethod(method)

		status := c.Response().StatusCode()
		statusStr := strconv.Itoa(status)
		m.API.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
		m.API.HTTPRequestDuration.WithLabelValues(method, path, statusStr).Observe(time.Since(start).Seconds())
		return err
	})

	return app
}

// normalizeHTTPMethod нормализует HTTP метод, исправляя возможные опечатки
func normalizeHTTPMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))

	// Стандартные HTTP методы
	validMethods := map[string]string{
		"GET":     "GET",
		"POST":    "POST",
		"PUT":     "PUT",
		"DELETE":  "DELETE",
		"PATCH":   "PATCH",
		"HEAD":    "HEAD",
		"OPTIONS": "OPTIONS",
		"TRACE":   "TRACE",
		"CONNECT": "CONNECT",
		// Исправление возможных опечаток
		"GETT":  "GET",
		"POSTT": "POST",
		"PUTT":  "PUT",
	}

	if normalized, ok := validMethods[method]; ok {
		return normalized
	}

	// Если метод не найден, возвращаем как есть
	return method
}
