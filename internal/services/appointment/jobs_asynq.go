package appointment

import (
	"context"
	"errors"
	"log"

	"miconsul/internal/jobs"

	"github.com/hibiken/asynq"
)

const reminderSweepSchedule = "@every 1m"

func (s *service) bootstrapJobs() error {
	if s == nil {
		return nil
	}

	if err := s.registerReminderSweepHandler(); err != nil {
		if errors.Is(err, jobs.ErrRuntimeUnavailable) {
			return nil
		}
		return err
	}

	if _, err := s.JobsRuntime().RegisterScheduledTask(reminderSweepSchedule, TaskReminderSweep, TaskReminderSweepPayload{}); err != nil {
		if errors.Is(err, jobs.ErrRuntimeUnavailable) {
			return nil
		}
		return err
	}

	return nil
}

func (s *service) registerReminderSweepHandler() error {
	return s.JobsRuntime().RegisterTaskHandler(TaskReminderSweep, asynq.HandlerFunc(s.handleReminderSweepTask))
}

func (s *service) handleReminderSweepTask(context.Context, *asynq.Task) error {
	log.Printf("appointment jobs: reminder sweep task executed")
	return nil
}
