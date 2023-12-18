package counter

import (
	"context"
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Router struct {
	*server.Server
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) RegisterRoutes(s *server.Server) {
	r.Server = s

	g := r.Group("/counter")

	g.Get("", r.HandlePage)
	g.Put("/increment", r.HandleIncrement)
	g.Put("/decrement", r.HandleDecrement)
}

func (r *Router) HandleIncrement(c *fiber.Ctx) error {
	count := r.sessionCountVal(c)
	count++
	r.SessionSetVal(c, "count", strconv.FormatInt(count, 10))

	component := view.CounterContainer(int64(count))
	c.Append("Content-Type", "text/html")

	return component.Render(c.Context(), c)
}

func (r *Router) HandleDecrement(c *fiber.Ctx) error {
	count := r.sessionCountVal(c)
	count--
	r.SessionSetVal(c, "count", strconv.FormatInt(count, 10))

	component := view.CounterContainer(int64(count))
	c.Append("Content-Type", "text/html")

	return component.Render(context.Background(), c)
}

func (r *Router) HandlePage(c *fiber.Ctx) error {
	count := r.sessionCountVal(c)
	r.SessionSetVal(c, "count", strconv.FormatInt(count, 10))

	component := view.CounterPage(int64(count))
	c.Append("Content-Type", "text/html")

	return component.Render(context.Background(), c)
}

// Utils
func (r *Router) sessionCountVal(c *fiber.Ctx) int64 {
	count, err := strconv.Atoi(r.SessionVal(c, "count"))
	if err != nil {
		count = 0
	}

	return int64(count)
}
