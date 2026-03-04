package user

import (
	"miconsul/internal/model"
	"miconsul/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

// HandleIndexPage renders the admin users index page.
// GET: /admin/users
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "user/handlers:HandleIndexPage")
	defer span.End()

	users := []model.User{}
	users, _ = gorm.G[model.User](s.DB.GormDB()).Order("id DESC").Limit(10).Find(ctx)

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UsersIndexPage(vc, users))
}

// HandleEditPage renders the user edit page for admins.
// GET: /admin/users/:id
func (s *service) HandleEditPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	userID := c.Params("id", "")
	if userID == "" {
		return s.Redirect(c, "/")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, cu))
}

// HandleProfilePage renders the current user's profile page.
// GET: /profile
func (s *service) HandleProfilePage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, cu))
}

// HandleUpdateProfile updates the current user's profile data.
// POST: /profile
func (s *service) HandleUpdateProfile(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	userUpds := model.User{}
	c.Bind().Body(&userUpds)
	_, err := gorm.G[model.User](s.DB.GormDB()).Where("id = ?", cu.ID).Updates(c.Context(), userUpds)
	if err != nil {
		redirectPath := "/profile?err=failed to update profile&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, userUpds))
}

// HandleAPIUsers returns users as JSON for admin API clients.
// GET: /api/users
func (s *service) HandleAPIUsers(c fiber.Ctx) error {
	users, err := gorm.G[model.User](s.DB.GormDB()).Limit(10).Find(c.Context())
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	res := struct{ Users []model.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
}

// HandleAPIMakeUsers creates mock users for admin testing.
// POST: /api/users/make/:n
func (s *service) HandleAPIMakeUsers(c fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n", ""))
	if err != nil || n <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "n must be a positive integer",
		})
	}

	var users []model.User
	for i := 0; i <= n; i++ {
		users = append(users, model.User{
			Name:  faker.Name().Name(),
			Email: faker.Internet().Email(),
		})
	}

	err = gorm.G[model.User](s.DB.GormDB()).CreateInBatches(c.Context(), &users, 1000)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "unprocessable entity",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"created": len(users),
	})
}
