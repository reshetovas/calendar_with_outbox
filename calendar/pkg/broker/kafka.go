package broker

import (
	"calendar/pkg/config"
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/IBM/sarama"
)

const (
	_consumerGroup = "consumer-group"
)

type KafkaBroker struct {
	ConsumerTopic string
	ProducerTopic string
	ConsumerGroup sarama.ConsumerGroup
	SyncProducer  sarama.SyncProducer
	Brokers       []string
	conf          config.Kafka
	logger        *zap.SugaredLogger
}

func NewKafkaBroker(conf config.Kafka, logger *zap.SugaredLogger) (*KafkaBroker, error) {
	logger.Debugf("Создание consumer group для brokers: %s\n", conf.Brokers)
	consumerGroup, err := newConsumerGroup(conf)
	if err != nil {
		logger.Errorf("Ошибка создания consumer group: %v\n", err)
		return nil, fmt.Errorf("%w", err)
	}
	logger.Infof("Consumer group создан успешно\n")

	logger.Debugf("Создание producer для brokers: %s\n", conf.Brokers)
	syncProducer, err := newSyncProducer(conf)
	if err != nil {
		logger.Errorf("Ошибка создания producer: %v\n", err)
		return nil, fmt.Errorf("%w", err)
	}
	logger.Infof("Producer создан успешно\n")

	brokers := strings.Split(conf.Brokers, ",")
	broker := &KafkaBroker{
		ConsumerTopic: conf.ReaderTopic,
		ProducerTopic: conf.WriterTopic,
		ConsumerGroup: consumerGroup,
		SyncProducer:  syncProducer,
		Brokers:       brokers,
		conf:          conf,
		logger:        logger,
	}
	logger.Infof("KafkaBroker создан. Consumer topic: %s, Producer topic: %s\n", broker.ConsumerTopic, broker.ProducerTopic)
	return broker, nil
}

// HealthCheck проверяет доступность Kafka брокера, Producer и ConsumerGroup
//
// Важно: НЕ использует client.Partitions(), так как это требует операции Describe в ACL.
// Если на стенде настроены ограничения (например, consumer ТУЗ может только Write,
// а producer ТУЗ может только Read), то проверка Partitions() сломается.
//
// Вместо этого проверяем:
// 1. Инициализацию SyncProducer и ConsumerGroup (если они созданы - значит права есть)
// 2. Доступность брокеров через минимальный клиент (не требует Describe)
//
// Если продюсер и консьюмер успешно созданы, значит они имеют необходимые права
// для работы (Write для producer, Read для consumer). Проверка брокеров подтверждает
// доступность Kafka кластера.
func (kb *KafkaBroker) HealthCheck(ctx context.Context) error {
	// Проверяем, что Producer инициализирован
	// Если SyncProducer создан успешно - значит у него есть права Write на producer topic
	if kb.SyncProducer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	// Проверяем, что ConsumerGroup инициализирован
	// Если ConsumerGroup создан успешно - значит у него есть права Read на consumer topic
	if kb.ConsumerGroup == nil {
		return fmt.Errorf("kafka consumer group is not initialized")
	}

	// Проверяем доступность брокеров через минимальный клиент
	// Это не требует Describe прав, только базовое подключение
	cfg := sarama.NewConfig()
	cfg.Net.DialTimeout = 2 * time.Second
	cfg.Net.ReadTimeout = 2 * time.Second
	cfg.Net.WriteTimeout = 2 * time.Second
	cfg.Metadata.Timeout = 2 * time.Second
	cfg.Metadata.Retry.Max = 1

	// Применяем те же настройки SASL, что и в producer (приоритет Writer credentials)
	if kb.conf.WriterUsr != "" && kb.conf.WriterUsrPwd != "" {
		applySASLConfig(cfg, kb.conf, true)
	} else {
		applySASLConfig(cfg, kb.conf, false)
	}

	client, err := sarama.NewClient(kb.Brokers, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to kafka brokers: %w", err)
	}
	defer client.Close()

	// Проверяем доступность брокеров (это не требует Describe прав)
	brokers := client.Brokers()
	if len(brokers) == 0 {
		return fmt.Errorf("no kafka brokers available")
	}

	return nil
}

// applySASLConfig применяет SASL конфигурацию к sarama.Config
// useWriterCreds: true - использует WriterUsr/WriterUsrPwd, false - ReaderUsr/ReaderUsrPwd
func applySASLConfig(cfg *sarama.Config, conf config.Kafka, useWriterCreds bool) {
	if useWriterCreds {
		if conf.WriterUsr != "" && conf.WriterUsrPwd != "" {
			cfg.Net.SASL.User = conf.WriterUsr
			cfg.Net.SASL.Password = conf.WriterUsrPwd
			cfg.Net.SASL.Enable = true
			cfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	} else {
		if conf.ReaderUsr != "" && conf.ReaderUsrPwd != "" {
			cfg.Net.SASL.User = conf.ReaderUsr
			cfg.Net.SASL.Password = conf.ReaderUsrPwd
			cfg.Net.SASL.Enable = true
			cfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}
}

func EnableSaramaZapLogs(base *zap.SugaredLogger) {
	logger := base.Named("sarama")
	sarama.Logger = &zapSarama{logger}
	logger.Info("🔧 Sarama logger initialized")
	sarama.Logger.Print("🔧 Test message from Sarama logger")
}

type zapSarama struct{ l *zap.SugaredLogger }

func (z *zapSarama) Print(v ...interface{})                 { z.l.Debug(v...) }
func (z *zapSarama) Printf(format string, v ...interface{}) { z.l.Debugf(format, v...) }
func (z *zapSarama) Println(v ...interface{})               { z.l.Debug(v...) }

func newConsumerGroup(conf config.Kafka) (sarama.ConsumerGroup, error) {
	kafkaConfig := sarama.NewConfig()
	applySASLConfig(kafkaConfig, conf, false) // используем Reader credentials

	brokers := strings.Split(conf.Brokers, ",")

	consumer, err := sarama.NewConsumerGroup(brokers, _consumerGroup, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании Kafka Consumer Group: %w", err)
	}

	return consumer, nil
}

func newSyncProducer(conf config.Kafka) (sarama.SyncProducer, error) {
	kafkaConfig := sarama.NewConfig()

	kafkaConfig.Net.DialTimeout = 10 * time.Second
	kafkaConfig.Net.ReadTimeout = 15 * time.Second
	kafkaConfig.Net.WriteTimeout = 15 * time.Second
	kafkaConfig.Net.KeepAlive = 30 * time.Second

	kafkaConfig.Metadata.Timeout = 10 * time.Second
	kafkaConfig.Metadata.Retry.Max = 1
	kafkaConfig.Metadata.Retry.Backoff = 1 * time.Second
	kafkaConfig.Metadata.RefreshFrequency = 1 * time.Minute

	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Retry.Max = 0
	kafkaConfig.Producer.Timeout = 10 * time.Second
	kafkaConfig.Producer.Partitioner = sarama.NewHashPartitioner

	applySASLConfig(kafkaConfig, conf, true) // используем Writer credentials

	brokers := strings.Split(conf.Brokers, ",")

	producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании Kafka Sync Producer: %w", err)
	}

	return producer, nil
}
