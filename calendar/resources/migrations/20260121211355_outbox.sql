-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS outbox_event (
    id               bigserial    PRIMARY KEY NOT NULL,
    aggregate_id     UUID         NOT NULL,                                                               -- FK на events.id
    aggregate_type   VARCHAR(32)  NOT NULL,                                                               -- например, "event"
    event_type       VARCHAR(64)  NOT NULL,                                                               -- "payment_created", "status_update"
    payload          JSONB        NOT NULL,                                                               -- полный JSON события
    status           VARCHAR(16)  NOT NULL DEFAULT 'NEW',                                                 -- статус доставки
    attempts         INT          NOT NULL DEFAULT 0 CHECK (attempts >= 0),                               -- попытки
    next_attempt_at  TIMESTAMP    NOT NULL DEFAULT now(),                                                 -- когда попробовать снова
    created_at       TIMESTAMP    NOT NULL DEFAULT now(),
    CONSTRAINT fk_outbox_event FOREIGN KEY (aggregate_id) REFERENCES events(id) ON DELETE RESTRICT,
    CONSTRAINT outbox_event_status_check CHECK (status IN ('NEW','SENT','FAILED','GAVE_UP'))
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE outbox_event;
-- +goose StatementEnd
