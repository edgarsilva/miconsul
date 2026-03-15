package jobs

import (
	"errors"
	"fmt"
	"strings"

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
}

var (
	ErrHandlerRequired     = errors.New("handler is required")
	ErrScheduleSpecMissing = errors.New("schedule spec is required")
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
		scheduler: asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{}),
		mux:       asynq.NewServeMux(),
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

func (r *Runtime) RegisterTaskHandler(taskType string, handler asynq.Handler) error {
	if r == nil || !r.enabled || r.mux == nil {
		return ErrRuntimeUnavailable
	}

	taskType = strings.TrimSpace(taskType)
	if taskType == "" {
		return ErrTaskTypeRequired
	}
	if handler == nil {
		return ErrHandlerRequired
	}

	r.mux.Handle(taskType, handler)
	return nil
}

func (r *Runtime) RegisterScheduledTask(spec, taskType string, payload any, opts ...asynq.Option) (string, error) {
	if r == nil || !r.enabled || r.scheduler == nil {
		return "", ErrRuntimeUnavailable
	}

	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", ErrScheduleSpecMissing
	}

	task, err := newTask(taskType, payload, opts...)
	if err != nil {
		return "", err
	}

	entryID, err := r.scheduler.Register(spec, task)
	if err != nil {
		return "", fmt.Errorf("register scheduled task %s: %w", taskType, err)
	}

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
