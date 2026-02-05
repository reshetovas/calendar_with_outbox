package observability

import (
	"log"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(level string) *zap.SugaredLogger {
	logConfig := zap.NewProductionConfig()
	logConfig.Sampling = nil
	logConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	logConfig.DisableStacktrace = true

	logConfig.Level = zap.NewAtomicLevelAt(DetermineLogLevel(level))

	logger, err := logConfig.Build()
	if err != nil {
		log.Fatal(err)
	}

	return logger.Sugar()
}

func DetermineLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}
