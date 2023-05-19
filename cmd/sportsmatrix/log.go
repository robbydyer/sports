package main

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (r *rootArgs) getLogger(level zapcore.Level) (*zap.Logger, error) {
	e := zap.NewProductionEncoderConfig()
	e.StacktraceKey = ""

	e.EncodeTime = zapcore.TimeEncoderOfLayout("03:04PM")

	var writer zapcore.WriteSyncer

	if r.logFile != "" {
		var err error
		f, err := os.OpenFile(r.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		r.writer = f
		writer = f
	} else {
		writer = os.Stdout
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(e), writer, level)

	return zap.New(core).WithOptions(
		zap.ErrorOutput(writer),
		zap.WithFatalHook(zapcore.WriteThenFatal),
	), nil
}
