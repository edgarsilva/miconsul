package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Clinic struct {
	ExtID      string
	ProfilePic string
	Name       string `gorm:"default:null;not null"`
	Email      string
	Phone      string
	UserID     string `gorm:"index;default:null;not null"`
	Address
	SocialMedia
	ModelBase
	User User
}

func (c *Clinic) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = xid.New("clnc")
	return nil
}
