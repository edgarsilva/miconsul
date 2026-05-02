// Logbook saves the purpose of storing error/info/warn logs.
package models

import (
	"miconsul/internal/lib/xid"

	"gorm.io/gorm"
)

type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelError LogLevel = "error"
	LogLevelWarn  LogLevel = "warn"
	LogLevelDebug LogLevel = "debug"
)

type Logbook struct {
	ID              uint   `gorm:"primaryKey"`
	UID             string `gorm:"uniqueIndex;default:null;not null"`
	Log             string `gorm:"default:null;not null"`
	Msg             string
	Data            string
	Level           LogLevel `gorm:"index;default:pending;not null;type:string" form:"-"`
	LogbookableID   string   `gorm:"index:idx_poly_logbookable"`
	LogbookableType string   `gorm:"index:idx_poly_logbookable"`
	ModelBase
}

func (l *Logbook) BeforeCreate(tx *gorm.DB) (err error) {
	l.UID = xid.New("lgbk")
	return nil
}
