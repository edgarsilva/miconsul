package todos

import (
	"fiber-blueprint/internal/database"
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/view"

	"github.com/a-h/templ"
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

	g := r.Group("/todos")
	g.Get("", r.HandleTodos)
	g.Post("", r.HandleCreateTodo)
	g.Delete("/:id<int>", r.HandleDeleteTodo)
	g.Post("/:id<int>/duplicate", r.HandleDuplicateTodo)
	g.Patch("/:id<int>/check", r.HandleCheckTodo)
	g.Patch("/:id<int>/uncheck", r.HandleUncheckTodo)

	g.Get("/api/todos", r.HandleApiTodos)
}

func (r *Router) HandleTodos(c *fiber.Ctx) error {
	var tds []database.Todo
	r.DB.Find(&tds)
	return render(c, view.TodosPage(tds))
}

func (r *Router) HandleApiTodos(c *fiber.Ctx) error {
	var tds []database.Todo
	r.DB.Find(&tds)
	return c.JSON(tds)
}

func (r *Router) HandleCreateTodo(c *fiber.Ctx) error {
	t := database.Todo{
		Title:     "Buy milk",
		Body:      "Ad the supermarket store",
		Priority:  "High",
		Completed: false,
	}

	r.DB.Create(&t)

	return c.JSON(t)
}

func (r *Router) HandleDuplicateTodo(c *fiber.Ctx) error {
	var (
		id  = c.Params("id")
		ori database.Todo
		dup database.Todo
	)

	if res := r.DB.First(&ori, id); res.Error != nil {
		c.SendStatus(fiber.StatusMethodNotAllowed)
		return c.SendString("")
	}

	dup = ori
	dup.ID = 0
	r.DB.Create(&dup)

	return render(c, view.Todo(dup))
}

func (r *Router) HandleDeleteTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	r.DB.Delete(&t, id)

	c.SendStatus(fiber.StatusOK)
	return c.SendString("")
}

func (r *Router) HandleCheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	if res := r.DB.First(&t, id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	t.Completed = true
	r.DB.Save(&t)

	return render(c, view.Checkbox(t))
}

func (r *Router) HandleUncheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	if res := r.DB.First(&t, id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	t.Completed = false
	r.DB.Save(&t)

	return render(c, view.Checkbox(t))
}

func render(ctx *fiber.Ctx, com templ.Component) error {
	ctx.Append("Content-Type", "text/html")
	return com.Render(ctx.Context(), ctx)
}
