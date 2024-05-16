package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        string `gorm:"primarykey;default:null;not null"`
}

func (mb *ModelBase) BeforeCreate(tx *gorm.DB) (err error) {
	mb.ID = xid.New("___")
	return nil
}

type Address struct {
	Line1   string `form:"addressLine1"`
	Line2   string `form:"addressLine2"`
	City    string `form:"addressCity"`
	State   string `form:"addressState"`
	Country string `form:"addressCountry"`
	Zip     string `form:"addressZipCode"`
}

type SocialMedia struct {
	Whatsapp  string `form:"whatsapp"`
	Telegram  string `form:"telegram"`
	Messenger string `form:"messenger"`
	Instagram string `form:"instagram"`
	Facebook  string `form:"facebook"`
}
