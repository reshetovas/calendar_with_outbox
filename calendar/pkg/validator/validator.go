package validator

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	// Validate - singleton экземпляр валидатора для переиспользования (best practice для highload)
	Validate *validator.Validate
)

func init() {
	Validate = validator.New()

	// Регистрируем кастомные валидаторы
	_ = Validate.RegisterValidation("rfc3339", validateRFC3339)
	_ = Validate.RegisterValidation("rfc3339_optional", validateRFC3339Optional)
}

// validateRFC3339 проверяет, что строка является валидной RFC3339 датой
func validateRFC3339(fl validator.FieldLevel) bool {
	dateStr := fl.Field().String()
	if dateStr == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, dateStr)
	return err == nil
}

// validateRFC3339Optional проверяет RFC3339 дату, но разрешает пустую строку
func validateRFC3339Optional(fl validator.FieldLevel) bool {
	dateStr := fl.Field().String()
	if dateStr == "" {
		return true // опциональное поле может быть пустым
	}
	_, err := time.Parse(time.RFC3339, dateStr)
	return err == nil
}
