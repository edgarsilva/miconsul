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
	ErrJobTypeRequired    = errors.New("job type is required")
)

type Option = asynq.Option

// JobInfo describes an enqueued job. It projects the asynq task metadata into
// our own fields so callers never depend on asynq types.
type JobInfo struct {
	ID    string
	Type  JobType
	Queue string
	State string
}

func newJobInfo(info *asynq.TaskInfo) JobInfo {
	if info == nil {
		return JobInfo{}
	}
	return JobInfo{
		ID:    info.ID,
		Type:  JobType(info.Type),
		Queue: info.Queue,
		State: info.State.String(),
	}
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

func (r *Runtime) EnqueueJob(ctx context.Context, jobType JobType, payload any, opts ...Option) (JobInfo, error) {
	if r == nil {
		return JobInfo{}, ErrRuntimeUnavailable
	}
	if !r.enabled {
		return JobInfo{}, ErrRuntimeDisabled
	}
	if r.client == nil {
		return JobInfo{}, ErrRuntimeUnavailable
	}

	jobType = JobType(strings.TrimSpace(jobType.String()))
	if jobType == "" {
		return JobInfo{}, ErrJobTypeRequired
	}

	task, err := newTask(jobType, payload, opts...)
	if err != nil {
		return JobInfo{}, err
	}

	info, err := r.client.EnqueueContext(ctx, task)
	if err != nil {
		return JobInfo{}, fmt.Errorf("enqueue %s: %w", jobType, err)
	}

	return newJobInfo(info), nil
}

func newTask(jobType JobType, payload any, opts ...Option) (*asynq.Task, error) {
	taskType := strings.TrimSpace(jobType.String())
	if taskType == "" {
		return nil, ErrJobTypeRequired
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
