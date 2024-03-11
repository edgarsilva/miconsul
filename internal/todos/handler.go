package todos

import (
	"fiber-blueprint/internal/database"
	"fiber-blueprint/internal/view"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

// GET: /todos.html - Get all todos paginated.
func (r *Router) handleTodos(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		theme  = c.Query("theme")
		tds    []database.Todo
		left   int64
	)

	if theme == "" {
		theme = r.SessionGet(c, "theme", "light")
	}

	r.SessionSet(c, "theme", theme)
	log.Info("error saving theme:", theme)

	tds = fetchTodos(r.DB, filter)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosPage(tds, strconv.Itoa(int(left)), filter, theme))
}

// GET: /todos.html - Get filtered todos.
func (r *Router) handleFilteredTodos(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		left   int64
	)

	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	c.Set("HX-Trigger", "fetchTodos")

	return view.Render(c, TodosFooter(strconv.Itoa(int(left)), filter))
}

func (r *Router) handleCreateTodo(c *fiber.Ctx) error {
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

// POST: /todos/:id/duplicate.html - Duplicates a todo
func (r *Router) handleDuplicateTodo(c *fiber.Ctx) error {
	var (
		id  = c.Params("id")
		src database.Todo
		dup database.Todo
	)

	src.ID = id
	if res := r.DB.First(&src); res.Error != nil {
		c.SendStatus(fiber.StatusMethodNotAllowed)
		return c.SendString("")
	}

	dup = src
	dup.ID = ""
	r.DB.Create(&dup)

	c.Set("HX-Trigger", "refreshFooter")

	return view.Render(c, TodoCard(dup))
}

// DELETE: /todos/:id.html - Delete a todo
func (r *Router) handleDeleteTodo(c *fiber.Ctx) error {
	var (
		id   = c.Params("id")
		todo database.Todo
	)

	todo.ID = id
	r.DB.Delete(&todo)

	c.Set("HX-Trigger", "refreshFooter")
	c.SendStatus(fiber.StatusOK)

	return c.SendString("")
}

func (r *Router) handleCheckTodo(c *fiber.Ctx) error {
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

func (r *Router) handleUncheckTodo(c *fiber.Ctx) error {
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
func (r *Router) handleFooterFragment(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		left   int64
	)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosFooter(strconv.Itoa(int(left)), filter))
}

func (r *Router) handleTodosFragment(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		tds    []database.Todo
		left   int64
	)

	tds = fetchTodos(r.DB, filter)
	r.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, TodosList(tds))
}

// API: /api/todos

// handleApiTodos returns all todos as JSON
// GET: /api/todos - Get all todos
func (r *Router) handleApiTodos(c *fiber.Ctx) error {
	var tds []database.Todo
	r.DB.Limit(10).Find(&tds)

	return c.JSON(tds)
}

// GET: /api/todos/Count - Count all todos
func (r *Router) handleCountTodos(c *fiber.Ctx) error {
	var count int64
	r.DB.Model(&database.Todo{}).Count(&count)
	res := struct {
		Count int64 `json:"count"`
	}{Count: count}

	return c.Status(fiber.StatusOK).JSON(res)
}

// handleCreate1000Todos creates 1000 todos
// GET: /api/todos/count
func (r *Router) handleCreate1000Todos(c *fiber.Ctx) error {
	todos := []database.Todo{}

	// var sb strings.Builder
	for i := 1; i <= 100000; i++ {
		todos = append(todos, database.Todo{
			Title:     "title " + strconv.Itoa(i),
			Body:      "body" + strconv.Itoa(i),
			Priority:  "High",
			Completed: false,
		})
	}

	r.DB.CreateInBatches(&todos, 500)

	var count int64
	r.DB.Model(&database.Todo{}).Count(&count)
	fmt.Println("count", count)

	res := struct {
		Count int64 `json:"count"`
	}{Count: count}

	return c.Status(fiber.StatusOK).JSON(res)
}
