package main

import (
	"calendar/docs"
	"calendar/internal/application"
	"calendar/pkg/broker"
	"calendar/pkg/config"
	"calendar/pkg/db"
	"calendar/pkg/httpserver"
	"calendar/pkg/metrics"
	"calendar/pkg/observability"
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// @title           Calendar Service API
// @version         1.0
// @description     Микросервис календарь

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @BasePath /calendar/api

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger := observability.InitLogger(conf.LoggingLevel)

	logger.Infof("LOGGING_LEVEL = %s", conf.LoggingLevel)
	if strings.ToLower(conf.LoggingLevel) == "debug" {
		broker.EnableSaramaZapLogs(logger)
	}

	docs.SwaggerInfo.Host = conf.Server.SwaggerHost
	docs.SwaggerInfo.Schemes = []string{conf.Server.SwaggerSchema}

	m := metrics.New(prometheus.DefaultRegisterer)

	fiberServer := httpserver.NewFiber(conf, m)
	if fiberServer == nil {
		logger.Fatal(errors.New("fiber server is nil"))
	}

	store, err := db.NewPostgres(ctx, conf.Postgres)
	if err != nil {
		logger.Fatal(err)
	}

	kafka, err := broker.NewKafkaBroker(conf.Broker.Kafka, logger)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Infof("🚀 Kafka broker создан успешно. Consumer topic: %s, Producer topic: %s", kafka.ConsumerTopic, kafka.ProducerTopic)

	server := application.NewApp(ctx, &conf, logger, store, fiberServer, kafka, m)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Calendar service started successfully")
	logger.Info(fmt.Sprintf("Server config: %+v", conf.Server))

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.Run(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal("error listening for server: %w \n", err)
				return
			}

			logger.Infof("server %v closed\n", conf.Server.Port)
		}
	}()

	//graceful shutdown
	osSignal := <-interrupt
	switch osSignal {
	case os.Interrupt:
		logger.Infof("%v Got SIGINT...", conf.Server.Port)
	case syscall.SIGTERM:
		logger.Infof("%v Got SIGTERM...", conf.Server.Port)
	}

	cancel()

	store.Close()

	logger.Infof("postgres db connection closed")

	if err := server.Shutdown(); err != nil {
		logger.Fatalf("server %v forced to shutdown: %v", conf.Server.Port, err)
		return
	}

	logger.Infof("server shutdown %v done", conf.Server.Port)
}
