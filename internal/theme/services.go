package theme

import (
	"rtx-blog/internal/server"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}
