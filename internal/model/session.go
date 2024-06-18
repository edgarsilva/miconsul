package model

import (
	"time"

	"miconsul/internal/lib/xid"
	"gorm.io/gorm"
)

type SessionType string

const (
	SessionCookie SessionType = "cookie"
	SessionJWT    SessionType = "jwt"
)

type Session struct {
	ExpiresAt time.Time
	ModelBase
	Token    string
	Email    string      `gorm:"index;default:null;not null"`
	UserID   string      `gorm:"index;default:null;not null"`
	AuthType SessionType `gorm:"default:null;not null;type:string"`
	User     User
}

func (u *Session) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = xid.New("sess")
	return nil
}
