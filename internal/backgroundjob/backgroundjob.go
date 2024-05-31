package backgroundjob

import (
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Sched struct {
	gocron.Scheduler
}

func New() (scheduler *Sched, shutdownFn func()) {
	s, err := gocron.NewScheduler(gocron.WithLogger(gocron.NewLogger(gocron.LogLevelDebug)))
	if err != nil {
		log.Panic("Failed to start gocron scheduler", err.Error())
	}
	s.Start()

	// when you're done, shut it down
	shutdownFn = func() {
		err = s.Shutdown()
		if err != nil {
			log.Panic("Failed to gracefully Shutdown gocron scheduler", err.Error())
		}
	}

	return &Sched{
		Scheduler: s,
	}, shutdownFn
}

// RunCronJob runs the function passed as a bg job (goroutine) at the interval
// specefied by the crontab
func (s *Sched) RunCronJob(crontab string, withSeconds bool, taskFn func()) (gocron.Job, error) {
	return s.NewJob(
		gocron.CronJob(
			// standard cron tab parsing
			crontab,
			withSeconds,
		),

		gocron.NewTask(taskFn),
	)
}

// RunJobAt runs the function passed as a bg job (goroutine) at the specified
// time.Time
func (s *Sched) RunJobAt(t time.Time, fn func()) (gocron.Job, error) {
	return s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartDateTime(t),
		),
		gocron.NewTask(fn),
	)
}

// RunJobImmediately runs the function passed as a bg job (goroutine) Immediately
func (s *Sched) RunJobImmediately(fn func()) (gocron.Job, error) {
	return s.NewJob(
		gocron.OneTimeJob(
			gocron.OneTimeJobStartImmediately(),
		),
		gocron.NewTask(
			func() {
				fmt.Println("-----------> This job runs immediately after server started once....")
			},
		),
	)
}
