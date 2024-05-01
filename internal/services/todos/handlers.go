package todos

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/views"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

// GET: /todos.html - Get all todos paginated.
func (s *service) HandleTodos(c *fiber.Ctx) error {
	var (
		filter  = c.Query("filter")
		todos   []database.Todo
		pending int
		count   int
		theme   string
	)

	theme = s.SessionGet(c, "theme", "light")
	if theme == "light" {
		s.SessionSet(c, "theme", "light")
	} else {
		s.SessionSet(c, "theme", "dark")
	}

	todos = fetchByFilter(s.DB, filter)
	pending = fetchPendingCount(s.DB)
	count = 0

	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	layoutProps, err := views.NewLayoutProps(views.WithCurrentUser(cu), views.WithTheme(theme))
	if err != nil {
		return c.Redirect("/login")
	}

	return views.Render(c, views.TodosPage(todos, count, pending, filter, layoutProps))
}

// GET: /todos.html - Get filtered todos.
func (s *service) HandleFilteredTodos(c *fiber.Ctx) error {
	var (
		filter         = c.Query("filter")
		allCount       int64
		completedCount int64
	)

	s.DB.Model(&database.Todo{}).Count(&allCount)
	s.DB.Model(&database.Todo{}).Where("completed = ?", true).Count(&completedCount)

	c.Set("HX-Trigger", "fetchTodos")

	return views.Render(c, views.TodosFooter(strconv.Itoa(int(allCount-completedCount)), filter))
}

func (s *service) HandleCreateTodo(c *fiber.Ctx) error {
	t := database.Todo{
		Content:   c.FormValue("todo"),
		UserID:    1,
		Completed: false,
	}
	res := s.DB.Create(&t)

	if res.Error != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Set("HX-Trigger", "refreshFooter")

	return views.Render(c, views.TodoCard(t))
}

// POST: /todos/:id/duplicate.html - Duplicates a todo
func (s *service) HandleDuplicateTodo(c *fiber.Ctx) error {
	var (
		id  = c.Params("id")
		src database.Todo
		dup database.Todo
	)

	src.UID = id
	if res := s.DB.First(&src); res.Error != nil {
		c.SendStatus(fiber.StatusMethodNotAllowed)
		return c.SendString("")
	}

	dup = src
	dup.UID = ""
	s.DB.Create(&dup)

	c.Set("HX-Trigger", "refreshFooter")

	return views.Render(c, views.TodoCard(dup))
}

// DELETE: /todos/:id.html - Delete a todo
func (s *service) HandleDeleteTodo(c *fiber.Ctx) error {
	var (
		uid  = c.Params("id")
		todo database.Todo
	)

	todo.UID = uid
	s.DB.Delete(&todo, "uid = ?", uid)

	c.Set("HX-Trigger", "refreshFooter")
	c.SendStatus(fiber.StatusOK)

	return c.SendString("")
}

func (s *service) HandleCheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	if res := s.DB.First(&t, "uid = ?", id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	c.Set("HX-Trigger", "refreshFooter")

	t.Completed = true
	s.DB.Save(&t)

	return views.Render(c, views.TodoContent(t))
}

func (s *service) HandleUncheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  database.Todo
	)

	if res := s.DB.First(&t, "uid = ?", id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	t.Completed = false
	s.DB.Save(&t)

	c.Set("HX-Trigger", "refreshFooter")

	return views.Render(c, views.TodoContent(t))
}

// Fragments
func (s *service) HandleFooterFragment(c *fiber.Ctx) error {
	var (
		filter         = c.Query("filter")
		allCount       int64
		completedCount int64
	)
	s.DB.Model(&database.Todo{}).Count(&allCount)
	s.DB.Model(&database.Todo{}).Where("completed = ?", true).Count(&completedCount)

	return views.Render(c, views.TodosFooter(strconv.Itoa(int(allCount-completedCount)), filter))
}

func (s *service) HandleTodosFragment(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		tds    []database.Todo
		left   int64
	)

	tds = fetchByFilter(s.DB, filter)
	s.DB.Model(&database.Todo{}).Where("completed = ?", false).Count(&left)

	return views.Render(c, views.TodosList(tds))
}

// API: /api/todos

// handleApiTodos returns all todos as JSON
// GET: /api/todos - Get all todos
func (s *service) HandleApiTodos(c *fiber.Ctx) error {
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
	type APIUser struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var tds []struct {
		User    APIUser `json:"user"`
		ID      string  `json:"id"`
		Content string  `json:"content"`
		UserID  uint    `json:"user_id"`
	}

	s.DB.
		Model(&database.Todo{}).
		Preload("User", func(DB *gorm.DB) *gorm.DB {
			return DB.Select("id", "name")
		}).
		Select("id, content, user_id").
		Where("user_id != ''").
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tds)

	return c.JSON(tds)
}

// handleApiUsers returns all users as JSON
// GET: /api/todos - Get all todos
func (s *service) HandleGetUsers(c *fiber.Ctx) error {
	var users []database.User

	s.DB.
		Model(&database.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []database.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// GET: /api/todos/Count - Count all todos
func (s *service) HandleCountTodos(c *fiber.Ctx) error {
	var count int64
	s.DB.Model(&database.Todo{}).Count(&count)
	res := struct {
		Count int64 `json:"count"`
	}{Count: count}

	return c.Status(fiber.StatusOK).JSON(res)
}

// handleCreate100000Todos creates 100000 todos
// GET: /api/todos/count
func (s *service) HandleCreateNTodos(c *fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil {
		n = 1000
	}

	userID, err := strconv.Atoi(c.Params("userID"))
	if err != nil {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	todos := []database.Todo{}

	// var sb strings.Builder
	for i := 1; i <= n; i++ {
		todos = append(todos, database.Todo{
			Content:   "content " + strconv.Itoa(i),
			UserID:    uint(userID),
			Completed: false,
		})
	}

	res := s.DB.CreateInBatches(&todos, 2000)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}

func (s *service) HandleEcho(c *fiber.Ctx) error {
	str := c.Params("str")

	return c.SendString(str)
}

func (s *service) HandleEchoQuery(c *fiber.Ctx) error {
	str := c.Query("str")

	return c.SendString(str)
}

func (s *service) HandleMakeUsers(c *fiber.Ctx) error {
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

	res := s.DB.Create(&users)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}
