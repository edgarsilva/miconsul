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

// EnqueueJob enqueues a background job through the jobs runtime.
func (s *Server) EnqueueJob(ctx context.Context, jobType jobs.JobType, payload any) (jobs.EnqueueInfo, error) {
	return s.JobsRuntime().EnqueueTask(ctx, jobType, payload)
}

// RegisterJobHandler registers a job handler in the jobs runtime.
func (s *Server) RegisterJobHandler(jobType jobs.JobType, handler jobs.Handler) error {
	return s.JobsRuntime().RegisterTaskHandler(jobType, handler)
}

// RegisterScheduledJob registers a recurring job in the jobs runtime scheduler.
func (s *Server) RegisterScheduledJob(cronspec string, jobType jobs.JobType, payload any, opts ...jobs.Option) (string, error) {
	return s.JobsRuntime().RegisterScheduledTask(cronspec, jobType, payload, opts...)
}
