package common

import (
	"calendar/internal/appers"
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	// допустимы: "123", "123.4", "123,45", "+0.99", "-10", пробелы по краям
	reDec = regexp.MustCompile(`^\s*([+-])?(\d+)(?:[.,](\d+))?\s*$`)

	// NUMERIC(18,2) -> максимум 16 цифр в целой части (с учётом 2 знаков после запятой)
	maxIntDigits = 16
	maxScale     = 2
)

// NumericFromString2Strict парсит строку в точное десятичное число
// с масштабом не более 2 и целой частью до 16 знаков.
// Ничего не округляет: если больше 2 знаков после запятой — вернёт ошибку.
// Если строка пустая или содержит только пробелы, возвращает невалидный pgtype.Numeric (Valid = false).
func NumericFromString2Strict(s string) (pgtype.Numeric, error) {
	var zero pgtype.Numeric

	// Если строка пустая или только пробелы - возвращаем невалидный Numeric
	s = strings.TrimSpace(s)
	if s == "" {
		return zero, nil // Valid = false, но без ошибки
	}

	m := reDec.FindStringSubmatch(s)
	if m == nil {
		return zero, appers.ErrFormat
	}
	sign := m[1]
	intPart := trimZeros(m[2])
	frac := m[3]

	if len(frac) > maxScale {
		return zero, appers.ErrScale
	}
	if len(intPart) > maxIntDigits {
		return zero, appers.ErrPrecision
	}

	if frac == "" {
		frac = "00"
	} else if len(frac) == 1 {
		frac += "0"
	}
	canonical := sign + intPart + "." + frac

	var n pgtype.Numeric
	if err := n.Scan(canonical); err != nil {
		return zero, err
	}
	return n, nil
}

func NumericToString(n pgtype.Numeric) (string, error) {
	if !n.Valid {
		return "", nil // NULL
	}
	v, err := n.Value()
	if err != nil {
		return "", err
	}
	switch vv := v.(type) {
	case string:
		return vv, nil
	case []byte:
		return string(vv), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprint(vv), nil
	}
}

func trimZeros(s string) string {
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return "0"
	}
	return s
}

func PgInterval(d time.Duration) string {
	sec := int64(d / time.Second)
	return fmt.Sprintf("%d seconds", sec)
}

func NextBackoffWithJitter(attempts int) time.Duration {
	if attempts < 0 {
		attempts = 0
	}

	base := time.Second << attempts

	limit := 30 * time.Minute
	if base > limit {
		base = limit
	}

	jitter := time.Duration(rand.Int63n(int64(base / 2)))

	return base/2 + jitter
}

func SleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer func() {
		if !t.Stop() {
			select {
			case <-t.C:
			default:
			}
		}
	}()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
