package user

import (
	"miconsul/internal/model"
	"miconsul/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

// HandleIndexPage list all users in a table *the index*
//
// GET: /admin/users
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	ctx, span := s.Tracer.Start(c.Context(), "user/handlers:HandleIndexPage")
	defer span.End()

	cu, err := s.CurrentUser(c)
	if err != nil || cu.Role != model.UserRoleAdmin {
		return c.Redirect().To("/")
	}

	users := []model.User{}
	users, _ = gorm.G[model.User](s.DB.DB).Order("id DESC").Limit(10).Find(ctx)

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UsersIndexPage(vc, users))
}

// HandleEditPage shows the edit/new form for users
//
// GET: /admin/users/:id
func (s *service) HandleEditPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil || cu.Role != model.UserRoleAdmin {
		return c.Redirect().To("/")
	}

	userID := c.Params("id", "")
	if userID == "" {
		return c.Redirect().To("/")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, cu))
}

// HandleProfilePage show the CurrentUser profile page
//
// GET: /profile
func (s *service) HandleProfilePage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, cu))
}

// HandleProfilePage show the CurrentUser profile page
//
// GET: /profile
func (s *service) HandleUpdateProfile(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/")
	}

	userUpds := model.User{}
	c.Bind().Body(&userUpds)
	_, err = gorm.G[model.User](s.DB.DB).Where("id = ?", cu.ID).Updates(c.Context(), userUpds)
	if err != nil {
		redirectPath := "/profile?err=failed to update profile&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect().To(redirectPath)
		}
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, userUpds))
}

// handleApiUsers returns all users as JSON
//
// GET: /api/todos - Get all todos
func (s *service) HandleGetUsers(c fiber.Ctx) error {
	users, _ := gorm.G[model.User](s.DB.DB).Limit(10).Find(c.Context())

	res := struct{ Users []model.User }{
		Users: users,
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (s *service) HandleMakeUsers(c fiber.Ctx) error {
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

	err = gorm.G[model.User](s.DB.DB).CreateInBatches(c.Context(), &users, 1000)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}

// API
//
// handleAPIUsers returns all users as JSON
// GET: /api/todos - Get all todos
func (s *service) HandleAPIUsers(c fiber.Ctx) error {
	users, _ := gorm.G[model.User](s.DB.DB).Limit(10).Find(c.Context())

	res := struct{ Users []model.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
}
