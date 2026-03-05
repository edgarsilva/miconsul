package user

import (
	"errors"
	"miconsul/internal/server"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("user service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}
