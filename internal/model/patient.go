package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Patient struct {
	ExtID      string
	ProfilePic string
	FirstName  string `gorm:"default:null;not null" form:"firstName"`
	LastName   string `gorm:"default:null;not null" form:"lastName"`
	Username   string
	Phone      string `form:"phone"`
	Email      string `form:"email"`
	ocupation  string `form:"ocupation"`
	UserID     string `gorm:"index;default:null;not null"`
	Address
	SocialMedia
	ModelBase
	User User
	Age  int `form:"age"`
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = xid.New("ptnt")
	return nil
}

func (p *Patient) Name() string {
	return p.FirstName + " " + p.LastName
}
