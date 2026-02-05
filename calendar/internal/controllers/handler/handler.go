package handler

import (
	"calendar/internal/appers"
	"calendar/internal/application/common"
	"calendar/internal/application/entity"
	use_cases "calendar/internal/application/use-cases"
	"calendar/pkg/validator"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	playgroundvalidator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler interface {
	CreateEvent(c *fiber.Ctx) error
	GetEventsByPeriod(c *fiber.Ctx) error
	UpdateEvent(c *fiber.Ctx) error
	DeleteEvent(c *fiber.Ctx) error
	HealthCheck(c *fiber.Ctx) error
}
type HandlerImpl struct {
	usecase use_cases.UseCaser
	logger  *zap.SugaredLogger
}

func NewEventHandler(usecase use_cases.UseCaser, logger *zap.SugaredLogger) *HandlerImpl {
	return &HandlerImpl{
		usecase: usecase,
		logger:  logger,
	}
}

// formatValidationErrors форматирует ошибки валидации в понятный формат для клиента
func formatValidationErrors(err error) fiber.Map {
	var errors []string
	if validationErrors, ok := err.(playgroundvalidator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := e.Field()
			tag := e.Tag()
			var message string
			switch tag {
			case "required":
				message = fmt.Sprintf("поле '%s' обязательно для заполнения", field)
			case "min":
				message = fmt.Sprintf("поле '%s' должно содержать минимум %s символов", field, e.Param())
			case "max":
				message = fmt.Sprintf("поле '%s' должно содержать максимум %s символов", field, e.Param())
			case "rfc3339", "rfc3339_optional":
				message = fmt.Sprintf("поле '%s' должно быть в формате RFC3339 (например, 2026-01-20T15:00:00Z)", field)
			default:
				message = fmt.Sprintf("поле '%s' не прошло валидацию: %s", field, tag)
			}
			errors = append(errors, message)
		}
	} else {
		errors = append(errors, err.Error())
	}
	return fiber.Map{
		"error":   "validation failed",
		"details": errors,
	}
}

// validateEventDates выполняет логическую валидацию дат события
func validateEventDates(event *entity.Event) error {
	dateEvent, err := time.Parse(time.RFC3339, event.DateEvent)
	if err != nil {
		return fmt.Errorf("неверный формат dateEvent: %w", err)
	}

	endDateEvent, err := time.Parse(time.RFC3339, event.EndDateEvent)
	if err != nil {
		return fmt.Errorf("неверный формат durationEvent: %w", err)
	}

	// Проверяем, что дата окончания после даты начала
	if !endDateEvent.After(dateEvent) {
		return fmt.Errorf("дата окончания события должна быть после даты начала")
	}

	// Если указано время уведомления, проверяем что оно до начала события
	if event.TimeForNotification != "" {
		notificationTime, err := time.Parse(time.RFC3339, event.TimeForNotification)
		if err != nil {
			return fmt.Errorf("неверный формат timeForNotification: %w", err)
		}
		if !notificationTime.Before(dateEvent) {
			return fmt.Errorf("время уведомления должно быть до начала события")
		}
	}

	return nil
}

// HealthCheck godoc
// @Summary     Проверка состояния сервиса
// @Description Проверяет доступность базы данных PostgreSQL и Kafka (Producer и Consumer). Возвращает детальную информацию о состоянии каждого компонента.
// @Accept      json
// @Produce     json
// @Success     200   {object} entity.HealthCheckResponse "Все сервисы доступны"
// @Failure     503   {object} entity.HealthCheckResponse "Один или несколько сервисов недоступны"
// @tags        Health
// @Router      /health [get]
func (h *HandlerImpl) HealthCheck(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbHealthy, kafkaHealthy, _ := h.usecase.HealthCheck(ctx)

	health := fiber.Map{
		"status":  dbHealthy && kafkaHealthy,
		"message": "success",
		"version": common.Version,
		"checks": fiber.Map{
			"database": fiber.Map{
				"status": dbHealthy,
				"type":   "postgresql",
			},
			"kafka": fiber.Map{
				"status": kafkaHealthy,
				"type":   "kafka",
			},
		},
	}
	if !dbHealthy {
		health["checks"].(fiber.Map)["database"].(fiber.Map)["error"] = "Database connection failed"
		health["message"] = "Some services are unavailable"
	}
	if !kafkaHealthy {
		health["checks"].(fiber.Map)["kafka"].(fiber.Map)["error"] = "Kafka connection failed"
		health["message"] = "Some services are unavailable"
	}

	if !dbHealthy || !kafkaHealthy {
		return c.Status(fiber.StatusServiceUnavailable).JSON(health)
	}

	return c.Status(fiber.StatusOK).JSON(health)
}

// CreateEvent godoc
// @Summary     Создание события
// @Description Создает новое событие и записывает его в БД
// @Accept      json
// @Produce     json
// @Param       body  body     entity.Event  true  "Данные события"
// @Success     200
// @Failure     400
// @Failure     409
// @Failure     500
// @tags        Event
// @Router      /v1/event [post]
func (h *HandlerImpl) CreateEvent(c *fiber.Ctx) error {
	var event entity.Event
	err := c.BodyParser(&event)
	if err != nil {
		h.logger.Errorf("error parsing body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Валидация структуры
	if err = validator.Validate.Struct(&event); err != nil {
		h.logger.Warnf("validation error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(formatValidationErrors(err))
	}

	// Логическая валидация дат
	if err = validateEventDates(&event); err != nil {
		h.logger.Warnf("date validation error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	err = h.usecase.CreateEvent(c.Context(), event)
	switch {
	case errors.Is(err, appers.ErrEventAlreadyExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"description": err.Error()})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"description": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"description": "ok"})

}

// GetEventsByPeriod godoc
// @Summary     Получение событий за период
// @Description Возвращает список событий за период, заданный query-параметрами start и end
// @Produce     json
// @Param       start  query    string true "Дата/время начала периода (например, 2026-01-01T00:00:00Z)"
// @Param       end    query    string true "Дата/время конца периода (например, 2026-01-31T23:59:59Z)"
// @Success     200    {array}  entity.Event
// @Failure     400
// @Failure     409
// @Failure     500
// @tags        Event
// @Router      /v1/event [get]
func (h *HandlerImpl) GetEventsByPeriod(c *fiber.Ctx) error {
	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "start and end are required",
		})
	}

	// Декодируем URL-encoded параметры
	var err error
	startStr, err = url.QueryUnescape(startStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid start parameter encoding",
		})
	}
	endStr, err = url.QueryUnescape(endStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid end parameter encoding",
		})
	}

	// Убираем кавычки, если они есть
	startStr = strings.Trim(startStr, `"`)
	endStr = strings.Trim(endStr, `"`)

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid start format, expected RFC3339 (e.g., 2026-01-20T11:00:00Z)",
		})
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid end format, expected RFC3339 (e.g., 2026-01-20T19:00:00Z)",
		})
	}
	start = start.UTC()
	end = end.UTC()

	events, err := h.usecase.GetEvent(c.Context(), start, end)
	if err != nil {
		return appers.SanitizeError(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(events)
}

// UpdateEvent godoc
// @Summary     Обновление события
// @Description Обновляет существующее событие по данным из тела запроса
// @Accept      json
// @Produce     json
// @Param       body  body     entity.Event  true  "Данные события для обновления"
// @Success     200
// @Failure     400
// @Failure     404
// @Failure     500
// @tags        Event
// @Router      /v1/event [patch]
func (h *HandlerImpl) UpdateEvent(c *fiber.Ctx) error {
	var event entity.Event
	err := c.BodyParser(&event)
	if err != nil {
		h.logger.Errorf("error parsing body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Валидация структуры (fail fast - best practice для highload)
	if err = validator.Validate.Struct(&event); err != nil {
		h.logger.Warnf("validation error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(formatValidationErrors(err))
	}

	// Логическая валидация дат
	if err = validateEventDates(&event); err != nil {
		h.logger.Warnf("date validation error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	err = h.usecase.UpdateEvent(c.Context(), event)
	switch {
	case errors.Is(err, appers.ErrEventNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"description": err.Error()})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"description": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"description": "ok"})
}

// DeleteEvent godoc
// @Summary     Удаление события
// @Description Удаляет событие по идентификатору
// @Accept      json
// @Produce     json
// @Param       id   path     string  true  "ID события"
// @Success     200
// @Failure     400
// @Failure     404
// @Failure     500
// @tags        Event
// @Router      /v1/event/{id} [delete]
func (h *HandlerImpl) DeleteEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.usecase.DeleteEvent(c.Context(), id)
	switch {
	case errors.Is(err, appers.ErrEventNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"description": err.Error()})
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"description": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"description": "ok"})
}
