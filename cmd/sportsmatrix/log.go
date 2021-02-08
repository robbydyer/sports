package main

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getLogger(level zapcore.Level) *zap.Logger {
	e := zap.NewProductionEncoderConfig()
	e.StacktraceKey = ""

	e.EncodeTime = zapcore.TimeEncoderOfLayout("03:04PM")

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(e), os.Stdout, level)

	return zap.New(core).WithOptions(zap.ErrorOutput(os.Stdout))
}
