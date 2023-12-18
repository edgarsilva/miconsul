package database

import "gorm.io/gorm"

type Todo struct {
	gorm.Model
	Title     string `json:"title"`
	Body      string `json:"body"`
	Priority  string `json:"priority"`
	Completed bool   `json:"completed"`
}
