// Package jobs provides the jobs runtime that wraps asynq library.
package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

var (
	ErrRuntimeUnavailable = errors.New("jobs runtime is unavailable")
	ErrRuntimeDisabled    = errors.New("jobs runtime is disabled")
	ErrTaskTypeRequired   = errors.New("task type is required")
)

type Option = asynq.Option

type EnqueueInfo struct {
	*asynq.TaskInfo
}

// JobType identifies a kind of job (the asynq task type). It is the routing
// key that maps an enqueued job to its registered handler.
type JobType string

func (t JobType) String() string { return string(t) }

type Job struct {
	Type       JobType
	Payload    []byte
	ID         string
	Queue      string
	RetryCount int
	MaxRetry   int
}

type Handler func(ctx context.Context, job Job) error

type asynqHandlerAdapter struct {
	fn Handler
}

func newJobHandler(handler Handler) asynqHandlerAdapter {
	return asynqHandlerAdapter{fn: handler}
}

func (a asynqHandlerAdapter) ProcessTask(ctx context.Context, task *asynq.Task) error {
	t := Job{
		Type:    JobType(task.Type()),
		Payload: task.Payload(),
	}
	if id, ok := asynq.GetTaskID(ctx); ok {
		t.ID = id
	}
	if queue, ok := asynq.GetQueueName(ctx); ok {
		t.Queue = queue
	}
	if retryCount, ok := asynq.GetRetryCount(ctx); ok {
		t.RetryCount = retryCount
	}
	if maxRetry, ok := asynq.GetMaxRetry(ctx); ok {
		t.MaxRetry = maxRetry
	}

	return a.fn(ctx, t)
}

func (r *Runtime) EnqueueTask(ctx context.Context, jobType JobType, payload any, opts ...Option) (EnqueueInfo, error) {
	if r == nil {
		return EnqueueInfo{}, ErrRuntimeUnavailable
	}
	if !r.enabled {
		return EnqueueInfo{}, ErrRuntimeDisabled
	}
	if r.client == nil {
		return EnqueueInfo{}, ErrRuntimeUnavailable
	}

	jobType = JobType(strings.TrimSpace(jobType.String()))
	if jobType == "" {
		return EnqueueInfo{}, ErrTaskTypeRequired
	}

	task, err := newTask(jobType, payload, opts...)
	if err != nil {
		return EnqueueInfo{}, err
	}

	info, err := r.client.EnqueueContext(ctx, task)
	if err != nil {
		return EnqueueInfo{}, fmt.Errorf("enqueue %s: %w", jobType, err)
	}

	return EnqueueInfo{TaskInfo: info}, nil
}

func newTask(jobType JobType, payload any, opts ...Option) (*asynq.Task, error) {
	taskType := strings.TrimSpace(jobType.String())
	if taskType == "" {
		return nil, ErrTaskTypeRequired
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}

	allOpts := append(defaultTaskOptions(), opts...)
	return asynq.NewTask(taskType, payloadJSON, allOpts...), nil
}

func defaultTaskOptions() []Option {
	return []Option{
		asynq.Queue("default"),
		asynq.MaxRetry(10),
		asynq.Timeout(30 * time.Second),
	}
}
