# --- build stage ---
FROM golang:1.25-alpine AS build

RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    make \
    bash \
    curl \
    sqlite-dev

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.1

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o calendar ./cmd/calendar \
 && go clean -modcache

# --- runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

COPY --from=build /app/calendar ./calendar
COPY --from=build /go/bin/goose /usr/local/bin/goose

# копируем миграции
COPY --from=build /app/resources/migrations ./resources/migrations

CMD ["./calendar"]
