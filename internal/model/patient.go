package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Patient struct {
	ExtID       string
	Name        string `gorm:"default:null;not null"`
	Email       string
	Phone       string `gorm:"default:null;not null"`
	FacebookURL string
	ProfileURL  string
	UserID      string `gorm:"index;default:null;not null"`
	ModelBase
	User User
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = xid.New("patn")
	return nil
}
