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

type EnqueueInfo struct {
	*asynq.TaskInfo
}

func (r *Runtime) EnqueueTask(ctx context.Context, taskType string, payload any, opts ...asynq.Option) (EnqueueInfo, error) {
	if r == nil {
		return EnqueueInfo{}, ErrRuntimeUnavailable
	}
	if !r.enabled {
		return EnqueueInfo{}, ErrRuntimeDisabled
	}
	if r.client == nil {
		return EnqueueInfo{}, ErrRuntimeUnavailable
	}

	taskType = strings.TrimSpace(taskType)
	if taskType == "" {
		return EnqueueInfo{}, ErrTaskTypeRequired
	}

	task, err := newTask(taskType, payload, opts...)
	if err != nil {
		return EnqueueInfo{}, err
	}

	info, err := r.client.EnqueueContext(ctx, task)
	if err != nil {
		return EnqueueInfo{}, fmt.Errorf("enqueue %s: %w", taskType, err)
	}

	return EnqueueInfo{TaskInfo: info}, nil
}

func newTask(taskType string, payload any, opts ...asynq.Option) (*asynq.Task, error) {
	if strings.TrimSpace(taskType) == "" {
		return nil, ErrTaskTypeRequired
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}

	allOpts := append(defaultTaskOptions(), opts...)
	return asynq.NewTask(taskType, payloadJSON, allOpts...), nil
}

func defaultTaskOptions() []asynq.Option {
	return []asynq.Option{
		asynq.Queue("default"),
		asynq.MaxRetry(10),
		asynq.Timeout(30 * time.Second),
	}
}
