package clinic

import (
	"cmp"
	"errors"
	"strings"

	"miconsul/internal/lib/amount"
	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
)

const (
	QUERY_LIMIT int = 10
)

// HandleIndexPage renders the clinics index page.
// GET: /clinics
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinics, err := s.FindClinicsBySearchTerm(c.Context(), cu, "")
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicIndexPage(vc, clinics))
}

// HandleNewPage renders the new clinic HTML page
// GET: /clinics/new
func (s *service) HandleNewPage(c fiber.Ctx) error {
	clinic := models.Clinic{}
	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleEditPage renders the clinic edit page.
// GET: /clinics/:id
func (s *service) HandleEditPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	id := strings.TrimSpace(c.Params("id", ""))
	if id == "" {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic without ID", view.LevelError)
	}

	clinic, err := s.TakeClinicByID(c.Context(), cu.ID, id)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic without ID", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleCreate inserts a new clinic record for the given user
// POST: /clinics
func (s *service) HandleCreate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	price := amount.StrToAmount(c.FormValue("price", ""))
	input := clinicUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics/new", "Invalid clinic input", view.LevelError)
	}
	clinic := input.toClinic(0, "", cu.ID, price)

	err = s.CreateClinic(c.Context(), &clinic)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics/new", "Failed to create clinic", view.LevelError)
	}

	s.attachClinicProfilePicBestEffort(c, cu.ID, &clinic)
	return s.respondWithClinicPage(c, clinic)
}

// HandleUpdate updates a clinic record for the CurrentUser
// PATCH: /clinics/:id
func (s *service) HandleUpdate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinicID := cmp.Or(c.Params("id", ""), c.FormValue("id", ""))
	if clinicID == "" {
		return s.RedirectWithToast(c, "/clinics", "Can't update without an id", view.LevelError)
	}

	clinic, err := s.TakeClinicByID(c.Context(), cu.ID, clinicID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/clinics", "Can't update without an id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic", view.LevelError)
	}

	input := clinicUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics/"+clinicID, "Invalid clinic input", view.LevelError)
	}

	clinic = input.toClinic(clinic.ID, clinic.UID, clinic.UserID, amount.StrToAmount(c.FormValue("price", "")))

	s.attachClinicProfilePicBestEffort(c, cu.ID, &clinic)

	err = s.UpdateClinicByID(c.Context(), cu.ID, clinicID, clinic)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/clinics", "Clinic does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to update clinic", view.LevelError)
	}

	return s.respondWithClinicPage(c, clinic)
}

// HandleDelete deletes a clinic record from the DB
// DELETE: /clinics/:id
func (s *service) HandleDelete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinicID := strings.TrimSpace(c.Params("id", ""))
	if clinicID == "" {
		return s.RedirectWithToast(c, "/clinics", "Can't delete without an id", view.LevelError)
	}

	_, err := s.TakeClinicByID(c.Context(), cu.ID, clinicID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/clinics", "Can't delete without an id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic", view.LevelError)
	}

	err = s.DeleteClinicByID(c.Context(), cu.ID, clinicID)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to delete clinic", view.LevelError)
	}

	return s.HXRedirect(c, "/clinics")
}

// HandleIndexSearch searches clinics and returns an HTML fragment to be
// replacesd in the HTMX active search
// GET: /clinics/search
func (s *service) HandleIndexSearch(c fiber.Ctx) error {
	searchTerm := strings.TrimSpace(c.Query("searchTerm", ""))
	if models.SearchTermTooShort(searchTerm) {
		return view.Render(c, view.CmpSearchTooShortToast())
	}

	cu := s.CurrentUser(c)
	clinics, err := s.FindClinicsBySearchTerm(c.Context(), cu, searchTerm)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicList(vc, clinics))
}

func (s *service) respondWithClinicPage(c fiber.Ctx, clinic models.Clinic) error {
	c.Set("HX-Push-Url", "/clinics/"+clinic.UID)
	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicProfileSection(vc, clinic))
}

func (s *service) attachClinicProfilePicBestEffort(c fiber.Ctx, userID uint, clinic *models.Clinic) {
	path, picErr := SaveProfilePicToDisk(c, *clinic)
	if errors.Is(picErr, ErrProfilePicNotProvided) {
		return
	}
	if picErr != nil {
		log.Error(picErr)
		return
	}

	clinic.ProfilePic = path
	profilePicUpdate := models.Clinic{ProfilePic: path}
	err := s.UpdateClinicByID(c.Context(), userID, clinic.UID, profilePicUpdate)
	if err != nil {
		log.Error(err)
	}
}
