package cron

import (
	use_cases "calendar/internal/application/use-cases"
	"calendar/pkg/config"
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Controller struct {
	scheduler *Scheduler
	logger    *zap.SugaredLogger
}

func NewController(ctx context.Context, logger *zap.SugaredLogger) *Controller {
	return &Controller{
		scheduler: NewScheduler(ctx),
		logger:    logger,
	}
}

// Поддерживает два режима:
// 1. По расписанию (cron format): например, "0 16 * * *" - каждый день в 16:00
// 2. По интервалу: например, "@every 1m" - каждую минуту
func (c *Controller) RegisterDeleteOldEventsJob(usecase use_cases.UseCaser, conf config.Cron) error {
	job := NewOutdatedJob(usecase, c.logger)

	var spec string

	// Приоритет: если указан Schedule, используем его, иначе Interval
	if conf.Schedule != "" {
		spec = conf.Schedule
		c.logger.Infof("Регистрация задачи удаления событий по расписанию: %s", spec)
	} else if conf.Interval != "" {
		spec = conf.Interval
		c.logger.Infof("Регистрация задачи удаления событий по интервалу: %s", spec)
	} else {
		// По умолчанию: каждую минуту
		spec = "@every 1m"
		c.logger.Warnf("⚠Расписание не указано, используется интервал по умолчанию: %s", spec)
	}

	entryID, err := c.scheduler.Add(spec, job)
	if err != nil {
		return fmt.Errorf("не удалось зарегистрировать задачу удаления событий: %w", err)
	}

	c.logger.Infof("Задача удаления событий зарегистрирована с ID: %d, расписание: %s", entryID, spec)
	return nil
}

// Start запускает планировщик задач
func (c *Controller) Start() {
	c.logger.Info("Запуск планировщика cron задач")
	c.scheduler.Start()
}

// Stop останавливает планировщик задач
func (c *Controller) Stop() {
	c.logger.Info("Остановка планировщика cron задач")
	c.scheduler.Stop()
	c.logger.Info("Планировщик cron задач остановлен")
}
