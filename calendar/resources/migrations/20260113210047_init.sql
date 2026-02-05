-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events(
    id UUID    PRIMARY KEY NOT NULL,
    title VARCHAR(255),
    start_date_event TIMESTAMP,
    creation_date TIMESTAMP NOT NULL DEFAULT now(),
    end_date_event TIMESTAMP,
    description_event VARCHAR,
    user_id VARCHAR(255),
    time_for_notification TIMESTAMP,
    rq_tm TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd
