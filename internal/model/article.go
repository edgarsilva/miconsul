package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Article struct {
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	UpdatedAt time.Time
	Title     string
	Content   string
	ModelBase
	Comments []Comment
	User     User
	UserID   uint `gorm:"index;default:null;not null"`
}

func (t Article) BeforeCreate(tx *gorm.DB) (err error) {
	t.UID = xid.New("tdo")
	return nil
}
