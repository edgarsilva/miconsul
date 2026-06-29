package server

import (
	"context"

	"miconsul/internal/jobs"
)

// JobsRuntime returns the active background jobs runtime when available.
func (s *Server) JobsRuntime() *jobs.Runtime {
	if s == nil {
		return nil
	}

	return s.jobs
}

// EnqueueJob enqueues a background task through the jobs runtime.
func (s *Server) EnqueueJob(ctx context.Context, taskType string, payload any) (jobs.EnqueueInfo, error) {
	return s.JobsRuntime().EnqueueTask(ctx, taskType, payload)
}

// RegisterJobHandler registers a task handler in the jobs runtime.
func (s *Server) RegisterJobHandler(taskType string, handler jobs.JobHandler) error {
	return s.JobsRuntime().RegisterTaskHandler(taskType, handler)
}

// RegisterScheduledJob registers a recurring task in the jobs runtime scheduler.
func (s *Server) RegisterScheduledJob(cronspec, taskType string, payload any, opts ...jobs.Option) (string, error) {
	return s.JobsRuntime().RegisterScheduledTask(cronspec, taskType, payload, opts...)
}
