package jobs

import (
	"net/http"

	"miconsul/internal/lib/appenv"

	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
)

func NewMonitorHandler(rootPath string, env *appenv.Env) http.Handler {
	return asynqmon.New(asynqmon.Options{
		RootPath: rootPath,
		RedisConnOpt: asynq.RedisClientOpt{
			Addr:     env.ValkeyAddress(),
			Password: env.ValkeyPassword,
			DB:       env.ValkeyDB,
		},
	})
}
