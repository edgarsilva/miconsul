package user

import (
	"context"
	"errors"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"strings"

	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

var ErrIDRequired = errors.New("id is required")

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("user service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}

func (s service) FindRecentUsers(ctx context.Context, limit int) ([]model.User, error) {
	users, err := gorm.G[model.User](s.DB.GormDB()).Order("id DESC").Limit(limit).Find(ctx)
	if err != nil {
		return []model.User{}, err
	}

	return users, nil
}

func (s service) TakeUserByID(ctx context.Context, userID string) (model.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return model.User{}, ErrIDRequired
	}

	user, err := gorm.G[model.User](s.DB.GormDB()).Where("id = ?", userID).Take(ctx)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s service) UpdateUserProfileByID(ctx context.Context, userID string, updates model.User) (model.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return model.User{}, ErrIDRequired
	}

	err := normalizeUserWriteInput(&updates)
	if err != nil {
		return model.User{}, err
	}

	rowsAffected, err := gorm.G[model.User](s.DB.GormDB()).
		Where("id = ?", userID).
		Updates(ctx, updates)
	if err != nil {
		return model.User{}, err
	}
	if rowsAffected != 1 {
		return model.User{}, gorm.ErrRecordNotFound
	}

	user, err := s.TakeUserByID(ctx, userID)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (s service) CreateUsersInBatches(ctx context.Context, users []model.User, batchSize int) error {
	return gorm.G[model.User](s.DB.GormDB()).CreateInBatches(ctx, &users, batchSize)
}

func normalizeUserWriteInput(user *model.User) error {
	if user == nil {
		return errors.New("user is required")
	}

	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Phone = strings.TrimSpace(user.Phone)

	if len(user.Name) > 120 {
		return errors.New("name exceeds max length")
	}
	if len(user.Email) > 254 {
		return errors.New("email exceeds max length")
	}
	if len(user.Phone) > 40 {
		return errors.New("phone exceeds max length")
	}

	return nil
}
