package handler

import (
	"calendar/pkg/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"go.uber.org/zap"
)

type Router struct {
	handler Handler
	app     *fiber.App
	conf    *config.Config
	logger  *zap.SugaredLogger
}

func NewRouter(handler Handler, app *fiber.App, conf *config.Config, logger *zap.SugaredLogger) *Router {
	return &Router{
		logger:  logger,
		app:     app,
		conf:    conf,
		handler: handler,
	}
}

func (r *Router) RegisterRouter() {
	r.app.Get("/health", r.handler.HealthCheck)

	r.app.Use(
		recover.New(recover.Config{
			EnableStackTrace: true,
		}),
		logger.New(),
	)

	r.app.Route("/calendar", func(router fiber.Router) {

		router.Use("/swagger/*", swagger.New(swagger.Config{
			DeepLinking: false,
			URL:         "/calendar/swagger/doc.json",
		}))

		api := router.Group("/api")

		v1 := api.Group("/v1")

		v1.Post("/event", r.handler.CreateEvent)
		v1.Get("/event", r.handler.GetEventsByPeriod)
		v1.Patch("/event", r.handler.UpdateEvent)
		v1.Delete("/event/:id", r.handler.DeleteEvent)
	})
}
