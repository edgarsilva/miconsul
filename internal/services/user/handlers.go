package user

import (
	"errors"
	"miconsul/internal/model"
	view "miconsul/internal/views"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

// HandleIndexPage renders the admin users index page.
// GET: /admin/users
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "user/handlers:HandleIndexPage")
	defer span.End()

	users, err := s.FindRecentUsers(ctx, 10)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UsersIndexPage(vc, users))
}

// HandleEditPage renders the user edit page for admins.
// GET: /admin/users/:id
func (s *service) HandleEditPage(c fiber.Ctx) error {
	userID := c.Params("id", "")
	if userID == "" {
		return s.Redirect(c, "/")
	}
	user, err := s.TakeUserByID(c.Context(), userID)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/admin/users?toast=User does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/admin/users?toast=Failed to load user&level=error")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, user))
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

	input := userProfileUpdateInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=Invalid profile input&level=error", fiber.StatusBadRequest)
	}

	userUpds := input.toUserProfileUpdates()
	updatedUser, err := s.UpdateUserProfileByID(c.Context(), cu.ID, userUpds)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.respondWithRedirect(c, "/profile?toast=User does not exist&level=warning", fiber.StatusNotFound)
	}
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=Failed to update profile&level=error", fiber.StatusUnprocessableEntity)
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/profile?toast=Profile updated&level=success")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, updatedUser))
}

// HandleAPIUsers returns users as JSON for admin API clients.
// GET: /api/users
func (s *service) HandleAPIUsers(c fiber.Ctx) error {
	users, err := s.FindRecentUsers(c.Context(), 10)
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
	const maxMockUsers = 10000

	n, err := strconv.Atoi(c.Params("n", ""))
	if err != nil || n <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "n must be a positive integer",
		})
	}
	if n > maxMockUsers {
		n = maxMockUsers
	}

	users := []model.User{}
	for i := 0; i < n; i++ {
		users = append(users, model.User{
			Name:  strings.TrimSpace(faker.Name().Name()),
			Email: strings.ToLower(strings.TrimSpace(faker.Internet().Email())),
		})
	}

	err = s.CreateUsersInBatches(c.Context(), users, 1000)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "unprocessable entity",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"created": len(users),
	})
}

func (s *service) respondWithRedirect(c fiber.Ctx, redirectPath string, htmxStatus int) error {
	if s.NotHTMX(c) {
		return s.Redirect(c, redirectPath)
	}

	c.Set("HX-Location", redirectPath)
	return c.SendStatus(htmxStatus)
}
