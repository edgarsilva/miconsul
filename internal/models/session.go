package models

import (
	"time"

	"gorm.io/gorm"
	"miconsul/internal/lib/xid"
)

type SessionType string

const (
	SessionCookie SessionType = "cookie"
	SessionJWT    SessionType = "jwt"
)

type Session struct {
	ExpiresAt time.Time
	ModelBase
	ID       uint   `gorm:"primaryKey"`
	UID      string `gorm:"uniqueIndex;default:null;not null"`
	Token    string
	Email    string      `gorm:"index;default:null;not null"`
	UserID   uint        `gorm:"index;default:null;not null"`
	AuthType SessionType `gorm:"default:null;not null;type:string"`
	User     User
}

func (u *Session) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.New("sess")
	return nil
}
