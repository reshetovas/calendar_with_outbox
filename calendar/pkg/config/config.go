package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       Server      `mapstructure:"server"`
	Postgres     Postgres    `mapstructure:"postgres"`
	Broker       Broker      `mapstructure:"broker"`
	Cron         Cron        `mapstructure:"cron"`
	Realay       RelayConfig `mapstructure:"relay"`
	HTTPClient   HTTPClient  `mapstructure:"httpClient"`
	LoggingLevel string      `mapstructure:"logging-level"`
}

type Server struct {
	Port          string `mapstructure:"port"`
	SwaggerUrl    string `mapstructure:"swagger_json"`
	SwaggerHost   string `mapstructure:"swagger_host"`
	SwaggerSchema string `mapstructure:"swagger_schema"`
	BodyLimit     int    `mapstructure:"body_limit"`
}

type Postgres struct {
	ConnString     string `mapstructure:"conn_string"`
	MaxConnections int32  `mapstructure:"max_connections"`
}

type Broker struct {
	Kafka Kafka `mapstructure:"kafka"`
}

type Kafka struct {
	Brokers      string `mapstructure:"brokers"`
	ReaderTopic  string `mapstructure:"readerTopic"`
	ReaderUsr    string `mapstructure:"readerUsr"`
	ReaderUsrPwd string `mapstructure:"readerUsrPwd"`
	WriterTopic  string `mapstructure:"writerTopic"`
	WriterUsr    string `mapstructure:"writerUsr"`
	WriterUsrPwd string `mapstructure:"writerUsrPwd"`
	MaxAttempts  int    `mapstructure:"maxAttempts"`
}

type Cron struct {
	DaysToDelete int    `mapstructure:"daysToDelete"` // Количество дней для удаления старых событий
	Schedule     string `mapstructure:"schedule"`     // Расписание в формате cron (например, "0 16 * * *" - каждый день в 16:00)
	Interval     string `mapstructure:"interval"`     // Интервал в формате "@every 1m" (например, "@every 1m" - каждую минуту)
	// Приоритет: если указан Schedule, используется он, иначе Interval
}

type RelayConfig struct {
	Workers     int           `mapstructure:"workers"`
	BatchSize   int           `mapstructure:"batchSize"`
	Lease       time.Duration `mapstructure:"lease"`
	PollPeriod  time.Duration `mapstructure:"pollPeriod"`
	MaxAttempts int           `mapstructure:"maxAttempts"`
}

type HTTPClient struct {
	//адреса
	BConnectExtStateURL     string `mapstructure:"bConnectExtStatePath"`
	BConnectCheckBalanceURL string `mapstructure:"bConnectCheckBalanceURL"`

	//конфиг клиента
	ConnectTimeout        time.Duration `mapstructure:"connectTimeout"`        // TCP коннект
	TLSHandshakeTimeout   time.Duration `mapstructure:"TLSHandshakeTimeout"`   // TLS рукопожатие
	ResponseHeaderTimeout time.Duration `mapstructure:"responseHeaderTimeout"` // ожидание заголовков ответа
	ExpectContinueTimeout time.Duration `mapstructure:"expectContinueTimeout"` // 100-continue

	// Пул соединений
	IdleConnTimeout     time.Duration `mapstructure:"idleConnTimeout"`
	MaxIdleConns        int           `mapstructure:"maxIdleConns"`
	MaxIdleConnsPerHost int           `mapstructure:"maxIdleConnsPerHost"`
	MaxConnsPerHost     int           `mapstructure:"maxConnsPerHost"`
	KeepAlives          bool          `mapstructure:"keepAlives"`

	// Общий таймаут клиента. 0 — контролируем дедлайном через context.
	ClientTimeout time.Duration `mapstructure:"clientTimeout"`

	// Прочее
	UserAgent  string `mapstructure:"userAgent"`
	MaxRetries int    `mapstructure:"maxRetries"`

	// SSL/TLS настройки
	InsecureSkipVerify bool `mapstructure:"insecureSkipVerify"` // отключить проверку SSL сертификатов
}

func NewConfig() (Config, error) {
	viper.AutomaticEnv()
	// Настраиваем замену точек и дефисов на подчеркивания для переменных окружения
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	var conf Config
	err := viper.ReadInConfig() // Find and read the config file
	// Игнорируем ошибку, если файл не найден - используем только переменные окружения
	if err != nil {
		// Если это не ошибка "файл не найден", возвращаем её
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return conf, err
		}
	}

	// unmarshal
	err = viper.Unmarshal(&conf)

	return conf, err
}
