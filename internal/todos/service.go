package todos

import (
	"fiber-blueprint/internal/database"
	"fmt"
	"strings"
)

func fetchTodos(db *database.Database, filter string) []database.Todo {
	todos := []database.Todo{}
	fmt.Println("filter:", filter)

	switch {
	case strings.EqualFold(filter, "all"):
		db.Find(&todos)
	case strings.EqualFold(filter, "pending"):
		db.Where("completed = ?", false).Find(&todos)
	case strings.EqualFold(filter, "completed"):
		db.Where("completed = ?", true).Find(&todos)
	default:
		db.Find(&todos)
	}

	return todos
}
