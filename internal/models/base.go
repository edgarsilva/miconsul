package models

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ModelBase struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	fieldErrors map[string]string `gorm:"-:all"`
}

type Address struct {
	Line1   string `form:"addressLine1"`
	Line2   string `form:"addressLine2"`
	City    string `form:"addressCity"`
	State   string `form:"addressState"`
	Country string `form:"addressCountry"`
	Zip     string `form:"addressZipCode"`
}

func (a Address) FullAddress(sep string) string {
	return a.Line1 + sep + a.Line2 + sep + a.City + ", " + a.State + ", " + a.Country + sep + a.Zip
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
	ViaSMS              bool `form:"viaSMS"`
	ViaWhatsapp         bool `form:"viaWhatsapp"`
	ViaMessenger        bool `form:"viaMessenger"`
	ViaTelegram         bool `form:"viaTelegram"`
}

func (mb *ModelBase) IsValid() error {
	mb.fieldErrors = make(map[string]string)
	return nil
}

func (mb *ModelBase) FieldErrors() map[string]string {
	if mb.fieldErrors == nil {
		mb.fieldErrors = make(map[string]string)
	}

	return mb.fieldErrors
}

// FieldError returns the error associated to a field, defined by key
func (mb *ModelBase) FieldError(key string) string {
	if len(mb.fieldErrors) == 0 {
		return ""
	}

	return mb.fieldErrors[key]
}

// SetFieldError sets an error for a field, defined by key
func (mb *ModelBase) SetFieldError(key, errStr string) {
	if mb.fieldErrors == nil {
		mb.fieldErrors = make(map[string]string)
	}

	mb.fieldErrors[key] = errStr
}

type Country struct {
	Name         string
	Language     string
	Code         string
	Abbreviation string
	Locale       string
}

func Countries() []Country {
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

// GlobalFTS ranks the rows of the query's model by a full-text search for
// term. Rows are matched by uid through global_search_docs, which the DB keeps
// in sync with triggers. term is matched as a literal prefix phrase, so FTS5
// query syntax in it has no effect. An empty term orders by recency instead.
//
//	s.DB.
//		Model(&models.Patient{}).
//		Scopes(models.GlobalFTS(term)).
//		Find(&patients)
func GlobalFTS(term string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if term == "" {
			return db.Order("created_at desc")
		}

		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(db.Statement.Model); err != nil {
			db.AddError(fmt.Errorf("resolve global fts table: %w", err))
			return db
		}
		table := stmt.Schema.Table

		phrase := "\"" + strings.ReplaceAll(term, "\"", "\"\"") + "\" *"

		return db.
			Joins("INNER JOIN global_search_docs ON global_search_docs.gid = ?", clause.Column{Table: table, Name: "uid"}).
			Joins("INNER JOIN global_fts ON global_fts.rowid = global_search_docs.id").
			Where("global_fts MATCH ?", phrase).
			Order("bm25(global_fts, 5, 3, 2)")
	}
}
