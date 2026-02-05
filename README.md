# Calendar Service

Микросервис для управления событиями календаря.

## API Методы

### Health Check
```bash
curl http://localhost:8081/health
```

### Создание события
```bash
curl -X POST http://localhost:8081/calendar/api/v1/event \
  -H "Content-Type: application/json" \
  -d '{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Встреча",
    "dateEvent": "2026-01-20T15:00:00Z",
    "creationDate": "2026-01-15T12:00:00Z",
    "durationEvent": "2026-01-20T18:00:00Z",
    "descriptionEvent": "Описание события",
    "userID": "user123",
    "timeForNotification": "2026-01-20T14:00:00Z",
    "RqTm": "2026-01-15T10:00:00Z"
  }'
```

### Получение событий за период
```bash
curl "http://localhost:8081/calendar/api/v1/event?start=2026-01-01T00:00:00Z&end=2026-01-31T23:59:59Z"
```

### Обновление события
```bash
curl -X PATCH http://localhost:8081/calendar/api/v1/event \
  -H "Content-Type: application/json" \
  -d '{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Обновленное событие",
    "dateEvent": "2026-01-20T16:00:00Z",
    "durationEvent": "2026-01-20T19:00:00Z"
  }'
```

### Удаление события
```bash
curl -X DELETE http://localhost:8081/calendar/api/v1/event/550e8400-e29b-41d4-a716-446655440000
```

### Swagger UI
```bash
# Откройте в браузере
http://localhost:8081/calendar/swagger/
```

## Cron и Kafka Consumer

### Cron задачи
Cron запускается автоматически при старте приложения. Удаляет события старше указанного количества дней.

**Настройка:**
- `cron.daysToDelete` - количество дней (по умолчанию 365)
- `cron.interval` - интервал выполнения (например, `@every 1m`)

**Логирование:** Все операции логируются в консоль.

### Kafka Consumer
Kafka consumer запускается автоматически в отдельной горутине при старте приложения.

**Настройка:**
- `broker.kafka.brokers` - адреса брокеров
- `broker.kafka.readerTopic` - топик для чтения
- `broker.kafka.writerTopic` - топик для записи

**Обработка:** Сообщения обрабатываются через `ConsumerMessage` use case.

## Запуск через Makefile

### Базовые команды
```bash
# Запуск без пересборки
make run

# Пересборка и запуск
make rebuild

# Остановка
make stop

# Просмотр логов
make logs

# Очистка (удаление контейнеров и volumes)
make clean
```

### Структура команд
- `run` - запускает контейнеры без пересборки
- `rebuild` - пересобирает образы и запускает контейнеры
- `stop` - останавливает и удаляет контейнеры
- `logs` - показывает логи всех сервисов
- `clean` - удаляет контейнеры и volumes (включая данные БД)

## Переменные окружения

Создайте файл `calendar/.env` в корне проекта:

```env
# PostgreSQL
postgres.conn_string=postgres://payment_user:payment_password@postgres:5432/payment_iban?sslmode=disable
postgres.max_connections=10

# Kafka
broker.kafka.brokers=kafka:29092
broker.kafka.readerTopic=calendar-events
broker.kafka.writerTopic=calendar-events
broker.kafka.maxAttempts=3

# Server
server.port=8081
server.swagger_host=localhost:8081
server.swagger_schema=http

# Logging
logging_level=info

# Relay (Outbox pattern)
relay.workers=2
relay.batchSize=10
relay.lease=30s
relay.pollPeriod=5s
relay.maxAttempts=3

# Cron
cron.daysToDelete=365
cron.interval=@every 1m
```

### Формат переменных
- Используйте точку (`.`) для вложенных структур
- Viper автоматически преобразует точки в подчеркивания для переменных окружения
- Пример: `postgres.conn_string` → `POSTGRES_CONN_STRING`

### Приоритет
1. Переменные окружения (высший приоритет)
2. Файл `.env` (если существует)
3. Значения по умолчанию в коде
