package logger

import (
	"fmt"
	"qdebrid/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var settings = config.GetSettings()

func initializeLogger(level string) *zap.Logger {
	var atomicLevel zap.AtomicLevel

	switch level {
	case "debug":
		atomicLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		atomicLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	default:
		atomicLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:       atomicLevel,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger
}



func Sugar() *zap.SugaredLogger {
	logLevel := settings.QDebrid.LogLevel

	return initializeLogger(logLevel).Sugar()
}

func EndpointMessage(module string, endpoint string, message string) string {
	return fmt.Sprintf("[%s] %s: %s", module, endpoint, message)
}
