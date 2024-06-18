package model

import (
	"time"

	"github.com/edgarsilva/miconsul/internal/lib/xid"
	"gorm.io/gorm"
)

type Todo struct {
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	Content   string    `gorm:"default:null;not null"`
	UserID    string    `gorm:"index;default:null;not null"`
	ModelBase
	User      User
	Completed bool
}

func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = xid.New("tdo")
	return nil
}
