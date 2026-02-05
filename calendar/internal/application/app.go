package application

import (
	"calendar/internal/application/common"
	"calendar/internal/application/repo"
	"calendar/internal/application/service"
	"calendar/internal/application/use-cases"
	"calendar/internal/controllers/cron"
	"calendar/internal/controllers/handler"
	"calendar/internal/controllers/listener"
	"calendar/internal/transport/producer"
	"calendar/pkg/broker"
	"calendar/pkg/config"
	"calendar/pkg/db"
	"calendar/pkg/metrics"
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type App struct {
	ctx            context.Context
	conf           *config.Config
	logger         *zap.SugaredLogger
	postgres       *db.Postgres
	httpServer     *fiber.App
	kafka          *broker.KafkaBroker
	cronController *cron.Controller
}

func NewApp(
	ctx context.Context,
	conf *config.Config,
	logger *zap.SugaredLogger,
	postgres *db.Postgres,
	httpServer *fiber.App,
	kafkaBroker *broker.KafkaBroker,
	m *metrics.Metrics) *App {
	//Логируем версию приложения
	logger.Infof("Запуск Calendar Service версии: %s", common.Version)

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("закрытие consumer group")
				kafkaBroker.ConsumerGroup.Close()
				logger.Info("закрытие consumer group: done")
				return
			}
		}
	}()

	store := repo.NewRepo(postgres, logger)
	tx := repo.NewTransactions(store, logger)
	kafkaProducer := producer.NewProducer(kafkaBroker, logger, conf.Broker.Kafka.MaxAttempts, m)
	srv := service.NewService(store, tx, kafkaProducer, logger, &conf.Realay)
	uc := use_cases.NewUseCase(srv, logger, conf)
	h := handler.NewEventHandler(uc, logger)
	r := handler.NewRouter(h, httpServer, conf, logger)

	// Инициализация cron контроллера
	cronController := cron.NewController(ctx, logger)
	if err := cronController.RegisterDeleteOldEventsJob(uc, conf.Cron); err != nil {
		logger.Fatalf("не удалось зарегистрировать cron задачу: %v", err)
	}
	cronController.Start()

	go uc.RunRelay(ctx)

	r.RegisterRouter()

	app := &App{
		ctx:            ctx,
		conf:           conf,
		postgres:       postgres,
		httpServer:     httpServer,
		kafka:          kafkaBroker,
		cronController: cronController,
	}

	go app.runConsumer(ctx, logger, uc, kafkaBroker)

	return app
}

func (a *App) Run() error {
	return a.httpServer.Listen(fmt.Sprintf(":%s", a.conf.Server.Port))
}

func (a *App) Shutdown() error {
	// Останавливаем cron задачи
	if a.cronController != nil {
		a.cronController.Stop()
	}
	return a.httpServer.Shutdown()
}

func (a *App) runConsumer(ctx context.Context, logger *zap.SugaredLogger, usecase use_cases.UseCaser, kafkaBroker *broker.KafkaBroker) {
	logger.Infof("🚀 Запуск consumer для топика: %s", kafkaBroker.ConsumerTopic)

	kafkaBrokerConsumer := listener.NewKafkaBrokerConsumer(usecase, logger)

	for {
		logger.Infof("🔄 Попытка подключения к consumer group...")
		err := kafkaBroker.ConsumerGroup.Consume(ctx, []string{kafkaBroker.ConsumerTopic}, kafkaBrokerConsumer)
		if err != nil {
			logger.Errorf("Ошибка consumer: %v", err)
		}
		if ctx.Err() != nil {
			logger.Info("Consumer остановлен по контексту")
			return
		}
	}

}
