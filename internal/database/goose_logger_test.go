package database

import (
	"testing"

	lognoop "go.opentelemetry.io/otel/log/noop"
)

func TestGooseLoggerAdapters(t *testing.T) {
	adapter := NewGooseLogger(lognoop.Logger{})
	adapter.Printf("hello %s", "db")
	adapter.Fatalf("fatal %s", "db")

	NoopGooseLogger{}.Printf("ignored")
	NoopGooseLogger{}.Fatalf("ignored")
}
