package repo

import (
	"calendar/internal/appers"
	"calendar/internal/application/entity"
	"calendar/pkg/config"
	"context"
	"fmt"
	"go.uber.org/zap"
)

type Transactions interface {
	CreateEvent(ctx context.Context, in *entity.Event, payload []byte) error
	GetOperationsFromOutbox(ctx context.Context, c config.RelayConfig) ([]entity.OutboxEvent, error)
	MarkSentAndUpdateEvent(ctx context.Context, outboxID int) error
}
type TransactionsImpl struct {
	repo   *RepoImpl
	logger *zap.SugaredLogger
}

func NewTransactions(repo *RepoImpl, logger *zap.SugaredLogger) *TransactionsImpl {
	return &TransactionsImpl{repo: repo, logger: logger}
}

func (t *TransactionsImpl) CreateEvent(ctx context.Context, in *entity.Event, payload []byte) error {

	if len(payload) == 0 {
		t.logger.Warnf("[ID %s] empty payload for outbox", in.ID)
	}

	return t.repo.db.WithinTransaction(ctx, func(ctx context.Context) error {
		inserted, err := t.repo.CreateEvent(ctx, in)
		if err != nil {
			t.logger.Errorf("[ID %s] insert event failed: %v", in.ID, err)
			return err
		}

		if inserted {
			evt := entity.OutboxEvent{
				AggregateID:   in.ID,
				AggregateType: entity.AggregateEvent,
				EventType:     entity.EventCreated,
				Payload:       payload,
				Status:        entity.OutboxNew,
			}

			if err = t.repo.InsertOutbox(ctx, &evt); err != nil {
				t.logger.Errorf("[ID %s] insert outbox failed: %v", in.ID, err)
				return err
			}
		} else {
			// запись уже существует
			t.logger.Infof("[ID %s] idempotent hit: Event already exists", in.ID)
			return appers.ErrEventAlreadyExists
		}
		return nil
	})
}

func (t *TransactionsImpl) GetOperationsFromOutbox(ctx context.Context, c config.RelayConfig) ([]entity.OutboxEvent, error) {
	var events []entity.OutboxEvent
	err := t.repo.db.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		events, err = t.repo.ReserveOutboxBatch(txCtx, c.Lease, c.BatchSize, c.MaxAttempts)
		return err
	})
	if err != nil {
		t.logger.Errorw("reserve outbox batch failed", "err", err)
		return nil, err
	}
	return events, nil
}

func (t *TransactionsImpl) MarkSentAndUpdateEvent(ctx context.Context, outboxID int) error {

	err := t.repo.db.WithinTransaction(ctx, func(ctx context.Context) error {
		t.logger.Infof("[ID %d] start transaction to mark event as sent", outboxID)
		result, err := t.repo.db.Exec(ctx, markSentSQL, outboxID, entity.OutboxSent)
		if err != nil {
			return fmt.Errorf("outbox mark sent: %w", err)
		}
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return fmt.Errorf("[ID %d] outbox not found", outboxID)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
