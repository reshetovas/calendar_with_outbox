package entity

// HealthCheckResponse структура ответа для health check
type HealthCheckResponse struct {
	Status  bool                    `json:"status" example:"true"`
	Message string                  `json:"message" example:"success"`
	Version string                  `json:"version" example:"0.1.0"`
	Checks  HealthCheckResponseData `json:"checks"`
}

// HealthCheckResponseData детали проверок
type HealthCheckResponseData struct {
	Database HealthCheckItem `json:"database"`
	Kafka    HealthCheckItem `json:"kafka"`
}

// HealthCheckItem информация о проверке компонента
type HealthCheckItem struct {
	Status bool   `json:"status" example:"true"`
	Type   string `json:"type" example:"postgresql"`
	Error  string `json:"error,omitempty" example:"Database connection failed"`
}
