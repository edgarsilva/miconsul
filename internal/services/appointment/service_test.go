package appointment

import (
	"testing"
	"time"
)

func TestServiceContextHelpers(t *testing.T) {
	svc := &service{}

	t.Run("newWorkerContext has timeout and supports cancellation", func(t *testing.T) {
		ctx, cancel := svc.newWorkerContext()
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected worker context deadline")
		}

		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > defaultWorkerContextTimeout+time.Second {
			t.Fatalf("unexpected worker context timeout window: %v", remaining)
		}

		cancel()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("expected worker context cancellation to propagate")
		}
	})

	t.Run("newCronJobContext has timeout and supports cancellation", func(t *testing.T) {
		ctx, cancel := svc.newCronJobContext()
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected cron context deadline")
		}

		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > defaultCronJobContextTimeout+time.Second {
			t.Fatalf("unexpected cron context timeout window: %v", remaining)
		}

		cancel()
		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("expected cron context cancellation to propagate")
		}
	})
}
