package theme

import (
	"errors"
	"miconsul/internal/server"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("theme service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}
