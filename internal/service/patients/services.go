package patients

import (
	"github.com/edgarsilva/go-scaffold/internal/server"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}
