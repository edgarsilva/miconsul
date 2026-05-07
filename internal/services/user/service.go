package user

import (
	"context"
	"errors"
	"strings"

	"miconsul/internal/models"
	"miconsul/internal/server"

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

func (s service) FindRecentUsers(ctx context.Context, limit int) ([]models.User, error) {
	users, err := gorm.G[models.User](s.DB.GormDB()).Order("id DESC").Limit(limit).Find(ctx)
	if err != nil {
		return []models.User{}, err
	}

	return users, nil
}

func (s service) TakeUserByUID(ctx context.Context, UID string) (models.User, error) {
	UID = strings.TrimSpace(UID)
	if UID == "" {
		return models.User{}, ErrIDRequired
	}

	user, err := gorm.G[models.User](s.DB.GormDB()).Where("uid = ?", UID).Take(ctx)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s service) TakeUserByID(ctx context.Context, userID string) (models.User, error) {
	return s.TakeUserByUID(ctx, userID)
}

func (s service) UpdateUserProfileByUID(ctx context.Context, userUID string, updates models.User) (models.User, error) {
	userUID = strings.TrimSpace(userUID)
	if userUID == "" {
		return models.User{}, ErrIDRequired
	}

	err := normalizeUserWriteInput(&updates)
	if err != nil {
		return models.User{}, err
	}

	rowsAffected, err := gorm.G[models.User](s.DB.GormDB()).
		Where("uid = ?", userUID).
		Updates(ctx, updates)
	if err != nil {
		return models.User{}, err
	}
	if rowsAffected != 1 {
		return models.User{}, gorm.ErrRecordNotFound
	}

	user, err := s.TakeUserByUID(ctx, userUID)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s service) CreateUsersInBatches(ctx context.Context, users []models.User, batchSize int) error {
	return gorm.G[models.User](s.DB.GormDB()).CreateInBatches(ctx, &users, batchSize)
}

func normalizeUserWriteInput(user *models.User) error {
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
