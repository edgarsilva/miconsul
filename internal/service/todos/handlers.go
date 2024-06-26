package todos

import (
	"strconv"

	"miconsul/internal/model"
	"miconsul/internal/view"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GET: /todos.html - Get all todos paginated.
func (s *service) HandleTodos(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	theme := s.SessionUITheme(c)

	Ctx, err := view.NewCtx(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	if err != nil {
		return c.Redirect("/login")
	}

	filter := c.Query("filter")
	todos := s.fetchByFilter(filter)
	pending := s.pendingTodosCount()
	count := s.todosCount()

	return view.Render(c, view.PageTodos(todos, count, pending, filter, Ctx))
}

// GET: /todos.html - Get filter todos.
func (s *service) HandleFilterTodos(c *fiber.Ctx) error {
	var (
		filter         = c.Query("filter")
		allCount       int
		completedCount int
		pendingCount   int
	)

	c.Set("HX-Trigger", "fetchTodos")

	allCount = s.todosCount()
	completedCount = s.completedCount()
	pendingCount = allCount - completedCount

	return view.Render(c, view.TodosFooter(allCount, pendingCount, filter))
}

// HandleCreateTodo adds a new todo for the CurrentUser to the DB
func (s *service) HandleCreateTodo(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	todo := model.Todo{
		Content:   c.FormValue("todo"),
		UserID:    cu.ID,
		Completed: false,
	}

	result := s.DB.Model(&todo).Create(&todo)
	if result.Error != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Set("HX-Trigger", "syncFooter")

	return view.Render(c, view.TodoCard(todo))
}

// POST: /todos/:id/duplicate.html - Duplicates a todo
func (s *service) HandleDuplicateTodo(c *fiber.Ctx) error {
	var (
		id  = c.Params("id")
		src model.Todo
	)

	src.ID = id
	if res := s.DB.First(&src); res.Error != nil {
		c.SendStatus(fiber.StatusMethodNotAllowed)
		return c.SendString("")
	}

	dup := src
	dup.ID = ""
	s.DB.Create(&dup)

	c.Set("HX-Trigger", "syncFooter")

	return view.Render(c, view.TodoCard(dup))
}

// DELETE: /todos/:id.html - Delete a todo
func (s *service) HandleDeleteTodo(c *fiber.Ctx) error {
	var (
		id   = c.Params("id")
		todo model.Todo
	)

	todo.ID = id
	s.DB.Delete(&todo, "id = ?", id)

	c.Set("HX-Trigger", "syncFooter")
	c.SendStatus(fiber.StatusOK)

	return c.SendString("")
}

func (s *service) HandleCheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  model.Todo
	)

	if res := s.DB.First(&t, "id = ?", id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	c.Set("HX-Trigger", "syncFooter")

	t.Completed = true
	s.DB.Save(&t)

	return view.Render(c, view.TodoContent(t))
}

func (s *service) HandleUncheckTodo(c *fiber.Ctx) error {
	var (
		id = c.Params("id")
		t  model.Todo
	)

	if res := s.DB.First(&t, "id = ?", id); res.Error != nil {
		return c.SendStatus(fiber.StatusMethodNotAllowed)
	}

	t.Completed = false
	s.DB.Save(&t)

	c.Set("HX-Trigger", "syncFooter")

	return view.Render(c, view.TodoContent(t))
}

// Fragments
func (s *service) HandleFooterFragment(c *fiber.Ctx) error {
	var (
		filter         = c.Query("filter")
		allCount       int
		completedCount int
		pendingCount   int
	)

	allCount = s.todosCount()
	completedCount = s.completedCount()
	pendingCount = allCount - completedCount

	return view.Render(c, view.TodosFooter(allCount, pendingCount, filter))
}

func (s *service) HandleTodosFragment(c *fiber.Ctx) error {
	var (
		filter = c.Query("filter")
		tds    []model.Todo
		left   int64
	)

	tds = s.fetchByFilter(filter)
	s.DB.Model(&model.Todo{}).Where("completed = ?", false).Count(&left)

	return view.Render(c, view.TodosList(tds))
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

	// var tds []model.Todo
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
		Model(&model.Todo{}).
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

// GET: /api/todos/Count - Count all todos
func (s *service) HandleCountTodos(c *fiber.Ctx) error {
	var count int64
	s.DB.Model(&model.Todo{}).Count(&count)
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

	userID := c.Params("userID", "")
	if err != nil {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	todos := []model.Todo{}

	for i := 1; i <= n; i++ {
		todos = append(todos, model.Todo{
			Content:   "content " + strconv.Itoa(i),
			UserID:    userID,
			Completed: false,
		})
	}

	res := s.DB.CreateInBatches(&todos, 2000)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}
