package cronjob

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Sched struct {
	gocron.Scheduler
}

func New() (scheduler *Sched, shutdownFn func() error, err error) {
	s, err := gocron.NewScheduler(gocron.WithLogger(gocron.NewLogger(gocron.LogLevelInfo)))
	if err != nil {
		return nil, nil, fmt.Errorf("cronjob: start scheduler: %w", err)
	}
	s.Start()

	// when you're done, shut it down
	shutdownFn = func() error {
		if err := s.Shutdown(); err != nil {
			return fmt.Errorf("cronjob: graceful shutdown: %w", err)
		}

		return nil
	}

	return &Sched{
		Scheduler: s,
	}, shutdownFn, nil
}

// RunCron runs the function passed as a cronjob (goroutine) at the interval
// specefied by the crontab
func (s *Sched) RunCron(crontab string, withSeconds bool, taskFn func()) (gocron.Job, error) {
	return s.NewJob(
		gocron.CronJob(
			crontab,
			withSeconds,
		),

		gocron.NewTask(taskFn),
	)
}

// RunAt runs the function passed as a bg job (goroutine) at the specified
// time.Time
func (s *Sched) RunAt(t time.Time, fn func()) (gocron.Job, error) {
	return s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(t),
		),
		gocron.NewTask(fn),
	)
}

// RunImmediately runs the function passed as a bg job (goroutine) Immediately
func (s *Sched) RunImmediately(fn func()) (gocron.Job, error) {
	return s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartImmediately(),
		),
		gocron.NewTask(fn),
	)
}
