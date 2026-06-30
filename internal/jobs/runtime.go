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
	registeredSchedules map[scheduleKey]string
}

// scheduleKey dedups scheduled-job registrations by their (cronspec, jobType)
// pair. Using a struct key avoids any string-delimiter collisions.
type scheduleKey struct {
	cronspec string
	jobType  JobType
}

func (k scheduleKey) String() string {
	return k.cronspec + ":" + k.jobType.String()
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
		return nil, fmt.Errorf("build jobs runtime config: %w", err)
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
		registeredSchedules: map[scheduleKey]string{},
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

func (r *Runtime) RegisterJobHandler(jobType JobType, handler Handler) error {
	if r == nil || !r.enabled || r.mux == nil {
		return ErrRuntimeUnavailable
	}

	jobType = JobType(strings.TrimSpace(jobType.String()))
	if jobType == "" {
		return ErrJobTypeRequired
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
		log.Printf("failed to register job handler: duplicate registration skipped for %q", jobType)
		return nil
	}

	r.mux.Handle(jobType.String(), newJobHandler(handler))
	r.registeredHandlers[jobType] = struct{}{}
	return nil
}

func (r *Runtime) RegisterScheduledJob(cronspec string, jobType JobType, payload any, opts ...Option) (string, error) {
	if r == nil || !r.enabled || r.scheduler == nil {
		return "", ErrRuntimeUnavailable
	}

	cronspec = strings.TrimSpace(cronspec)
	if cronspec == "" {
		return "", ErrScheduleSpecMissing
	}

	key := scheduleKey{cronspec: cronspec, jobType: jobType}

	r.registrationMu.Lock()
	defer r.registrationMu.Unlock()

	if r.registeredSchedules == nil {
		r.registeredSchedules = map[scheduleKey]string{}
	}
	if entryID, exists := r.registeredSchedules[key]; exists {
		log.Printf("failed to register scheduled job: duplicate registration skipped for %q", key)
		return entryID, nil
	}

	task, err := newTask(jobType, payload, opts...)
	if err != nil {
		return "", err
	}

	entryID, err := r.scheduler.Register(cronspec, task)
	if err != nil {
		return "", fmt.Errorf("register scheduled job %s: %w", jobType, err)
	}
	r.registeredSchedules[key] = entryID

	return entryID, nil
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
