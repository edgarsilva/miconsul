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

// EnqueueTask enqueues a background task through the jobs runtime.
func (s *Server) EnqueueTask(ctx context.Context, taskType string, payload any) (jobs.EnqueueInfo, error) {
	return s.JobsRuntime().EnqueueTask(ctx, taskType, payload)
}

// RegisterTaskHandler registers a task handler in the jobs runtime.
func (s *Server) RegisterTaskHandler(taskType string, handler jobs.HandlerFunc) error {
	return s.JobsRuntime().RegisterTaskHandler(taskType, handler)
}

// RegisterScheduledTask registers a recurring task in the jobs runtime scheduler.
func (s *Server) RegisterScheduledTask(spec, taskType string, payload any, opts ...jobs.Option) (string, error) {
	return s.JobsRuntime().RegisterScheduledTask(spec, taskType, payload, opts...)
}
