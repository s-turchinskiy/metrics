package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger = zap.NewNop().Sugar()

func Initialize() error {

	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"agent.log", "stdout"}
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)

	logger, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("cannot initialize zap")
	}

	Log = logger.Sugar()

	return nil
}
