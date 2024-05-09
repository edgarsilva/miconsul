package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
	UserRoleGuest UserRole = "guest"
	UserRoleAnon  UserRole = "anon"
	UserRoleTest  UserRole = "test"
)

type User struct {
	ConfirmEmailExpiresAt time.Time
	ResetTokenExpiresAt   time.Time
	Name                  string
	Email                 string   `gorm:"uniqueIndex;default:null;not null"`
	Role                  UserRole `gorm:"index;default:null;not null;type:string"`
	Password              string   `json:"-"`
	Theme                 string
	ResetToken            string
	ConfirmEmailToken     string
	ModelBase
	Todos    []Todo
	Articles []Article
	Comments []Comment
}

func (u User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.New("usr")
	return nil
}
