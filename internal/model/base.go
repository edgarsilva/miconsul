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
