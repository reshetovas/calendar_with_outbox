package cron

import (
	"calendar/internal/application/use-cases"
	"context"
	"go.uber.org/zap"
)

// OutdatedJob - задача для удаления устаревших событий
type OutdatedJob struct {
	usecase use_cases.UseCaser
	logger  *zap.SugaredLogger
}

// NewOutdatedJob создает новую задачу для удаления устаревших событий
func NewOutdatedJob(usecase use_cases.UseCaser, logger *zap.SugaredLogger) *OutdatedJob {
	return &OutdatedJob{
		usecase: usecase,
		logger:  logger,
	}
}

// Run выполняет задачу удаления устаревших событий
func (j *OutdatedJob) Run(ctx context.Context) {
	j.logger.Info("Запуск задачи удаления устаревших событий")

	defer func() {
		if r := recover(); r != nil {
			j.logger.Errorf("Паника при выполнении задачи удаления событий: %v", r)
		}
	}()

	j.usecase.DeleteOldEventsByYear(ctx)
	j.logger.Info("Задача удаления устаревших событий завершена")
}
