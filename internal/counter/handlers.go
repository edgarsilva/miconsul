package counter

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/views"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandleIncrement(c *fiber.Ctx) error {
	cnt := s.sessionCountVal(c)
	cnt++
	s.SessionSet(c, "cnt", strconv.FormatInt(cnt, 10))

	return views.Render(c, CounterContainer(int64(cnt)))
}

func (s *service) HandleDecrement(c *fiber.Ctx) error {
	cnt := s.sessionCountVal(c)
	cnt--
	s.SessionSet(c, "cnt", strconv.FormatInt(cnt, 10))

	return views.Render(c, CounterContainer(int64(cnt)))
}

func (s *service) HandlePage(c *fiber.Ctx) error {
	cnt := s.sessionCountVal(c)
	s.SessionSet(c, "cnt", strconv.FormatInt(cnt, 10))

	return views.Render(c, CounterPage(views.Props{}, int64(cnt)))
}

// Utils
func (s *service) sessionCountVal(c *fiber.Ctx) int64 {
	cnt, err := strconv.Atoi(s.SessionGet(c, "cnt", "0"))
	if err != nil {
		cnt = 0
	}

	return int64(cnt)
}
