package todos

import (
	"strings"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/server"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}

func fetchByFilter(DB *database.Database, filter string) []database.Todo {
	todos := []database.Todo{}

	switch {
	case strings.EqualFold(filter, "all"):
		DB.Order("created_at desc").Limit(100).Find(&todos)
	case strings.EqualFold(filter, "pending"):
		DB.Order("created_at desc").Limit(100).Where("completed = ?", false).Find(&todos)
	case strings.EqualFold(filter, "completed"):
		DB.Order("created_at desc").Limit(100).Where("completed = ?", true).Find(&todos)
	default:
		DB.Order("created_at desc, id desc").Limit(100).Find(&todos)
	}

	return todos
}

func fetchPendingCount(DB *database.Database) int {
	var (
		allCount       int64
		completedCount int64
	)
	DB.Model(&database.Todo{}).Count(&allCount)
	DB.Model(&database.Todo{}).Where("completed = ?", true).Count(&completedCount)

	return int(allCount - completedCount)
}