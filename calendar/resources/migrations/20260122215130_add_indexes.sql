-- +goose Up
-- +goose StatementBegin

-- Индексы для таблицы events

-- Индекс для поиска событий по периоду (getEventsByPeriod)
-- Используется в WHERE start_date_event >= $1 and end_date_event <= $2
CREATE INDEX IF NOT EXISTS idx_events_start_date_event ON events(start_date_event);
CREATE INDEX IF NOT EXISTS idx_events_end_date_event ON events(end_date_event);

-- Составной индекс для оптимизации запросов по периоду
-- Позволяет эффективно фильтровать по диапазону дат
CREATE INDEX IF NOT EXISTS idx_events_date_range ON events(start_date_event, end_date_event);

-- Индекс для удаления старых событий (deleteOldEvents)
-- Используется в WHERE creation_date < now() - make_interval(days => $1)
CREATE INDEX IF NOT EXISTS idx_events_creation_date ON events(creation_date);

-- Индексы для таблицы outbox_event

-- Составной индекс для резервирования батчей (reserveBatchSQL)
-- Используется в WHERE status IN ('NEW','FAILED') AND next_attempt_at <= now() AND attempts < $3
-- ORDER BY id FOR UPDATE SKIP LOCKED
-- Индекс оптимизирован для быстрого поиска записей готовых к обработке
CREATE INDEX IF NOT EXISTS idx_outbox_event_reserve ON outbox_event(status, next_attempt_at, attempts, id);

-- Дополнительный индекс на status для быстрой фильтрации по статусу
CREATE INDEX IF NOT EXISTS idx_outbox_event_status ON outbox_event(status);

-- Индекс на next_attempt_at для быстрого поиска записей готовых к повторной попытке
CREATE INDEX IF NOT EXISTS idx_outbox_event_next_attempt ON outbox_event(next_attempt_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_events_start_date_event;
DROP INDEX IF EXISTS idx_events_end_date_event;
DROP INDEX IF EXISTS idx_events_date_range;
DROP INDEX IF EXISTS idx_events_creation_date;

DROP INDEX IF EXISTS idx_outbox_event_reserve;
DROP INDEX IF EXISTS idx_outbox_event_status;
DROP INDEX IF EXISTS idx_outbox_event_next_attempt;

-- +goose StatementEnd
