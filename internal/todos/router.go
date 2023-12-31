package todos

import (
	"fiber-blueprint/internal/database"
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

	g := r.Group("/todos")
	g.Get("", r.HandleTodos)
	g.Post("", r.HandleCreateTodo)
	g.Delete("/:id<int>", r.HandleDeleteTodo)
	g.Post("/:id<int>/duplicate", r.HandleDuplicateTodo)
	g.Patch("/:id<int>/check", r.HandleCheckTodo)
	g.Patch("/:id<int>/uncheck", r.HandleUncheckTodo)

	g.Get("/api/todos", r.HandleApiTodos)

	// OOB Fragments
	g.Get("/fragment/footer", r.HandleFooterFragment)
}

func (r *Router) HandleTodos(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		tds    []database.Todo
		left   int64
	)
	r.DB.Find(&tds)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosPage(tds, strconv.Itoa(int(left)), filter))
}

func (r *Router) HandleApiTodos(c *fiber.Ctx) error {
	var tds []database.Todo
	r.DB.Find(&tds)
	return c.JSON(tds)
}

func (r *Router) HandleCreateTodo(c *fiber.Ctx) error {
	t := database.Todo{
		Title:     "",
		Body:      "",
		Priority:  "High",
		Completed: false,
	}

	t.Title = c.FormValue("title")
	res := r.DB.Create(&t)

	if res.Error != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Set("HX-Trigger", "todosUpd")

	return view.Render(c, TodoLi(t))
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

	c.Set("HX-Trigger", "todosUpd")

	return view.Render(c, TodoLi(dup))
}

func (r *Router) HandleDeleteTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	r.DB.Delete(&t, id)

	c.Set("HX-Trigger", "todosUpd")
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

	c.Set("HX-Trigger", "todosUpd")

	t.Completed = true
	r.DB.Save(&t)

	return view.Render(c, TodoCheckbox(t))
}

func (r *Router) HandleUncheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	if res := r.DB.First(&t, id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	c.Set("HX-Trigger", "todosUpd")

	t.Completed = false
	r.DB.Save(&t)

	return view.Render(c, TodoCheckbox(t))
}

// Fragments
func (r *Router) HandleFooterFragment(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		left   int64
	)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosFooter(strconv.Itoa(int(left)), filter))
}
