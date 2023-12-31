package counter

import (
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Router struct {
	*server.Server
}

func (r *Router) RegisterRoutes(s *server.Server) {
	r.Server = s

	g := r.Group("/counter")

	g.Get("", r.HandlePage)
	g.Put("/increment", r.HandleIncrement)
	g.Put("/decrement", r.HandleDecrement)
}

func (r *Router) HandleIncrement(c *fiber.Ctx) error {
	cnt := r.sessionCountVal(c)
	cnt++
	r.SessionSetVal(c, "cnt", strconv.FormatInt(cnt, 10))

	return view.Render(c, view.CounterContainer(int64(cnt)))
}

func (r *Router) HandleDecrement(c *fiber.Ctx) error {
	cnt := r.sessionCountVal(c)
	cnt--
	r.SessionSetVal(c, "cnt", strconv.FormatInt(cnt, 10))

	return view.Render(c, view.CounterContainer(int64(cnt)))
}

func (r *Router) HandlePage(c *fiber.Ctx) error {
	cnt := r.sessionCountVal(c)
	r.SessionSetVal(c, "cnt", strconv.FormatInt(cnt, 10))

	return view.Render(c, view.CounterPage(int64(cnt)))
}

// Utils
func (r *Router) sessionCountVal(c *fiber.Ctx) int64 {
	cnt, err := strconv.Atoi(r.SessionVal(c, "cnt"))
	if err != nil {
		cnt = 0
	}

	return int64(cnt)
}
