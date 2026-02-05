package repo

import (
	"calendar/internal/application/common"
	"calendar/internal/application/entity"
	"context"
	"fmt"
	"time"
)

func (r *RepoImpl) InsertOutbox(ctx context.Context, e *entity.OutboxEvent) error {
	r.logger.Debugf("[event: %s] InsertOutbox started", e.AggregateID)
	_, err := r.db.Exec(ctx, insertOutboxQuery,
		e.AggregateID, e.AggregateType, e.EventType, []byte(e.Payload), string(e.Status),
	)
	if err != nil {
		return fmt.Errorf("insert outbox_event: %w", err)
	}

	return nil
}

func (r *RepoImpl) ReserveOutboxBatch(ctx context.Context, lease time.Duration, limit, maxAttempts int) ([]entity.OutboxEvent, error) {
	r.logger.Debugf("[lease: %s, limit: %d, maxAttempts: %d] ReserveOutboxBatch started", lease, limit, maxAttempts)

	rows, err := r.db.Query(ctx, reserveBatchSQL, common.PgInterval(lease), limit, maxAttempts)

	if err != nil {
		return nil, fmt.Errorf("reserve outbox batch: %w", err)
	}
	defer rows.Close()

	var res []entity.OutboxEvent
	for rows.Next() {
		var e entity.OutboxEvent
		var status string
		if err := rows.Scan(
			&e.ID, &e.AggregateID, &e.AggregateType, &e.EventType,
			&e.Payload, &status, &e.Attempts, &e.NextAttemptAt, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reserved outbox: %w", err)
		}
		e.Status = entity.OutboxStatus(status)
		res = append(res, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reserve rows err: %w", err)
	}

	return res, nil
}

func (r *RepoImpl) MarkFailedWithBackoff(ctx context.Context, outboxID int, nextAttemptAt time.Time) error {

	_, err := r.db.Exec(ctx, markFailedSQL, outboxID, entity.OutboxFailed, nextAttemptAt)
	if err != nil {
		return fmt.Errorf("outbox mark failed: %w", err)
	}

	return nil
}

func (r *RepoImpl) MarkGaveUp(ctx context.Context, outboxID int) error {

	_, err := r.db.Exec(ctx, markGaveUpSQL, outboxID, entity.OutboxGaveUp)
	if err != nil {
		return fmt.Errorf("outbox mark gave_up: %w", err)
	}

	return nil
}
