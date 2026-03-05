package database

import (
	"go.uber.org/zap"
)

type ZapLogger struct {
	l *zap.SugaredLogger
}

type NoopLogger struct{}

func (z ZapLogger) Printf(format string, v ...any) {
	z.l.Infof(format, v...)
}

func (z ZapLogger) Fatalf(format string, v ...any) {
	z.l.Fatalf(format, v...)
}

func (NoopLogger) Printf(string, ...any) {}

func (NoopLogger) Fatalf(string, ...any) {}
