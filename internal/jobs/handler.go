package jobs

import (
	"context"

	"github.com/hibiken/asynq"
)

type Handler func(ctx context.Context, job Job) error

type asynqHandlerAdapter struct {
	fn Handler
}

func newJobHandler(handler Handler) asynqHandlerAdapter {
	return asynqHandlerAdapter{fn: handler}
}

func (a asynqHandlerAdapter) ProcessTask(ctx context.Context, task *asynq.Task) error {
	t := Job{
		Type:    JobType(task.Type()),
		Payload: task.Payload(),
	}
	if id, ok := asynq.GetTaskID(ctx); ok {
		t.ID = id
	}
	if queue, ok := asynq.GetQueueName(ctx); ok {
		t.Queue = queue
	}
	if retryCount, ok := asynq.GetRetryCount(ctx); ok {
		t.RetryCount = retryCount
	}
	if maxRetry, ok := asynq.GetMaxRetry(ctx); ok {
		t.MaxRetry = maxRetry
	}

	return a.fn(ctx, t)
}
