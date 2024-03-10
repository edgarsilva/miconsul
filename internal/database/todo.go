package database

import (
	"errors"
	"fiber-blueprint/internal/nanoid"
	"time"

	"gorm.io/gorm"
)

type Todo struct {
	ID        string `gorm:"type:string;primary_key"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Content   string `json:"content"`
	Priority  string `json:"priority"`
	Completed bool   `json:"completed"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID, err = nanoid.New("td_")
	if err != nil {
		err = errors.New("failed to generate Todo primaryID(nanoid)")
	}

	return
}

func (t *Todo) TableName() string {
	return "todos"
}
