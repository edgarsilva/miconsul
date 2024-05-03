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

func (s service) fetchByFilter(filter string) []database.Todo {
	todos := []database.Todo{}

	switch {
	case strings.EqualFold(filter, "all"):
		s.DB.Order("created_at desc").Limit(100).Find(&todos)
	case strings.EqualFold(filter, "pending"):
		s.DB.Order("created_at desc").Limit(100).Where("completed = ?", false).Find(&todos)
	case strings.EqualFold(filter, "completed"):
		s.DB.Order("created_at desc").Limit(100).Where("completed = ?", true).Find(&todos)
	default:
		s.DB.Order("created_at desc, id desc").Limit(100).Find(&todos)
	}

	return todos
}

func (s service) pendingTodosCount() int {
	var (
		allCount       int64
		completedCount int64
	)
	s.DB.Model(&database.Todo{}).Count(&allCount)
	s.DB.Model(&database.Todo{}).Where("completed = ?", true).Count(&completedCount)

	return int(allCount - completedCount)
}

func (s service) todosCount() int {
	var allCount int64

	s.DB.Model(&database.Todo{}).Count(&allCount)
	return int(allCount)
}

func (s service) completedCount() int {
	var completedCount int64

	s.DB.Model(&database.Todo{}).Where("completed = ?", true).Count(&completedCount)
	return int(completedCount)
}
