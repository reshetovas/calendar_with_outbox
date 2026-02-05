package service

import (
	"calendar/internal/application/common"
	"calendar/internal/application/entity"
	"context"
	"time"
)

func (s *ServiceImpl) RelayEventRun(ctx context.Context) {
	s.logger.Infow("relay started", "workers", s.cfg.Workers, "batch", s.cfg.BatchSize, "lease", s.cfg.Lease.String())

	jobs := make(chan entity.OutboxEvent, s.cfg.BatchSize*2)

	// стартуем воркеров
	for i := 0; i < s.cfg.Workers; i++ {
		go s.worker(ctx, i, jobs)
	}

	ticker := time.NewTicker(s.cfg.PollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("relay stopping")
			return
		case <-ticker.C:
			events, err := s.transactions.GetOperationsFromOutbox(ctx, *s.cfg)
			if err != nil {
				s.logger.Errorw("get operations from outbox failed", "err", err)
				continue
			}

			s.logger.Debugf("len jobs: %d, len events: %d", len(jobs), len(events))
			for _, e := range events {
				select {
				case jobs <- e:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (s *ServiceImpl) worker(ctx context.Context, id int, jobs <-chan entity.OutboxEvent) {
	s.logger.Infow("worker started", "id", id)
	for {
		select {
		case <-ctx.Done():
			s.logger.Infow("worker stopping", "id", id)
			return
		case e := <-jobs:
			s.ProcessOne(ctx, id, e)
		}
	}
}

// ProcessOne обрабатывает одно событие из outbox (экспортируем для тестирования)
func (s *ServiceImpl) ProcessOne(ctx context.Context, wid int, e entity.OutboxEvent) {
	s.logger.Debugf("[ID %d] relay-process started, workerID: %d", e.ID, wid)

	// Отправка сообщения в Kafka
	if err := s.kafkaProducer.ProduceMessage(ctx, e.ID, e.Payload); err != nil {
		s.logger.Errorf("[ID %d] kafka send failed, err: %v", e.ID, err)
		_ = s.markOutboxFailedOrGaveUp(context.Background(), e.ID, e.Attempts, s.cfg.MaxAttempts, common.NextBackoffWithJitter(e.Attempts))

		return
	}
	s.logger.Infof("[ID %d] sent to kafka", e.ID)

	//обновление в БД
	if err := s.transactions.MarkSentAndUpdateEvent(ctx, e.ID); err != nil {
		// сообщение уже ушло — повторно слать нельзя; статус апдейтим при следующем цикле
		s.logger.Errorf("[ID %d] mark sent & update Event failed, err:  %v", e.ID, err)
		_ = s.repo.MarkGaveUp(ctx, e.ID)

		return
	}
	s.logger.Infof("[ID %d] updated Event", e.ID)

	s.logger.Infof("[ID %d] relay-process completed", e.ID)
}

func (s *ServiceImpl) markOutboxFailedOrGaveUp(ctx context.Context, outboxID int, attempts, maxAttempts int, backoff time.Duration) error {
	if attempts+1 >= maxAttempts {
		return s.repo.MarkGaveUp(ctx, outboxID)
	}
	return s.repo.MarkFailedWithBackoff(ctx, outboxID, time.Now().UTC().Add(backoff))
}
