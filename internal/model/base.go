package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        string `gorm:"primarykey;default:null;not null" form:"id"`
}

func (mb *ModelBase) BeforeCreate(tx *gorm.DB) (err error) {
	mb.ID = xid.New("____")
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

type Country struct {
	Name         string
	Language     string
	Code         string
	Abbreviation string
	Locale       string
}

func countries() []Country {
	c := []Country{
		{
			Name:         "United States of America",
			Language:     "English",
			Code:         "US",
			Abbreviation: "USA",
			Locale:       "en_US",
		},
		{
			Name:         "Mexico",
			Language:     "Spanish",
			Code:         "MX",
			Abbreviation: "MEX",
			Locale:       "es_MX",
		},
		{
			Name:         "Canada",
			Language:     "English, French",
			Code:         "CA",
			Abbreviation: "CAN",
			Locale:       "en_CA, fr_CA",
		},
	}

	return c
}
