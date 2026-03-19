package server

import (
	"context"
	"errors"
	"testing"

	"miconsul/internal/jobs"
)

func TestEnqueueTask(t *testing.T) {
	s := &Server{}

	_, err := s.EnqueueTask(context.Background(), "appointment:booked_alert", map[string]any{"appointment_id": "a1"})
	if !errors.Is(err, jobs.ErrRuntimeUnavailable) {
		t.Fatalf("enqueue error = %v, want %v", err, jobs.ErrRuntimeUnavailable)
	}
}
