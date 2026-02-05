package use_cases

import (
	"calendar/internal/application/entity"
	"calendar/internal/application/service"
	"calendar/pkg/config"
	"context"
	"time"

	"go.uber.org/zap"
)

type UseCaser interface {
	CreateEvent(ctx context.Context, event entity.Event) error
	GetEvent(ctx context.Context, start, end time.Time) ([]*entity.EventResponse, error)
	UpdateEvent(ctx context.Context, event entity.Event) error
	DeleteEvent(ctx context.Context, id string) error
	DeleteOldEventsByYear(ctx context.Context)
	RunRelay(ctx context.Context)
	ConsumerMessage(ctx context.Context, msg []byte, msgTime time.Time)

	HealthCheck(ctx context.Context) (dbHealthy bool, kafkaHealthy bool, err error)
}
type UseCase struct {
	service service.Service
	logger  *zap.SugaredLogger
	conf    *config.Config
}

func NewUseCase(service service.Service, logger *zap.SugaredLogger, conf *config.Config) *UseCase {
	return &UseCase{
		service: service,
		logger:  logger,
		conf:    conf,
	}
}

func (u *UseCase) HealthCheck(ctx context.Context) (dbHealthy bool, kafkaHealthy bool, err error) {
	return u.service.HealthCheck(ctx)
}

func (u *UseCase) CreateEvent(ctx context.Context, event entity.Event) error {
	u.logger.Debugf("[event: %s] CreateEvent started]", event.ID)
	return u.service.CreateEvent(ctx, &event)
}

func (u *UseCase) GetEvent(ctx context.Context, start, end time.Time) ([]*entity.EventResponse, error) {
	u.logger.Debugf("[start: %s, end: %s] GetPaymentsByPeriod started]", start, end)
	return u.service.GetEventsByPeriod(ctx, start, end)
}

func (u *UseCase) UpdateEvent(ctx context.Context, event entity.Event) error {
	u.logger.Debugf("[event: %s] UpdatePaymentstatus started]", event.ID)
	return u.service.UpdateEvent(ctx, &event)
}

func (u *UseCase) DeleteEvent(ctx context.Context, id string) error {
	u.logger.Debugf("[event: %s] DeletePayment started]", id)
	return u.service.DeleteEvent(ctx, id)
}

func (u *UseCase) DeleteOldEventsByYear(ctx context.Context) {
	days := u.conf.Cron.DaysToDelete
	u.logger.Infof("DeleteOldEventsByYear called with daysToDelete=%d", days)
	u.service.DeleteOldEventsByYear(ctx, &days)
}

func (u *UseCase) RunRelay(ctx context.Context) {
	u.logger.Debug("relay started")
	u.service.RelayEventRun(ctx)
}
func (u *UseCase) ConsumerMessage(ctx context.Context, msg []byte, msgTime time.Time) {
	u.logger.Debugf("consumer message: %s, time: %v", msg, msgTime)
}
