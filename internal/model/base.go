package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UID       string `gorm:"uniqueIndex;default:null;not null"`
	ID        uint   `gorm:"primarykey"`
}

func (mb *ModelBase) BeforeCreate(tx *gorm.DB) (err error) {
	mb.UID = xid.New("bas")
	return nil
}
