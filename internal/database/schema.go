package database

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"

	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UID       string `gorm:"uniqueIndex;default:null;not null"`
	ID        uint   `gorm:"primarykey"`
}

func (u *ModelBase) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.New("usr")
	return nil
}

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
	UserRoleGuest UserRole = "guest"
	UserRoleAnon  UserRole = "anon"
	UserRoleTest  UserRole = "test"
)
