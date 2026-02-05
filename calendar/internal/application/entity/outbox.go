package entity

import (
	"encoding/json"
	"github.com/gofrs/uuid"
	"time"
)

type OutboxStatus string

const (
	OutboxNew    OutboxStatus = "NEW"
	OutboxSent   OutboxStatus = "SENT"
	OutboxFailed OutboxStatus = "FAILED"
	OutboxGaveUp OutboxStatus = "GAVE_UP"
)

type OutboxAggregate string

const (
	AggregateEvent OutboxAggregate = "event"
)

type OutboxEventType string

const (
	EventCreated OutboxEventType = "event_created"
)

type OutboxEvent struct {
	ID            int             `db:"id"`
	AggregateID   uuid.UUID       `db:"aggregate_id"`   // FK -> events.id
	AggregateType OutboxAggregate `db:"aggregate_type"` // "event"
	EventType     OutboxEventType `db:"event_type"`     // "event_created" / ...
	Payload       json.RawMessage `db:"payload"`        // JSONB для Kafka
	Status        OutboxStatus    `db:"status"`         // NEW | SENT | FAILED
	Attempts      int             `db:"attempts"`
	NextAttemptAt time.Time       `db:"next_attempt_at"`
	CreatedAt     time.Time       `db:"created_at"`
}
