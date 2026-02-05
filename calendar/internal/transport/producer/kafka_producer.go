package producer

import (
	"calendar/internal/application/common"
	"calendar/pkg/broker"
	"calendar/pkg/metrics"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Producer interface {
	ProduceMessage(ctx context.Context, ID int, message []byte) error
	HealthCheck(ctx context.Context) error
}

type KafkaProducerConfig struct {
	broker      *broker.KafkaBroker
	logger      *zap.SugaredLogger
	maxAttempts int
	m           *metrics.Metrics
}

func NewProducer(broker *broker.KafkaBroker, logger *zap.SugaredLogger, maxAttempts int, m *metrics.Metrics) *KafkaProducerConfig {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	return &KafkaProducerConfig{
		broker:      broker,
		logger:      logger,
		maxAttempts: maxAttempts,
		m:           m,
	}
}

// HealthCheck проверяет доступность Kafka через broker
func (p *KafkaProducerConfig) HealthCheck(ctx context.Context) error {
	if p.broker == nil {
		return errors.New("kafka broker is not initialized")
	}

	// Используем метод HealthCheck из KafkaBroker для реальной проверки
	return p.broker.HealthCheck(ctx)
}

func (p *KafkaProducerConfig) ProduceMessage(ctx context.Context, id int, message []byte) error {
	topic := p.broker.ProducerTopic
	var lastErr error

	for attempt := 1; attempt <= p.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		msg := &sarama.ProducerMessage{
			Topic:     topic,
			Key:       sarama.StringEncoder(strconv.FormatInt(int64(id), 10)),
			Value:     sarama.ByteEncoder(message),
			Timestamp: time.Now(),
		}

		t0 := time.Now()
		part, off, err := p.broker.SyncProducer.SendMessage(msg)
		rt := time.Since(t0)

		//Metric: attempt latency: ok/error
		if p.m != nil {
			res := "ok"
			if err != nil {
				res = "error"
			}
			p.m.Kafka.ProducerAttemptLatencySeconds.WithLabelValues(topic, res).Observe(rt.Seconds())
		}

		if err == nil {
			// Metric success
			if p.m != nil {
				p.m.Kafka.ProducerOperationsTotal.WithLabelValues(topic, "success").Inc()
				p.m.Kafka.ProducerSuccessAttempts.WithLabelValues(topic).Observe(float64(attempt))
			}
			p.logger.Infof("[transactionID %s] sent topic=%s partition=%d offset=%d attempt=%d rt=%s",
				id, p.broker.ProducerTopic, part, off, attempt, rt)
			return nil
		}

		lastErr = err

		if kerr, ok := err.(sarama.KError); ok {
			if isPermanent(kerr) {
				if p.m != nil {
					p.m.Kafka.ProducerOperationsTotal.WithLabelValues(topic, "permanent").Inc()
				}
				p.logger.Errorf("[transactionID %s] permanent kafka error attempt=%d rt=%s kafka_error=%s code=%d", id, attempt, rt, kerr.Error(), int16(kerr))
				return fmt.Errorf("permanent kafka error: %w", kerr)
			}

			p.logger.Warnf("[transactionID %s] retryable kafka error attempt=%d rt=%s kafka_error=%s code=%d",
				id, attempt, rt, kerr.Error(), int16(kerr))
		} else {
			p.logger.Warnf("[transactionID %s] retryable non-kafka error attempt=%d rt=%s err=%v",
				id, attempt, rt, err)
		}

		if attempt == p.maxAttempts {
			break
		}

		if err := common.SleepCtx(ctx, common.NextBackoffWithJitter(attempt-1)); err != nil {
			// отмена/таймаут контекста считаем как canceled
			if p.m != nil {
				p.m.Kafka.ProducerOperationsTotal.WithLabelValues(topic, "canceled").Inc()
			}
			return err
		}
	}
	p.logger.Errorf("[transactionID %s] produce_failed after %d attempts: %v", id, p.maxAttempts, lastErr)
	return fmt.Errorf("produce failed after %d attempts: %w", p.maxAttempts, lastErr)
}

func isPermanent(k sarama.KError) bool {
	switch k {
	case sarama.ErrTopicAuthorizationFailed,
		sarama.ErrClusterAuthorizationFailed,
		sarama.ErrInvalidRequest,
		sarama.ErrInvalidMessage,
		sarama.ErrMessageSizeTooLarge,
		sarama.ErrSASLAuthenticationFailed:
		return true
	default:
		return false
	}
}

func ClassifyRetry(err error) string {
	if k, ok := err.(sarama.KError); ok {
		switch k {
		case sarama.ErrLeaderNotAvailable:
			return "leader_not_available"
		case sarama.ErrRequestTimedOut:
			return "broker_timeout"
		case sarama.ErrNotEnoughReplicas, sarama.ErrNotEnoughReplicasAfterAppend:
			return "not_enough_replicas"
		default:
			return k.Error()
		}
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return "net_timeout"
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "client_deadline"
	}
	return "other"
}
