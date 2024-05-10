package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Comment struct {
	UserID    string `gorm:"index;default:null;not null"`
	ArticleID string `gorm:"index;default:null;not null"`
	Content   string
	User      User // Belongs to User
	ModelBase
	Article Article // Belongs to Article
}

func (c *Comment) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = xid.New("cmt")
	return nil
}
