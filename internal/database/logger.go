package database

import (
	"context"
	"fmt"
	"log"
	"os"

	obslogging "miconsul/internal/observability/logging"

	otellog "go.opentelemetry.io/otel/log"
)

type GooseLogger struct {
	stdout    *log.Logger
	obsLogger obslogging.Logger
}

type NoopLogger struct{}

func NewGooseLogger(obsLogger obslogging.Logger) GooseLogger {
	return GooseLogger{
		stdout:    log.New(os.Stdout, "", log.LstdFlags),
		obsLogger: obsLogger,
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
	rec.SetBody(otellog.StringValue(msg))
	rec.SetSeverity(severity)
	rec.SetSeverityText(severityText)
	rec.AddAttributes(
		otellog.String("event", "db_migration"),
		otellog.String("component", "goose"),
		otellog.String("message", msg),
	)

	l.obsLogger.Emit(context.Background(), rec)
}

func (NoopLogger) Printf(string, ...any) {}

func (NoopLogger) Fatalf(string, ...any) {}
