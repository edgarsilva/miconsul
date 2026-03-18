package jobs

import (
	"context"
	"errors"
	"testing"

	"miconsul/internal/lib/appenv"

	"github.com/hibiken/asynq"
)

func TestNew(t *testing.T) {
	t.Run("returns error for nil env", func(t *testing.T) {
		runtime, err := New(nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if runtime != nil {
			t.Fatal("expected nil runtime")
		}
	})

	t.Run("returns disabled runtime when jobs are disabled", func(t *testing.T) {
		runtime, err := New(&appenv.Env{JobsEnabled: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runtime == nil {
			t.Fatal("expected runtime")
		}
		if runtime.Enabled() {
			t.Fatal("expected runtime disabled")
		}
		if err := runtime.Shutdown(); err != nil {
			t.Fatalf("unexpected shutdown error: %v", err)
		}
	})
}

func TestRegisterTaskHandlerGuards(t *testing.T) {
	t.Run("returns unavailable when runtime disabled", func(t *testing.T) {
		runtime := &Runtime{}
		err := runtime.RegisterTaskHandler("appointment:booked_alert", func(context.Context, Task) error { return nil })
		if !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("register task handler error = %v, want %v", err, ErrRuntimeUnavailable)
		}
	})

	t.Run("returns task type required when missing", func(t *testing.T) {
		runtime := &Runtime{enabled: true, mux: asynq.NewServeMux()}
		err := runtime.RegisterTaskHandler("", func(context.Context, Task) error { return nil })
		if !errors.Is(err, ErrTaskTypeRequired) {
			t.Fatalf("register task handler error = %v, want %v", err, ErrTaskTypeRequired)
		}
	})

	t.Run("returns handler required when nil", func(t *testing.T) {
		runtime := &Runtime{enabled: true, mux: asynq.NewServeMux()}
		err := runtime.RegisterTaskHandler("appointment:booked_alert", nil)
		if !errors.Is(err, ErrHandlerRequired) {
			t.Fatalf("register task handler error = %v, want %v", err, ErrHandlerRequired)
		}
	})

	t.Run("skips duplicate task handler registrations", func(t *testing.T) {
		runtime := &Runtime{enabled: true, mux: asynq.NewServeMux(), registeredHandlers: map[string]struct{}{}}
		handler := HandlerFunc(func(context.Context, Task) error { return nil })
		if err := runtime.RegisterTaskHandler("appointment:booked_alert", handler); err != nil {
			t.Fatalf("first register task handler error: %v", err)
		}
		if err := runtime.RegisterTaskHandler("appointment:booked_alert", handler); err != nil {
			t.Fatalf("duplicate register task handler error: %v", err)
		}
	})
}

func TestRegisterScheduledTaskGuards(t *testing.T) {
	t.Run("returns unavailable when runtime disabled", func(t *testing.T) {
		runtime := &Runtime{}
		_, err := runtime.RegisterScheduledTask("@every 1m", "appointment:reminder_sweep", map[string]any{})
		if !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("register scheduled task error = %v, want %v", err, ErrRuntimeUnavailable)
		}
	})

	t.Run("returns spec required when missing", func(t *testing.T) {
		runtime := &Runtime{enabled: true, scheduler: asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"}, &asynq.SchedulerOpts{})}
		t.Cleanup(runtime.scheduler.Shutdown)

		_, err := runtime.RegisterScheduledTask("", "appointment:reminder_sweep", map[string]any{})
		if !errors.Is(err, ErrScheduleSpecMissing) {
			t.Fatalf("register scheduled task error = %v, want %v", err, ErrScheduleSpecMissing)
		}
	})

	t.Run("skips duplicate schedule registrations", func(t *testing.T) {
		registrationKey := scheduledTaskRegistrationKey("@every 1m", "appointment:reminder_sweep")
		runtime := &Runtime{
			enabled:             true,
			scheduler:           &asynq.Scheduler{},
			registeredSchedules: map[string]string{registrationKey: "entry-1"},
		}

		entryID, err := runtime.RegisterScheduledTask("@every 1m", "appointment:reminder_sweep", map[string]any{})
		if err != nil {
			t.Fatalf("duplicate register scheduled task error: %v", err)
		}
		if entryID != "entry-1" {
			t.Fatalf("entryID = %q, want %q", entryID, "entry-1")
		}
	})
}
