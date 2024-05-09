package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Comment struct {
	Content   string
	ArticleID string `gorm:"index;default:null;not null"`
	User      User   // Belongs to User
	ModelBase
	Article Article // Belongs to Article
	UserID  uint    `gorm:"index;default:null;not null"`
}

func (c Comment) BeforeCreate(tx *gorm.DB) (err error) {
	c.UID = xid.New("cmt")
	return nil
}
