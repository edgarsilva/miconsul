// Package jobs provides the runtime for jobs that wraps asynq library.
package jobs

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/valkey"

	"github.com/hibiken/asynq"
)

type Runtime struct {
	enabled   bool
	client    *asynq.Client
	server    *asynq.Server
	scheduler *asynq.Scheduler
	mux       *asynq.ServeMux

	registrationMu      sync.Mutex
	registeredHandlers  map[JobType]struct{}
	registeredSchedules map[string]string
}

var (
	ErrHandlerRequired     = errors.New("handler is required")
	ErrScheduleSpecMissing = errors.New("schedule cronspec is required")
)

func New(env *appenv.Env) (*Runtime, error) {
	if env == nil {
		return nil, fmt.Errorf("jobs runtime requires environment")
	}

	if !env.JobsEnabled {
		return &Runtime{}, nil
	}

	valkeyConfig, err := valkey.NewConfig(env)
	if err != nil {
		return nil, fmt.Errorf("jobs runtime valkey config: %w", err)
	}

	redisOpt := asynq.RedisClientOpt{
		Addr:     valkeyConfig.Address,
		Password: valkeyConfig.Password,
		DB:       valkeyConfig.DB,
	}

	runtime := &Runtime{
		enabled: true,
		client:  asynq.NewClient(redisOpt),
		server: asynq.NewServer(redisOpt, asynq.Config{
			Concurrency: 10,
			Queues:      map[string]int{"default": 1},
		}),
		scheduler:           asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{}),
		mux:                 asynq.NewServeMux(),
		registeredHandlers:  map[JobType]struct{}{},
		registeredSchedules: map[string]string{},
	}

	if err := runtime.server.Start(runtime.mux); err != nil {
		_ = runtime.client.Close()
		return nil, fmt.Errorf("start jobs server: %w", err)
	}
	if err := runtime.scheduler.Start(); err != nil {
		runtime.server.Shutdown()
		_ = runtime.client.Close()
		return nil, fmt.Errorf("start jobs scheduler: %w", err)
	}

	return runtime, nil
}

func (r *Runtime) RegisterTaskHandler(jobType JobType, handler Handler) error {
	if r == nil || !r.enabled || r.mux == nil {
		return ErrRuntimeUnavailable
	}

	jobType = JobType(strings.TrimSpace(jobType.String()))
	if jobType == "" {
		return ErrTaskTypeRequired
	}
	if handler == nil {
		return ErrHandlerRequired
	}

	r.registrationMu.Lock()
	defer r.registrationMu.Unlock()

	if r.registeredHandlers == nil {
		r.registeredHandlers = map[JobType]struct{}{}
	}
	if _, exists := r.registeredHandlers[jobType]; exists {
		log.Printf("jobs runtime: duplicate task handler registration skipped for %q", jobType)
		return nil
	}

	r.mux.Handle(jobType.String(), newJobHandler(handler))
	r.registeredHandlers[jobType] = struct{}{}
	return nil
}

func (r *Runtime) RegisterScheduledTask(cronspec string, jobType JobType, payload any, opts ...Option) (string, error) {
	if r == nil || !r.enabled || r.scheduler == nil {
		return "", ErrRuntimeUnavailable
	}

	cronspec = strings.TrimSpace(cronspec)
	if cronspec == "" {
		return "", ErrScheduleSpecMissing
	}

	registrationKey := scheduledTaskRegistrationKey(cronspec, jobType)

	r.registrationMu.Lock()
	defer r.registrationMu.Unlock()

	if r.registeredSchedules == nil {
		r.registeredSchedules = map[string]string{}
	}
	if entryID, exists := r.registeredSchedules[registrationKey]; exists {
		log.Printf("jobs runtime: duplicate scheduled task registration skipped for %q", registrationKey)
		return entryID, nil
	}

	task, err := newTask(jobType, payload, opts...)
	if err != nil {
		return "", err
	}

	entryID, err := r.scheduler.Register(cronspec, task)
	if err != nil {
		return "", fmt.Errorf("register scheduled task %s: %w", jobType, err)
	}
	r.registeredSchedules[registrationKey] = entryID

	return entryID, nil
}

func scheduledTaskRegistrationKey(cronspec string, jobType JobType) string {
	return strings.TrimSpace(cronspec) + ":" + strings.TrimSpace(jobType.String())
}

func (r *Runtime) Enabled() bool {
	if r == nil {
		return false
	}

	return r.enabled
}

func (r *Runtime) Shutdown() error {
	if r == nil {
		return nil
	}

	if r.scheduler != nil {
		r.scheduler.Shutdown()
	}
	if r.server != nil {
		r.server.Shutdown()
	}
	if r.client != nil {
		if err := r.client.Close(); err != nil {
			return fmt.Errorf("close jobs client: %w", err)
		}
	}

	return nil
}
