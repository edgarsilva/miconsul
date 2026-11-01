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

// HandleClinicsIndexPage renders the clinics index page.
// GET: /clinics
func (s *service) HandleClinicsIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinics, err := s.FindClinicsBySearchTerm(c.Context(), cu, "")
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicIndexPage(vc, clinics))
}

// HandleClinicsNewPage renders the new clinic HTML page
// GET: /clinics/new
func (s *service) HandleClinicsNewPage(c fiber.Ctx) error {
	clinic := models.Clinic{}
	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsShowPage renders the clinics page HTML
// GET: /clinics/:id
func (s *service) HandleClinicsShowPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	id := strings.TrimSpace(c.Params("id", ""))
	if id == "" {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic without ID", "error")
	}

	clinic, err := s.TakeClinicByID(c.Context(), cu.ID, id)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic without ID", "error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic", "error")
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsCreate inserts a new clinic record for the given user
// POST: /clinics
func (s *service) HandleClinicsCreate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	price := amount.StrToAmount(c.FormValue("price", ""))
	input := clinicUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics/new", "Invalid clinic input", "error")
	}
	clinic := input.toClinic(0, "", cu.ID, price)

	err = s.CreateClinic(c.Context(), &clinic)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics/new", "Failed to create clinic", "error")
	}

	s.attachClinicProfilePicBestEffort(c, cu.ID, &clinic)
	return s.respondWithClinicPage(c, clinic)
}

// HandleClinicsUpdate updates a clinic record for the CurrentUser
// PATCH: /clinics/:id
func (s *service) HandleClinicsUpdate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinicID := cmp.Or(c.Params("id", ""), c.FormValue("id", ""))
	if clinicID == "" {
		return s.RedirectWithToast(c, "/clinics", "Can't update without an id", "error")
	}

	clinic, err := s.TakeClinicByID(c.Context(), cu.ID, clinicID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/clinics", "Can't update without an id", "error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic", "error")
	}

	input := clinicUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics/"+clinicID, "Invalid clinic input", "error")
	}

	clinic = input.toClinic(clinic.ID, clinic.UID, clinic.UserID, amount.StrToAmount(c.FormValue("price", "")))

	s.attachClinicProfilePicBestEffort(c, cu.ID, &clinic)

	err = s.UpdateClinicByID(c.Context(), cu.ID, clinicID, clinic)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/clinics", "Clinic does not exist", "warning")
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to update clinic", "error")
	}

	return s.respondWithClinicPage(c, clinic)
}

// HandleClinicsDelete deletes a clinic record from the DB
// DELETE: /clinics/:id
func (s *service) HandleClinicsDelete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinicID := strings.TrimSpace(c.Params("id", ""))
	if clinicID == "" {
		return s.RedirectWithToast(c, "/clinics", "Can't delete without an id", "error")
	}

	_, err := s.TakeClinicByID(c.Context(), cu.ID, clinicID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/clinics", "Can't delete without an id", "error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to load clinic", "error")
	}

	err = s.DeleteClinicByID(c.Context(), cu.ID, clinicID)
	if err != nil {
		return s.RedirectWithToast(c, "/clinics", "Failed to delete clinic", "error")
	}

	return s.HXRedirect(c, "/clinics")
}

// HandleClinicsIndexSearch searches clinics and returns an HTML fragment to be
// replacesd in the HTMX active search
// GET: /clinics/search
func (s *service) HandleClinicsIndexSearch(c fiber.Ctx) error {
	searchTerm := strings.TrimSpace(c.Query("searchTerm", ""))
	if len(searchTerm) > 0 && len(searchTerm) < 3 {
		return view.Render(c, view.CmpSearchTooShortToast(3))
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
