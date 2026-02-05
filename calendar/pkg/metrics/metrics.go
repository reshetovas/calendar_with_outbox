package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	Kafka KafkaMetrics
	API   APIMetrics
	Repo  RepoMetrics
	Go    GoMetrics
}

type KafkaMetrics struct {
	// Producer
	ProducerAttemptLatencySeconds *prometheus.HistogramVec
	ProducerOperationsTotal       *prometheus.CounterVec
	ProducerSuccessAttempts       *prometheus.HistogramVec

	// Consumer
	ConsumerMessagesTotal   *prometheus.CounterVec
	ConsumerProcessDuration *prometheus.HistogramVec
	ConsumerRebalancesTotal *prometheus.CounterVec
	ConsumerInFlight        *prometheus.GaugeVec
}

type APIMetrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
}

type RepoMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	DurationSeconds *prometheus.HistogramVec
	InFlight        *prometheus.GaugeVec
}

type GoMetrics struct {
	InternalGoroutines *prometheus.GaugeVec
}

func New(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		Kafka: KafkaMetrics{
			ProducerAttemptLatencySeconds: f.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "producer_attempt_latency_seconds",
				Help:      "Latency per single produce attempt.",
				Buckets:   prometheus.DefBuckets,
			}, []string{"topic", "result"}), // ok|error

			ProducerOperationsTotal: f.NewCounterVec(prometheus.CounterOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "producer_operations_total",
				Help:      "Total produce operations (one call) by result.",
			}, []string{"topic", "result"}), // success|failed|permanent|canceled

			ProducerSuccessAttempts: f.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "producer_success_attempts",
				Help:      "Attempt number on which produce operation succeeded.",
				Buckets:   []float64{1, 2, 3, 4, 5},
			}, []string{"topic"}),

			ConsumerMessagesTotal: f.NewCounterVec(prometheus.CounterOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "consumer_messages_total",
				Help:      "Total consumed Kafka messages by topic.",
			}, []string{"topic"}),

			ConsumerProcessDuration: f.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "consumer_process_duration_seconds",
				Help:      "Kafka message processing duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			}, []string{"topic"}),

			ConsumerRebalancesTotal: f.NewCounterVec(prometheus.CounterOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "consumer_rebalances_total",
				Help:      "Consumer rebalance lifecycle events.",
			}, []string{"event"}),

			ConsumerInFlight: f.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "payments",
				Subsystem: "kafka",
				Name:      "consumer_inflight_messages",
				Help:      "Messages currently being processed.",
			}, []string{"topic"}),
		},

		API: APIMetrics{
			HTTPRequestsTotal: f.NewCounterVec(prometheus.CounterOpts{
				Namespace: "payments",
				Subsystem: "api",
				Name:      "http_requests_total",
				Help:      "Total HTTP requests by method, path and status.",
			}, []string{"method", "path", "status"}),

			HTTPRequestDuration: f.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "payments",
				Subsystem: "api",
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency.",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			}, []string{"method", "path", "status"}), // статус сюда добавил осознанно
		},
		Repo: RepoMetrics{
			RequestsTotal: f.NewCounterVec(prometheus.CounterOpts{
				Namespace: "payments",
				Subsystem: "db",
				Name:      "requests_total",
				Help:      "Total DB requests by operation, name, result and error kind.",
			}, []string{"op", "name", "result", "error_kind"}),

			DurationSeconds: f.NewHistogramVec(prometheus.HistogramOpts{
				Namespace: "payments",
				Subsystem: "db",
				Name:      "request_duration_seconds",
				Help:      "DB request duration in seconds.",
				// DB обычно быстрее/короче HTTP, но хвосты бывают.
				Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			}, []string{"op", "name", "result"}),

			InFlight: f.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "payments",
				Subsystem: "db",
				Name:      "inflight",
				Help:      "Number of in-flight DB requests.",
			}, []string{"op", "name"}),
		},
		Go: GoMetrics{
			InternalGoroutines: f.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: "payments",
				Subsystem: "go",
				Name:      "internal_goroutines",
				Help:      "Number of running internal goroutines by name.",
			}, []string{"name"}),
		},
	}
}
