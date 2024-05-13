package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Clinic struct {
	ExtID string
	Name  string `gorm:"default:null;not null"`
	AddressBase
	Email        string
	Phone        string `gorm:"default:null;not null"`
	InstagramURL string
	FacebookURL  string
	UserID       string `gorm:"index;default:null;not null"`
	ModelBase
	User User
}

func (c *Clinic) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = xid.New("clnc")
	return nil
}
