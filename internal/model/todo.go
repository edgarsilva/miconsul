package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Todo struct {
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	Content   string    `gorm:"default:null;not null"`
	ModelBase
	User      User
	UserID    uint `gorm:"index;default:null;not null"`
	Completed bool
}

func (t Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.UID = xid.New("tdo")
	return nil
}
