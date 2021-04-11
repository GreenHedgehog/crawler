package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var g *zap.Logger

func init() {
	var err error
	g, err = zap.NewProduction(zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		panic("init global logger failed: " + err.Error())
	}
}

func G() *zap.Logger {
	return g
}
