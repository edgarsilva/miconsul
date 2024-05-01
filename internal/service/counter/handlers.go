package counter

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (s *service) HandlePage(c *fiber.Ctx) error {
	count := s.sessionCountVal(c)
	s.SessionSet(c, "cnt", strconv.FormatInt(count, 10))

	theme := s.SessionGet(c, "theme", "light")
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme))
	return view.Render(c, view.CounterPage(count, layoutProps))
}

func (s *service) HandleIncrement(c *fiber.Ctx) error {
	cnt := s.sessionCountVal(c)
	cnt++
	s.SessionSet(c, "cnt", strconv.FormatInt(cnt, 10))

	return view.Render(c, view.CounterContainer(int64(cnt)))
}

func (s *service) HandleDecrement(c *fiber.Ctx) error {
	cnt := s.sessionCountVal(c)
	cnt--
	s.SessionSet(c, "cnt", strconv.FormatInt(cnt, 10))

	return view.Render(c, view.CounterContainer(int64(cnt)))
}

// Utils
func (s *service) sessionCountVal(c *fiber.Ctx) int64 {
	cnt, err := strconv.Atoi(s.SessionGet(c, "cnt", "0"))
	if err != nil {
		cnt = 0
	}

	return int64(cnt)
}
