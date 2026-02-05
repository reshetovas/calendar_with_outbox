package appers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

var (
	// ErrFormat для парсинга строки в pgtype.Numeric
	ErrFormat    = errors.New("invalid decimal format")
	ErrScale     = errors.New("too many fractional digits (max 2)")
	ErrPrecision = errors.New("too many integer digits for NUMERIC(18,2)")
)

type ErrorResp struct {
	StatusCode int    `json:"statusCode,omitempty"`
	StatusDesc string `json:"statusDesc,omitempty"`
}

func (e ErrorResp) Error() string {
	return e.StatusDesc
}

var (
	ErrEventNotFound = ErrorResp{
		http.StatusNotFound,
		"не найден",
	}
	ErrEventAlreadyExists = ErrorResp{
		http.StatusForbidden,
		"уже создан",
	}
	ErrEventFormatDate = ErrorResp{
		StatusCode: http.StatusBadRequest,
		StatusDesc: "не верный формат даты, должен быть YYYY-MM-DD",
	}
)

func SanitizeError(c *fiber.Ctx, err error) error {
	var errResp ErrorResp

	if ok := errors.As(err, &errResp); ok {
		return c.Status(errResp.StatusCode).JSON(fiber.Map{
			"message": errResp.StatusDesc,
		})
	} else {
		return NewErr(c, http.StatusInternalServerError, err)
	}
}

func NewErr(ctx *fiber.Ctx, status int, err error) error {
	return ctx.Status(status).JSON(fiber.Map{
		"message": err.Error(),
	})
}
