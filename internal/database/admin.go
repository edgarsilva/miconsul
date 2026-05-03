package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func EnsureAdminUser(ctx context.Context, env *appenv.Env, db *gorm.DB) error {
	if env == nil || db == nil {
		return nil
	}

	var adminCount int64
	if err := db.WithContext(ctx).Model(&models.User{}).Where("role = ?", models.UserRoleAdmin).Count(&adminCount).Error; err != nil {
		return fmt.Errorf("count admin users: %w", err)
	}
	if adminCount > 0 {
		return nil
	}

	email := strings.TrimSpace(strings.ToLower(env.AdminUser))
	password := env.AdminPassword
	if email == "" || password == "" {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	user := models.User{
		Email:             email,
		Password:          string(hashedPassword),
		Role:              models.UserRoleAdmin,
		ConfirmEmailToken: "",
	}
	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}

		return fmt.Errorf("create admin user: %w", err)
	}

	return nil
}
