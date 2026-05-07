// Package user provides handlers and services for managing users.
package user

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"miconsul/internal/models"
	view "miconsul/internal/views"

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
	user, err := s.TakeUserByUID(c.Context(), userID)
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
	user, err := s.TakeUserByUID(c.Context(), cu.UID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/profile?toast=User does not exist&level=warning")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, user))
}

// HandleUpdateProfile updates the current user's profile data.
// POST: /profile
func (s *service) HandleUpdateProfile(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	input := userProfileUpdateInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=Invalid profile input&level=error")
	}

	userUpds := input.toUserProfileUpdates()
	path, picErr := SaveProfilePicToDisk(c, cu, s.Env.AssetsDir)
	if picErr != nil {
		if !errors.Is(picErr, ErrProfilePicNotProvided) {
			return s.respondWithRedirect(c, "/profile?toast=Failed to upload profile picture&level=error")
		}
	} else {
		userUpds.ProfilePic = path
	}

	updatedUser, err := s.UpdateUserProfileByUID(c.Context(), cu.UID, userUpds)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.respondWithRedirect(c, "/profile?toast=User does not exist&level=warning")
	}
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=Failed to update profile&level=error")
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/profile?toast=Profile updated&level=success")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.UserEditPage(vc, updatedUser))
}

// HandleProfilePicPreview stores and renders a temporary profile picture preview.
// POST: /profile/avatar/preview
func (s *service) HandleProfilePicPreview(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	user, err := s.TakeUserByUID(c.Context(), cu.UID)
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=User does not exist&level=error")
	}

	previewPath, err := SaveProfilePicPreviewToTmp(c, user, s.Env.AssetsDir)
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=Failed to display profile picture&level=error")
	}

	user.ProfilePic = previewPath
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ProfileAvatarPic(vc, user))
}

// HandleUserProfilePicImgSrc serves a user's profile picture file.
// GET: /users/:id/profilepic/:filename
func (s *service) HandleUserProfilePicImgSrc(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	isPreview := strings.HasSuffix(c.Path(), "preview")
	sufix := "avatar"
	if isPreview {
		sufix = "preview"
	}

	filename := cu.UID + "_" + sufix
	picPath, err := ProfilePicPath(filename, s.Env.AssetsDir)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	fileInfo, err := os.Stat(picPath)
	if errors.Is(err, os.ErrNotExist) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if fileInfo.IsDir() {
		return c.SendStatus(fiber.StatusNotFound)
	}

	return s.SendFile(c, picPath)
}

// HandleRemoveProfilePic removes the current user's profile picture.
// PATCH: /profile/removepic
func (s *service) HandleRemoveProfilePic(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	_, err := gorm.G[models.User](s.GormDB()).Where("uid = ?", cu.UID).Update(c.Context(), "profile_pic", "")
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.respondWithRedirect(c, "/profile?toast=User does not exist&level=warning")
	}
	if err != nil {
		return s.respondWithRedirect(c, "/profile?toast=Failed to remove profile picture&level=error")
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/profile?toast=Profile picture removed&level=success")
	}

	return s.respondWithRedirect(c, "/profile?toast=Profile picture removed&level=success")
}

// HandleAdminRemoveProfilePic removes a user's profile picture as admin.
// PATCH: /admin/users/:id/removepic
func (s *service) HandleAdminRemoveProfilePic(c fiber.Ctx) error {
	userID := c.Params("id", "")

	updatedUser, err := s.UpdateUserProfileByUID(c.Context(), userID, models.User{ProfilePic: ""})
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.respondWithRedirect(c, "/admin/users?toast=User does not exist&level=warning")
	}
	if err != nil {
		return s.respondWithRedirect(c, "/admin/users?toast=Failed to remove profile picture&level=error")
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/admin/users/"+updatedUser.UID+"?toast=Profile picture removed&level=success")
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

	res := struct{ Users []models.User }{
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

	users := []models.User{}
	for i := 0; i < n; i++ {
		users = append(users, models.User{
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

func (s *service) respondWithRedirect(c fiber.Ctx, redirectPath string) error {
	if s.NotHTMX(c) {
		return s.Redirect(c, redirectPath)
	}

	c.Set("HX-Redirect", redirectPath)
	return c.SendStatus(fiber.StatusNoContent)
}
