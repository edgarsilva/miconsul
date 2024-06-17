package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	fieldErrors map[string]string `gorm:"-:all"`
	ID          string            `gorm:"primarykey;default:null;not null"`
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

type NotificationFlags struct {
	EnableNotifications bool `form:"enableNotifications"`
	ViaEmail            bool `form:"viaEmail"`
	ViaWhatsapp         bool `form:"viaWhatsapp"`
	ViaMessenger        bool `form:"viaMessenger"`
	ViaTelegram         bool `form:"viaTelegram"`
}

func (mb *ModelBase) BeforeCreate(tx *gorm.DB) error {
	mb.ID = xid.New("____")
	return nil
}

func (mb *ModelBase) IsValid() error {
	mb.fieldErrors = make(map[string]string)
	return nil
}

func (c *ModelBase) FieldErrors() map[string]string {
	if c.fieldErrors == nil {
		c.fieldErrors = make(map[string]string)
	}

	return c.fieldErrors
}

// FieldError returns the error associated to a field, defined by key
func (mb *ModelBase) FieldError(key string) string {
	if len(mb.fieldErrors) == 0 {
		return ""
	}

	return mb.fieldErrors[key]
}

// SetFieldError sets an error for a field, defined by key
func (c *ModelBase) SetFieldError(key, errStr string) {
	if c.fieldErrors == nil {
		c.fieldErrors = make(map[string]string)
	}

	c.fieldErrors[key] = errStr
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

// GlobalFTS adds the necessary joins to do a Global full text search on the
// global_fts table, as well as the order clause to rank the results.
//
// You can use it with any model that saves to the global_fts table (usually
// handle by the DB with triggers).
//
//	// you can use it with your model like so:
//	s.DB.
//		Model(model.Patient{}).
//		Scopes(model.GlobalFTS(queryStr)).
//		.Find(&patients)
//
//	// the scope adds this snippet:
//	db.
//		 Joins("INNER JOIN global_fts ON gid = id").
//		 Where("global_fts MATCH ?", "\""+term+"\" * ").
//		 Order("bm25(global_fts, 0, 1, 2, 3)")
func GlobalFTS(term string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if term == "" {
			return db.Order("created_at desc")
		}

		return db.
			Joins("INNER JOIN global_fts ON gid = id").
			Where("global_fts MATCH ?", "\""+term+"\" * ").
			Order("bm25(global_fts)")
	}
}
