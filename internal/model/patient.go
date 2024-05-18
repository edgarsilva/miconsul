package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Patient struct {
	Address
	SocialMedia
	ModelBase
	ExtID               string
	Email               string `form:"email"`
	Phone               string `form:"phone"`
	Ocupation           string `form:"ocupation"`
	UserID              string `gorm:"index;default:null;not null"`
	LastName            string `gorm:"default:null;not null" form:"lastName"`
	FirstName           string `gorm:"default:null;not null" form:"firstName"`
	ProfilePic          string
	FamilyHistory       string `form:"familyHistory"`
	MedicalBackground   string `form:"medicalBackground"`
	Notes               string `form:"notes"`
	User                User
	Age                 int  `form:"age"`
	EnableNotifications bool `form:"enableNotifications"`
}

func (p *Patient) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = xid.New("ptnt")
	return nil
}

func (p *Patient) Name() string {
	return p.FirstName + " " + p.LastName
}
