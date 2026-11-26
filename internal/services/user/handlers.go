// Package user provides handlers and services for managing users.
package user

import (
	"errors"
	"os"
	"strings"
	"time"

	"miconsul/internal/lib/profilepic"
	"miconsul/internal/models"
	"miconsul/internal/services/auth"
	view "miconsul/internal/views"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
)

const (
	localeCookieTTL  = 365 * 24 * time.Hour
	recentUsersLimit = 10
)

// HandleIndexPage renders the admin users index page.
// GET: /admin/users
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "user/handlers:HandleIndexPage")
	defer span.End()

	users, err := s.FindRecentUsers(ctx, recentUsersLimit)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.UserIndexPage(vc, users))
}

// HandleIndexSearch renders the admin users list filtered by the search term.
// GET: /admin/users/search
func (s *service) HandleIndexSearch(c fiber.Ctx) error {
	term := strings.TrimSpace(c.Query("searchTerm", ""))
	if models.SearchTermTooShort(term) {
		return view.Render(c, view.CmpSearchTooShortToast())
	}

	var users []models.User
	var err error
	if term == "" {
		users, err = s.FindRecentUsers(c.Context(), recentUsersLimit)
	} else {
		users, err = s.FindUsersBySearchTerm(c.Context(), term, recentUsersLimit)
	}
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.UserList(vc, users))
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
		return s.RedirectWithToast(c, "/admin/users", "User does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/admin/users", "Failed to load user", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.UserFormPage(vc, user))
}

// HandleProfilePage renders the current user's profile page.
// GET: /profile
func (s *service) HandleProfilePage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	user, err := s.TakeUserByUID(c.Context(), cu.UID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/profile", "User does not exist", view.LevelWarning)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.UserFormPage(vc, user))
}

// HandleUpdateProfile updates the current user's profile data. A new locale
// is also saved to the locale cookie, and HTMX requests reload the page so
// the whole layout switches language.
func (s *service) HandleUpdateProfile(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	input := userProfileUpdateInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/profile", "Invalid profile input", view.LevelError)
	}

	userUpds := input.toUserProfileUpdates()
	err = normalizeUserWriteInput(&userUpds)
	if errors.Is(err, ErrInvalidUser) {
		return s.respondWithUserToast(c, userWithUpdates(cu, userUpds), "Name can't be blank", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/profile", "Failed to update profile", view.LevelError)
	}

	path, picErr := SaveProfilePicToDisk(c, cu, s.Env.AssetsDir)
	if picErr != nil {
		if !errors.Is(picErr, profilepic.ErrNotProvided) {
			return s.RedirectWithToast(c, "/profile", "Failed to upload profile picture", view.LevelError)
		}
	} else {
		userUpds.ProfilePic = path
	}

	updatedUser, err := s.UpdateUserProfileByUID(c.Context(), cu.UID, userUpds)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/profile", "User does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/profile", "Failed to update profile", view.LevelError)
	}

	if err := auth.RefreshSessionUser(c, s, updatedUser); err != nil {
		log.Warn("refresh signed-in user after profile update:", err)
	}

	if locale := userUpds.Locale; locale != "" && locale != s.CurrentLocale(c) {
		c.Cookie(s.NewCookie("locale", locale, localeCookieTTL))
		c.Set("HX-Refresh", "true")
	}

	vc := view.NewCtx(c)
	return view.Render(c, templ.Join(
		view.UserProfileSection(vc, updatedUser),
		view.CmpToastPartial("Profile updated", "", view.LevelSuccess),
	))
}

// HandleProfilePicPreview stores and renders a temporary profile picture preview.
// POST: /profile/avatar/preview
func (s *service) HandleProfilePicPreview(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	user, err := s.TakeUserByUID(c.Context(), cu.UID)
	if err != nil {
		return s.RedirectWithToast(c, "/profile", "User does not exist", view.LevelError)
	}

	previewPath, err := SaveProfilePicPreviewToTmp(c, user, s.Env.AssetsDir)
	if err != nil {
		return s.RedirectWithToast(c, "/profile", "Failed to display profile picture", view.LevelError)
	}

	user.ProfilePic = previewPath
	vc := view.NewCtx(c)
	return view.Render(c, view.UserAvatar(vc, user))
}

// HandleProfilePic serves a user's profile picture file.
// GET: /profile/avatar and /profile/avatar/preview
func (s *service) HandleProfilePic(c fiber.Ctx) error {
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

	_, err := s.RemoveProfilePicByUID(c.Context(), cu.UID)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/profile", "User does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/profile", "Failed to remove profile picture", view.LevelError)
	}

	cu.ProfilePic = ""
	if err := auth.RefreshSessionUser(c, s, cu); err != nil {
		log.Warn("refresh signed-in user after profile picture removal:", err)
	}

	return s.RedirectWithToast(c, "/profile", "Profile picture removed", view.LevelSuccess)
}

// HandleAdminRemoveProfilePic removes a user's profile picture as admin.
// PATCH: /admin/users/:id/removepic
func (s *service) HandleAdminRemoveProfilePic(c fiber.Ctx) error {
	userID := c.Params("id", "")

	updatedUser, err := s.RemoveProfilePicByUID(c.Context(), userID)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/admin/users", "User does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/admin/users", "Failed to remove profile picture", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, templ.Join(
		view.UserAvatar(vc, updatedUser),
		view.CmpToastPartial("Profile picture removed", "", view.LevelSuccess),
	))
}

// HandleAdminUpdateUser updates another user's name, email and phone as admin.
// PATCH: /admin/users/:id
func (s *service) HandleAdminUpdateUser(c fiber.Ctx) error {
	userID := c.Params("id", "")

	input := adminUserUpdateInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/admin/users/"+userID, "Invalid user input", view.LevelError)
	}

	userUpds := input.toUserUpdates()
	updatedUser, err := s.UpdateUserProfileByUID(c.Context(), userID, userUpds)
	if errors.Is(err, ErrInvalidUser) {
		target, takeErr := s.TakeUserByUID(c.Context(), userID)
		if takeErr != nil {
			return s.RedirectWithToast(c, "/admin/users", "User does not exist", view.LevelWarning)
		}
		return s.respondWithUserToast(c, userWithUpdates(target, userUpds), "Name can't be blank", view.LevelWarning)
	}
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/admin/users", "User does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/admin/users/"+userID, "Failed to update user", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, templ.Join(
		view.UserProfileSection(vc, updatedUser),
		view.CmpToastPartial("User updated", "", view.LevelSuccess),
	))
}

func (s *service) respondWithUserToast(c fiber.Ctx, user models.User, message string, level view.Level) error {
	vc := view.NewCtx(c)
	return view.Render(c, templ.Join(
		view.UserProfileSection(vc, user),
		view.CmpToastPartial(message, "", level),
	))
}

func userWithUpdates(user, updates models.User) models.User {
	user.Name = updates.Name
	user.Email = updates.Email
	user.Phone = updates.Phone
	if updates.Locale != "" {
		user.Locale = updates.Locale
	}
	user.SetFieldError("name", "Name can't be blank")

	return user
}
