package user

import (
	"miconsul/internal/model"
	"miconsul/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"syreclabs.com/go/faker"
)

// HandleIndexPage list all users in a table *the index*
//
// GET: /admin/users
func (s *service) HandleIndexPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil || cu.Role != model.UserRoleAdmin {
		return c.Redirect("/login")
	}

	users := []model.User{}
	s.DB.Order("id DESC").Limit(10).Find(&users)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	return view.Render(c, view.UserIndexPage(vc, users))
}

// HandleProfilePage list all users in a table *the index*
//
// GET: /profile
func (s *service) HandleProfilePage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil || cu.Role != model.UserRoleAdmin {
		return c.Redirect("/login")
	}

	users := []model.User{}
	s.DB.Order("id DESC").Limit(10).Find(&users)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithCurrentUser(cu), view.WithTheme(theme))
	return view.Render(c, view.UserIndexPage(vc, users))
}

// handleApiUsers returns all users as JSON
//
// GET: /api/todos - Get all todos
func (s *service) HandleGetUsers(c *fiber.Ctx) error {
	var users []model.User

	s.DB.
		Model(&model.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []model.User }{
		Users: users,
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (s *service) HandleMakeUsers(c *fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil {
		n = 10
	}

	var users []model.User
	for i := 0; i <= n; i++ {
		users = append(users, model.User{
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

// API
//
// handleAPIUsers returns all users as JSON
// GET: /api/todos - Get all todos
func (s *service) HandleAPIUsers(c *fiber.Ctx) error {
	var users []model.User

	s.DB.
		Model(&model.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []model.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
}
