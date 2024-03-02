package todos

import (
	"fiber-blueprint/internal/database"
	"fiber-blueprint/internal/view"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (r *Router) HandleTodos(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		theme  = c.Query("theme")
		tds    []database.Todo
		left   int64
	)

	if theme == "" {
		theme = "light"
	}
	r.SessionSet(c, "theme", theme)

	tds = fetchTodos(r.DB, filter)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosPage(tds, strconv.Itoa(int(left)), filter, theme))
}

func (r *Router) HandleFilteredTodos(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		left   int64
	)

	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	c.Set("HX-Trigger", "fetchTodos")

	return view.Render(c, TodosFooter(strconv.Itoa(int(left)), filter))
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

	c.Set("HX-Trigger", "refreshFooter")

	return view.Render(c, TodoCard(t))
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

	c.Set("HX-Trigger", "refreshFooter")

	return view.Render(c, TodoCard(dup))
}

func (r *Router) HandleDeleteTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	r.DB.Delete(&t, id)

	c.Set("HX-Trigger", "refreshFooter")
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

	c.Set("HX-Trigger", "refreshFooter")

	t.Completed = true
	r.DB.Save(&t)

	return view.Render(c, TodoTitle(t))
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

	c.Set("HX-Trigger", "refreshFooter")

	return view.Render(c, TodoTitle(t))
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

func (r *Router) HandleTodosFragment(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		tds    []database.Todo
		left   int64
	)

	fmt.Println("Fetching todos")
	fmt.Println(filter)

	tds = fetchTodos(r.DB, filter)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosList(tds))
}
