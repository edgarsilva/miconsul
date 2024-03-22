package todos

import (
	"fiber-blueprint/internal/database"
	"fiber-blueprint/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

// GET: /todos.html - Get all todos paginated.
func (r *Router) handleTodos(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		tds    []database.Todo
		left   int64
	)

	theme := r.SessionGet(c, "theme", "light")
	if theme == "light" {
		r.SessionSet(c, "theme", "light")
	} else {
		r.SessionSet(c, "theme", "dark")
	}

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

	src.UID = id
	if res := r.DB.First(&src); res.Error != nil {
		c.SendStatus(fiber.StatusMethodNotAllowed)
		return c.SendString("")
	}

	dup = src
	dup.UID = ""
	r.DB.Create(&dup)

	c.Set("HX-Trigger", "refreshFooter")

	return view.Render(c, TodoCard(dup))
}

// DELETE: /todos/:id.html - Delete a todo
func (r *Router) handleDeleteTodo(c *fiber.Ctx) error {
	var (
		uid  = c.Params("id")
		todo database.Todo
	)

	todo.UID = uid
	r.DB.Delete(&todo, "uid = ?", uid)

	c.Set("HX-Trigger", "refreshFooter")
	c.SendStatus(fiber.StatusOK)

	return c.SendString("")
}

func (r *Router) handleCheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	if res := r.DB.First(&t, "uid = ?", id); res.Error != nil {
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

	if res := r.DB.First(&t, "uid = ?", id); res.Error != nil {
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
	var err error
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("pageSize"))
	if err != nil {
		pageSize = 10
	}

	// var tds []database.Todo
	type User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var tds []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		UserID string `json:"user_id"`
		User   User   `json:"user"`
	}

	r.DB.
		Model(&database.Todo{}).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name")
		}).
		Select("id, title, user_id").
		Where("user_id != ''").
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tds)

	return c.JSON(tds)
}

// handleApiUsers returns all users as JSON
// GET: /api/todos - Get all todos
func (r *Router) handleGetUsers(c *fiber.Ctx) error {
	var users []database.User

	r.DB.
		Model(&database.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []database.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
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

// handleCreate100000Todos creates 100000 todos
// GET: /api/todos/count
func (r *Router) handleCreateNTodos(c *fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n"))
	userID := c.Params("userID")
	if err != nil {
		n = 1000
	}

	todos := []database.Todo{}

	// var sb strings.Builder
	for i := 1; i <= n; i++ {
		todos = append(todos, database.Todo{
			Title:     "title " + strconv.Itoa(i),
			Body:      "body" + strconv.Itoa(i),
			Priority:  "High",
			UserID:    userID,
			Completed: false,
		})
	}

	res := r.DB.CreateInBatches(&todos, 2000)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (r *Router) handleEcho(c *fiber.Ctx) error {
	str := c.Params("str")

	return c.SendString(str)
}

func (r *Router) handleEchoQuery(c *fiber.Ctx) error {
	str := c.Query("str")

	return c.SendString(str)
}

func (r *Router) handleMakeUsers(c *fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil {
		n = 10
	}

	var users []database.User
	for i := 0; i <= n; i++ {
		users = append(users, database.User{
			Name:  faker.Name().Name(),
			Email: faker.Internet().Email(),
		})
	}

	res := r.DB.Create(&users)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}
