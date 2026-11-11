package appointment

import (
	"slices"
	"testing"
	"time"
)

func TestAppointmentUpdatesColumns(t *testing.T) {
	if slices.Contains(appointmentUpdates{}.columns(), "RescheduledAt") {
		t.Fatalf("RescheduledAt must not be written when zero")
	}
	if !slices.Contains(appointmentUpdates{RescheduledAt: time.Now()}.columns(), "RescheduledAt") {
		t.Fatalf("RescheduledAt must be written when set")
	}
}
