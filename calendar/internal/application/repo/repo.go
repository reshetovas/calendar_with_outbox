package repo

import (
	"calendar/internal/appers"
	"calendar/internal/application/entity"
	"calendar/pkg/db"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

const (
	defaultDeleteDays = 365
)

type Repo interface {
	CreateEvent(ctx context.Context, evt *entity.Event) (bool, error)
	UpdateEvent(ctx context.Context, evt *entity.Event) error
	DeleteEvent(ctx context.Context, id string) error
	GetEvents(ctx context.Context, start, end time.Time) ([]*entity.EventResponse, error)
	DeleteOldEvents(ctx context.Context, days *int) error

	InsertOutbox(ctx context.Context, e *entity.OutboxEvent) error
	ReserveOutboxBatch(ctx context.Context, lease time.Duration, limit, maxAttempts int) ([]entity.OutboxEvent, error)
	MarkFailedWithBackoff(ctx context.Context, outboxID int, nextAttemptAt time.Time) error
	MarkGaveUp(ctx context.Context, outboxID int) error

	HealthCheck(ctx context.Context) error
}
type RepoImpl struct {
	db     db.DB
	logger *zap.SugaredLogger
}

func NewRepo(db db.DB, logger *zap.SugaredLogger) *RepoImpl {
	return &RepoImpl{db: db, logger: logger}
}

func (r *RepoImpl) HealthCheck(ctx context.Context) error {
	// Проверяем доступность БД через простой запрос
	var result int
	err := r.db.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}

func (r *RepoImpl) CreateEvent(ctx context.Context, evt *entity.Event) (bool, error) {
	r.logger.Debugf("[event: %s] start inserting into DB", evt.ID)

	var insertedID uuid.UUID
	err := r.db.QueryRow(ctx, createEvent,
		evt.ID, evt.Title, evt.DateEvent, evt.CreationDate, evt.EndDateEvent,
		evt.DescriptionEvent, evt.UserID, evt.TimeForNotification, evt.RqTm).Scan(&insertedID)

	switch {
	case err == nil:
		r.logger.Debugf("[event: %s] inserted into DB successfully", evt.ID)
		return true, nil
	case errors.Is(err, pgx.ErrNoRows):
		// ON CONFLICT DO NOTHING вернул 0 строк - событие уже существует
		r.logger.Warnf("[event: %s] inserting event: already exists (conflict)", evt.ID)
		return false, appers.ErrEventAlreadyExists
	case isDuplicateKeyError(err):
		r.logger.Warnf("[event: %s] inserting event: already exists (duplicate key)", evt.ID)
		return false, appers.ErrEventAlreadyExists
	default:
		r.logger.Errorf("[event: %s] error inserting into DB: %v", evt.ID, err)
		return false, fmt.Errorf("error inserting into DB: %w", err)
	}
}

func (r *RepoImpl) UpdateEvent(ctx context.Context, evt *entity.Event) error {
	r.logger.Debugf("[event: %s] start updating in DB", evt.ID)
	query, args := createPatchQuery(evt)
	if query == "" {
		r.logger.Warnf("[event: %s] no fields to update", evt.ID)
		return nil
	}

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Errorf("[event: %s] error updating in DB: %v", evt.ID, err)
		return fmt.Errorf("error updating in DB: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.Warnf("[event: %s] no rows updated", evt.ID)
		return appers.ErrEventNotFound
	}
	r.logger.Debugf("[event: %s] updated in DB successfully", evt.ID)
	return nil
}

func (r *RepoImpl) DeleteEvent(ctx context.Context, id string) error {
	r.logger.Debugf("[event: %s] start deleting from DB", id)

	result, err := r.db.Exec(ctx, deleteEvent, id)
	if err != nil {
		r.logger.Errorf("[event: %s] error deleting from DB: %v", id, err)
		return fmt.Errorf("error deleting from DB: %w", err)
	}
	if result.RowsAffected() == 0 {
		r.logger.Warnf("[event: %s] no rows deleted", id)
		return appers.ErrEventNotFound
	}
	r.logger.Debugf("[event: %s] deleted from DB successfully", id)
	return nil
}

func (r *RepoImpl) GetEvents(ctx context.Context, start, end time.Time) ([]*entity.EventResponse, error) {
	r.logger.Debugf("[start: %s, end: %s] start getting from DB", start, end)

	rows, err := r.db.Query(ctx, getEventsByPeriod, start, end)
	if err != nil {
		r.logger.Errorf("[start: %s, end: %s] error getting from DB: %v", start, end, err)
		return nil, fmt.Errorf("error getting from DB: %w", err)
	}

	events := make([]*entity.EventResponse, 0)
	defer rows.Close()
	for rows.Next() {
		var evt entity.EventResponse
		var updatedAt time.Time // Поле updated_at из таблицы, но не используется в EventResponse
		err := rows.Scan(&evt.ID, &evt.Title, &evt.DateEvent, &evt.CreationDate, &evt.EndDateEvent,
			&evt.DescriptionEvent, &evt.UserID, &evt.TimeForNotification, &evt.RqTm, &updatedAt)
		if err != nil {
			r.logger.Errorf("[start: %s, end: %s] error getting from DB: %v", start, end, err)
			return nil, fmt.Errorf("error getting from DB: %w", err)
		}
		events = append(events, &evt)
	}
	r.logger.Debugf("[start: %s, end: %s] got from DB successfully", start, end)
	return events, nil
}

func (r *RepoImpl) DeleteOldEvents(ctx context.Context, days *int) error {
	d := defaultDeleteDays
	if days != nil && *days > 0 {
		d = *days
	} else if days != nil && *days == 0 {
		r.logger.Warnf("daysToDelete is 0, skipping deletion to prevent deleting all events")
		return nil
	}

	r.logger.Infof("start deleting old events from DB: events older than %d days", d)

	result, err := r.db.Exec(ctx, deleteOldEvents, d)
	if err != nil {
		r.logger.Errorf("error deleting old events from DB: %v", err)
		return fmt.Errorf("error deleting old events from DB: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.logger.Infof("no rows deleted (no events older than %d days)", d)
		return nil
	}
	r.logger.Infof("deleted %d old events from DB (older than %d days)", rowsAffected, d)
	return nil
}

// /
func createPatchQuery(patch *entity.Event) (string, []any) {
	set := make([]string, 0, 8)
	args := make([]any, 0, 8)
	i := 1

	add := func(field string, value any) {
		set = append(set, fmt.Sprintf("%s = $%d", field, i))
		args = append(args, value)
		i++
	}

	if patch.Title != "" {
		add("title", patch.Title)
	}
	if patch.DescriptionEvent != "" {
		add("description_event", patch.DescriptionEvent)
	}
	if patch.UserID != "" {
		add("user_id", patch.UserID)
	}
	if patch.DateEvent != "" {
		add("start_date_event", patch.DateEvent)
	}
	if patch.EndDateEvent != "" {
		add("end_date_event", patch.EndDateEvent)
	}
	if patch.CreationDate != "" {
		add("creation_date", patch.CreationDate)
	}
	if patch.RqTm != "" {
		add("rq_tm", patch.RqTm)
	}
	if patch.TimeForNotification != "" {
		add("time_for_notification", patch.TimeForNotification)
	}

	if len(set) == 0 {
		return "", nil
	}

	set = append(set, "updated_at = now()")

	sb := strings.Builder{}
	sb.WriteString("UPDATE events SET ")
	sb.WriteString(strings.Join(set, ", "))
	sb.WriteString(" WHERE id = $")
	sb.WriteString(fmt.Sprint(i))
	args = append(args, patch.ID)

	return sb.String(), args
}

// isDuplicateKeyError проверяет, является ли ошибка ошибкой дубликата ключа (SQLSTATE 23505)
func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
