package zap_wrapper

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewDevSugaredLogger() (logger *zap.SugaredLogger, closeFunc func() error, err error) {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logConfig := zap.NewDevelopmentConfig()
	logConfig.EncoderConfig = encoderConfig

	baseLogger, err := logConfig.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("can't initialize zap logger: %v", err)
	}

	closeFunc = func() error {
		if err = baseLogger.Sync(); err != nil {
			return fmt.Errorf("can't flush log entities: %v", err)
		}

		return nil
	}

	return baseLogger.Sugar(), closeFunc, nil
}
