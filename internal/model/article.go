package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Article struct {
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	UpdatedAt time.Time
	UserID    string `gorm:"index;default:null;not null"`
	Title     string
	Content   string
	ModelBase
	Comments []Comment
	User     User
}

func (t *Article) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = xid.New("art")
	return nil
}
