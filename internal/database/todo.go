package database

import (
	"fiber-blueprint/internal/xid"
	"time"

	"gorm.io/gorm"
)

type Todo struct {
	ID        string    `gorm:"type:string;primary_key;index:pxid_idx,unique"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Content   string    `json:"content"`
	Priority  string    `json:"priority"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	UpdatedAt time.Time
}

func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = xid.NewConcat("tdo")

	return
}

func (t *Todo) TableName() string {
	return "todos"
}
