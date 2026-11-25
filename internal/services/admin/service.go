package admin

import (
	"context"
	"errors"

	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("admin service requires a non-nil server")
	}

	return service{Server: s}, nil
}

// countUsers returns the number of users.
func (s *service) countUsers(ctx context.Context) (int64, error) {
	return gorm.G[models.User](s.DB.GormDB()).Count(ctx, "*")
}
