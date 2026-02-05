package listener

import (
	use_cases "calendar/internal/application/use-cases"
	"calendar/pkg/metrics"
	"context"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type KafkaBrokerConsumer struct {
	usecase use_cases.UseCaser
	logger  *zap.SugaredLogger
	m       *metrics.Metrics
}

func NewKafkaBrokerConsumer(usecase use_cases.UseCaser, logger *zap.SugaredLogger, m *metrics.Metrics) *KafkaBrokerConsumer {
	return &KafkaBrokerConsumer{
		logger:  logger,
		usecase: usecase,
		m:       m,
	}
}

func (k *KafkaBrokerConsumer) Setup(session sarama.ConsumerGroupSession) error {
	k.logger.Info("Kafka setup success")
	if k.m != nil {
		k.m.Kafka.ConsumerRebalancesTotal.WithLabelValues("setup").Inc()
	}
	return nil
}

func (k *KafkaBrokerConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
	k.logger.Info("Kafka cleanup success")
	if k.m != nil {
		k.m.Kafka.ConsumerRebalancesTotal.WithLabelValues("cleanup").Inc()
	}
	return nil
}

func (k *KafkaBrokerConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	topic := claim.Topic()

	for msg := range claim.Messages() {
		if k.m != nil {
			k.m.Kafka.ConsumerInFlight.WithLabelValues(topic).Inc()
		}
		start := time.Now()
		k.logger.Infof("Message topic:%q partition:%d offset:%d  value:%s", msg.Topic, msg.Partition, msg.Offset, msg.Value)

		k.usecase.ConsumerMessage(context.Background(), msg.Value, start)
		if k.m != nil {
			k.m.Kafka.ConsumerMessagesTotal.WithLabelValues(topic).Inc()
			k.m.Kafka.ConsumerProcessDuration.WithLabelValues(topic).Observe(time.Since(start).Seconds())
			k.m.Kafka.ConsumerInFlight.WithLabelValues(topic).Dec()
		}

		session.MarkMessage(msg, "")
	}

	return nil
}
