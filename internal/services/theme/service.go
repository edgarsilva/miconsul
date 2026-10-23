package theme

import (
	"context"
	"errors"
	"fmt"

	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/gorm"
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

// SaveUserTheme stores theme as the user's saved theme.
func (s *service) SaveUserTheme(ctx context.Context, userID uint, theme string) error {
	_, err := gorm.G[models.User](s.GormDB()).Where("id = ?", userID).Update(ctx, "theme", theme)
	if err != nil {
		return fmt.Errorf("save user theme: %w", err)
	}

	return nil
}
