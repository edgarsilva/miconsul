package jobs

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
)

func TestRuntimeKeepsScheduledTasksAcrossRestart(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	host, portStr, err := net.SplitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("split miniredis address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse miniredis port: %v", err)
	}

	env := &appenv.Env{
		JobsEnabled: true,
		ValkeyHost:  host,
		ValkeyPort:  port,
		ValkeyDB:    0,
	}

	runtime1, err := New(env)
	if err != nil {
		t.Fatalf("start first jobs runtime: %v", err)
	}

	const taskType = "appointment:restart_probe"
	_, err = runtime1.EnqueueTask(
		context.Background(),
		taskType,
		map[string]any{"appointment_id": "apnt_restart_probe"},
		asynq.ProcessIn(10*time.Minute),
	)
	if err != nil {
		t.Fatalf("enqueue scheduled task: %v", err)
	}

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: mr.Addr()})
	t.Cleanup(func() {
		if closeErr := inspector.Close(); closeErr != nil {
			t.Fatalf("close inspector: %v", closeErr)
		}
	})

	beforeRestart := mustCountScheduledTasksByType(t, inspector, "default", taskType)
	if beforeRestart == 0 {
		t.Fatalf("expected scheduled task %q before restart", taskType)
	}

	if err := runtime1.Shutdown(); err != nil {
		t.Fatalf("shutdown first jobs runtime: %v", err)
	}

	runtime2, err := New(env)
	if err != nil {
		t.Fatalf("start second jobs runtime: %v", err)
	}
	t.Cleanup(func() {
		if shutdownErr := runtime2.Shutdown(); shutdownErr != nil {
			t.Fatalf("shutdown second jobs runtime: %v", shutdownErr)
		}
	})

	afterRestart := mustCountScheduledTasksByType(t, inspector, "default", taskType)
	if afterRestart < beforeRestart {
		t.Fatalf("scheduled task count dropped after restart: before=%d after=%d", beforeRestart, afterRestart)
	}
}

func mustCountScheduledTasksByType(t *testing.T, inspector *asynq.Inspector, queue, taskType string) int {
	t.Helper()

	tasks, err := inspector.ListScheduledTasks(queue, asynq.Page(1), asynq.PageSize(200))
	if err != nil {
		t.Fatalf("list scheduled tasks: %v", err)
	}

	count := 0
	for _, task := range tasks {
		if task != nil && task.Type == taskType {
			count++
		}
	}

	return count
}
