package entity

import (
	"time"

	"github.com/gofrs/uuid"
)

type Event struct {
	ID                  uuid.UUID `json:"id" validate:"required"`
	Title               string    `json:"title" validate:"required,min=1,max=200"`
	DateEvent           string    `json:"dateEvent" validate:"required,rfc3339"`
	CreationDate        string    `json:"creationDate" validate:"required,rfc3339"`
	EndDateEvent        string    `json:"durationEvent" validate:"required,rfc3339"`
	DescriptionEvent    string    `json:"descriptionEvent" validate:"omitempty,max=1000"`
	UserID              string    `json:"userID" validate:"required,min=1,max=100"`
	TimeForNotification string    `json:"timeForNotification" validate:"omitempty,rfc3339_optional"`
	RqTm                string    `json:"RqTm" validate:"omitempty,rfc3339_optional"` //time request
}

type EventResponse struct {
	ID                  uuid.UUID `json:"id"`
	Title               string    `json:"title"`
	DateEvent           time.Time `json:"dateEvent"`
	CreationDate        time.Time `json:"creationDate"`
	EndDateEvent        time.Time `json:"durationEvent"`
	DescriptionEvent    string    `json:"descriptionEvent"`
	UserID              string    `json:"userID"`
	TimeForNotification time.Time `json:"timeForNotification"`
	RqTm                time.Time `json:"RqTm"` //time request
}
