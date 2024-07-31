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
		return c.Redirect("/")
	}

	users := []model.User{}
	s.DB.Order("id DESC").Limit(10).Find(&users)

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UsersIndexPage(vc, users))
}

// HandleEditPage shows the edit/new form for users
//
// GET: /admin/users/:id
func (s *service) HandleEditPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil || cu.Role != model.UserRoleAdmin {
		return c.Redirect("/")
	}

	userID := c.Params("id", "")
	if userID == "" {
		return c.Redirect("/")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, cu))
}

// HandleProfilePage show the CurrentUser profile page
//
// GET: /profile
func (s *service) HandleProfilePage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, cu))
}

// HandleProfilePage show the CurrentUser profile page
//
// GET: /profile
func (s *service) HandleUpdateProfile(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/")
	}

	userUpds := model.User{}
	c.BodyParser(&userUpds)
	result := s.DB.Where("id = ?", cu.ID).Updates(&userUpds)
	if err := result.Error; err != nil {
		redirectPath := "/profile?err=failed to update profile&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath)
		}
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, userUpds))
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
