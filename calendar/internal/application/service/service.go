package service

import (
	"calendar/internal/application/entity"
	"calendar/internal/application/repo"
	"calendar/internal/transport/producer"
	"calendar/pkg/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type Service interface {
	CreateEvent(ctx context.Context, event *entity.Event) error
	GetEventsByPeriod(ctx context.Context, start, end time.Time) ([]*entity.EventResponse, error)
	UpdateEvent(ctx context.Context, event *entity.Event) error
	DeleteEvent(ctx context.Context, id string) error
	DeleteOldEventsByYear(ctx context.Context, days *int)
	RelayEventRun(ctx context.Context)

	HealthCheck(ctx context.Context) (dbHealthy bool, kafkaHealthy bool, err error)
}

type ServiceImpl struct {
	repo          repo.Repo
	transactions  repo.Transactions
	kafkaProducer producer.Producer
	logger        *zap.SugaredLogger
	cfg           *config.RelayConfig
}

func NewService(repo repo.Repo, transactions repo.Transactions, kafkaProducer producer.Producer, logger *zap.SugaredLogger, cfg *config.RelayConfig) *ServiceImpl {
	return &ServiceImpl{
		repo:          repo,
		transactions:  transactions,
		kafkaProducer: kafkaProducer,
		logger:        logger,
		cfg:           cfg,
	}
}

// HealthCheck проверяет доступность БД и Kafka
func (s *ServiceImpl) HealthCheck(ctx context.Context) (dbHealthy bool, kafkaHealthy bool, err error) {
	// Проверка БД через repo
	dbErr := s.repo.HealthCheck(ctx)
	dbHealthy = dbErr == nil

	// Проверка Kafka через producer
	kafkaErr := s.kafkaProducer.HealthCheck(ctx)
	kafkaHealthy = kafkaErr == nil

	// Возвращаем ошибку только если обе проверки провалились
	if !dbHealthy && !kafkaHealthy {
		return dbHealthy, kafkaHealthy, fmt.Errorf("database: %v, kafka: %v", dbErr, kafkaErr)
	}

	return dbHealthy, kafkaHealthy, nil
}

func (s *ServiceImpl) CreateEvent(ctx context.Context, event *entity.Event) error {
	s.logger.Debugf("[event: %s] CreateEvent started", event.ID)

	payload, err := json.Marshal(event)
	if err != nil {
		s.logger.Errorf("[event: %s] failed to marshal event to JSON: %v", event.ID, err)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return s.transactions.CreateEvent(ctx, event, payload)
}

func (s *ServiceImpl) GetEventsByPeriod(ctx context.Context, start, end time.Time) ([]*entity.EventResponse, error) {
	s.logger.Debugf("[start: %s, end: %s] GetPaymentsByPeriod started", start, end)

	return s.repo.GetEvents(ctx, start, end)
}

func (s *ServiceImpl) UpdateEvent(ctx context.Context, event *entity.Event) error {
	s.logger.Debugf("[event: %s] UpdateEventstatus started", event.ID)

	return s.repo.UpdateEvent(ctx, event)
}

func (s *ServiceImpl) DeleteEvent(ctx context.Context, id string) error {
	s.logger.Debugf("[event: %s] DeleteEvent started", id)

	return s.repo.DeleteEvent(ctx, id)
}

func (s *ServiceImpl) DeleteOldEventsByYear(ctx context.Context, days *int) {
	s.logger.Debugf("[days: %d] DeleteOldEventsByYear started", days)

	_ = s.repo.DeleteOldEvents(ctx, days)
}
