package valkey

import (
	"fmt"

	"miconsul/internal/lib/appenv"
)

type Config struct {
	Address  string
	Password string
	DB       int
}

func NewConfig(env *appenv.Env) (Config, error) {
	if env == nil {
		return Config{}, fmt.Errorf("valkey config requires environment")
	}

	return Config{
		Address:  env.ValkeyAddress(),
		Password: env.ValkeyPassword,
		DB:       env.ValkeyDB,
	}, nil
}
