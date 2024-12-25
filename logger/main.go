package logger

import (
	"qdebrid/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var settings = config.GetSettings()

var instance *zap.SugaredLogger

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
		Level:            atomicLevel,
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
	if instance != nil {
		return instance
	}

	logLevel := settings.QDebrid.LogLevel

	instance = initializeLogger(logLevel).Sugar()

	return instance
}
