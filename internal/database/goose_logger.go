package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
)

type GooseLogger struct {
	stdout     *log.Logger
	otelLogger otellog.Logger
}

type NoopGooseLogger struct{}

func NewGooseLogger(otelLogger otellog.Logger) GooseLogger {
	return GooseLogger{
		stdout:     log.New(os.Stdout, "", log.LstdFlags),
		otelLogger: otelLogger,
	}
}

func (l GooseLogger) Printf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.stdout.Printf("goose: %s", msg)
	l.emit(otellog.SeverityInfo, "INFO", msg)
}

func (l GooseLogger) Fatalf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.stdout.Printf("goose: %s", msg)
	l.emit(otellog.SeverityError, "ERROR", msg)
}

func (l GooseLogger) emit(severity otellog.Severity, severityText, msg string) {
	rec := otellog.Record{}
	rec.SetEventName("db_migration")
	rec.SetBody(attribute.StringValue(msg))
	rec.SetSeverity(severity)
	rec.SetSeverityText(severityText)
	rec.AddAttributes(
		attribute.String("event", "db_migration"),
		attribute.String("component", "goose"),
		attribute.String("message", msg),
	)

	l.otelLogger.Emit(context.Background(), rec)
}

func (NoopGooseLogger) Printf(string, ...any) {}

func (NoopGooseLogger) Fatalf(string, ...any) {}
