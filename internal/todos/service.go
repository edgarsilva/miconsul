package todos

import (
	"fiber-blueprint/internal/database"
	"strings"
)

func fetchTodos(db *database.Database, filter string) []database.Todo {
	todos := []database.Todo{}

	switch {
	case strings.EqualFold(filter, "all"):
		db.Limit(100).Find(&todos)
	case strings.EqualFold(filter, "pending"):
		db.Limit(100).Where("completed = ?", false).Find(&todos)
	case strings.EqualFold(filter, "completed"):
		db.Limit(100).Where("completed = ?", true).Find(&todos)
	default:
		db.Limit(100).Find(&todos)
	}

	return todos
}
