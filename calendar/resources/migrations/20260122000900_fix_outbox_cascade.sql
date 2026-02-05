-- +goose Up
-- +goose StatementBegin
-- Изменяем внешний ключ на CASCADE, чтобы при удалении события автоматически удалялись связанные записи в outbox_event
ALTER TABLE outbox_event 
DROP CONSTRAINT IF EXISTS fk_outbox_event;

ALTER TABLE outbox_event 
ADD CONSTRAINT fk_outbox_event 
FOREIGN KEY (aggregate_id) 
REFERENCES events(id) 
ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Возвращаем RESTRICT при откате
ALTER TABLE outbox_event 
DROP CONSTRAINT IF EXISTS fk_outbox_event;

ALTER TABLE outbox_event 
ADD CONSTRAINT fk_outbox_event 
FOREIGN KEY (aggregate_id) 
REFERENCES events(id) 
ON DELETE RESTRICT;
-- +goose StatementEnd
