package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
)

func TestNewTask(t *testing.T) {
	t.Run("returns error when task type is missing", func(t *testing.T) {
		task, err := newTask("", map[string]any{"id": "a1"})
		if err == nil {
			t.Fatal("newTask() expected error")
		}
		if !errors.Is(err, ErrTaskTypeRequired) {
			t.Fatalf("newTask() error = %v, want %v", err, ErrTaskTypeRequired)
		}
		if task != nil {
			t.Fatal("newTask() expected nil task")
		}
	})

	t.Run("builds task with payload", func(t *testing.T) {
		task, err := newTask("appointment:booked_alert", map[string]any{"appointment_id": "a1"})
		if err != nil {
			t.Fatalf("newTask() unexpected error: %v", err)
		}
		if got := task.Type(); got != "appointment:booked_alert" {
			t.Fatalf("newTask() type = %q, want %q", got, "appointment:booked_alert")
		}
	})
}

func TestRuntimeEnqueueGuards(t *testing.T) {
	t.Run("returns unavailable when runtime is nil", func(t *testing.T) {
		var runtime *Runtime
		_, err := runtime.EnqueueTask(context.Background(), "appointment:booked_alert", map[string]any{"appointment_id": "a1"})
		if !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("enqueue error = %v, want %v", err, ErrRuntimeUnavailable)
		}
	})

	t.Run("returns disabled when runtime is disabled", func(t *testing.T) {
		runtime := &Runtime{}
		_, err := runtime.EnqueueTask(context.Background(), "appointment:booked_alert", map[string]any{"appointment_id": "a1"})
		if !errors.Is(err, ErrRuntimeDisabled) {
			t.Fatalf("enqueue error = %v, want %v", err, ErrRuntimeDisabled)
		}
	})

	t.Run("returns task type required", func(t *testing.T) {
		runtime := &Runtime{enabled: true, client: asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"})}
		t.Cleanup(func() { _ = runtime.client.Close() })

		_, err := runtime.EnqueueTask(context.Background(), "", map[string]any{"id": "a1"})
		if !errors.Is(err, ErrTaskTypeRequired) {
			t.Fatalf("enqueue error = %v, want %v", err, ErrTaskTypeRequired)
		}
	})
}
