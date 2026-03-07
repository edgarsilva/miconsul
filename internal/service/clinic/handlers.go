package clinic

import (
	"cmp"
	"errors"
	"strconv"
	"strings"

	"miconsul/internal/lib/amount"
	"miconsul/internal/lib/avatar"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
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

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicsPage(vc, clinics))
}

// HandleClinicsNewPage renders the new clinic HTML page
// GET: /clinics/new
func (s *service) HandleClinicsNewPage(c fiber.Ctx) error {
	clinic := model.Clinic{}
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsShowPage renders the clinics page HTML
// GET: /clinics/:id
func (s *service) HandleClinicsShowPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	id := strings.TrimSpace(c.Params("id", ""))
	if id == "" {
		return s.Redirect(c, "/clinics?toast=Failed to load clinic without ID&level=error")
	}

	clinic, err := s.TakeClinicByID(c.Context(), cu.ID, id)
	if errors.Is(err, ErrIDRequired) {
		return s.Redirect(c, "/clinics?toast=Failed to load clinic without ID&level=error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.Redirect(c, "/clinics?toast=Failed to load clinic&level=error")
	}

	vc, _ := view.NewCtx(c)
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
		return s.respondWithRedirect(c, "/clinics/new?toast=Invalid clinic input&level=error", fiber.StatusBadRequest)
	}
	clinic := input.toClinic("", cu.ID, price)

	err = s.CreateClinic(c.Context(), &clinic)
	if err != nil {
		return s.respondWithRedirect(c, "/clinics/new?toast=Failed to create clinic&level=error", fiber.StatusUnprocessableEntity)
	}

	s.attachClinicProfilePicBestEffort(c, cu.ID, &clinic)
	return s.respondWithClinicPage(c, clinic)
}

// HandleClinicsUpdate updates a clinic record for the CurrentUser
// PATCH: /clinics/:id
// POST: /clinics/:id/patch
func (s *service) HandleClinicsUpdate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinicID := cmp.Or(c.Params("id", ""), c.FormValue("id", ""))
	if clinicID == "" {
		return s.respondWithRedirect(c, "/clinics?toast=Can't update without an id&level=error", fiber.StatusBadRequest)
	}

	clinic, err := s.TakeClinicByID(c.Context(), cu.ID, clinicID)
	if errors.Is(err, ErrIDRequired) {
		return s.respondWithRedirect(c, "/clinics?toast=Can't update without an id&level=error", fiber.StatusBadRequest)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.respondWithRedirect(c, "/clinics?toast=Failed to load clinic&level=error", fiber.StatusInternalServerError)
	}

	input := clinicUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		return s.respondWithRedirect(c, "/clinics/"+clinicID+"?toast=Invalid clinic input&level=error", fiber.StatusBadRequest)
	}

	clinic = input.toClinic(clinic.ID, clinic.UserID, amount.StrToAmount(c.FormValue("price", "")))

	s.attachClinicProfilePicBestEffort(c, cu.ID, &clinic)

	err = s.UpdateClinicByID(c.Context(), cu.ID, clinicID, clinic)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.respondWithRedirect(c, "/clinics?toast=Clinic does not exist&level=warning", fiber.StatusNotFound)
	}
	if err != nil {
		return s.respondWithRedirect(c, "/clinics?toast=Failed to update clinic&level=error", fiber.StatusUnprocessableEntity)
	}

	return s.respondWithClinicPage(c, clinic)
}

// HandleClinicsDelete deletes a clinic record from the DB
// DELETE: /clinics/:id
// POST: /clinics/:id/delete
func (s *service) HandleClinicsDelete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	clinicID := strings.TrimSpace(c.Params("id", ""))
	if clinicID == "" {
		return s.respondWithRedirect(c, "/clinics?toast=Can't delete without an id&level=error", fiber.StatusBadRequest)
	}

	_, err := s.TakeClinicByID(c.Context(), cu.ID, clinicID)
	if errors.Is(err, ErrIDRequired) {
		return s.respondWithRedirect(c, "/clinics?toast=Can't delete without an id&level=error", fiber.StatusBadRequest)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return s.respondWithRedirect(c, "/clinics?toast=Failed to load clinic&level=error", fiber.StatusInternalServerError)
	}

	err = s.DeleteClinicByID(c.Context(), cu.ID, clinicID)
	if err != nil {
		return s.respondWithRedirect(c, "/clinics?toast=Failed to delete clinic&level=error", fiber.StatusUnprocessableEntity)
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/clinics")
	}

	c.Set("HX-Location", "/clinics")
	return c.SendStatus(fiber.StatusOK)
}

// HandleClinicsIndexSearch searches clinics and returns an HTML fragment to be
// replacesd in the HTMX active search
// GET: /clinics/search
func (s *service) HandleClinicsIndexSearch(c fiber.Ctx) error {
	searchTerm := strings.TrimSpace(c.Query("searchTerm", ""))
	if len(searchTerm) > 0 && len(searchTerm) < 3 {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	cu := s.CurrentUser(c)
	clinics, err := s.FindClinicsBySearchTerm(c.Context(), cu, searchTerm)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicsList(vc, clinics))
}

// HandleMockManyClinics creates many mock clinics for admin/testing flows.
// GET: /clinics/makeaton
func (s *service) HandleMockManyClinics(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	n, err := strconv.Atoi(c.Query("n", "100000"))
	if err != nil {
		n = 100000
	}

	var clinics []model.Clinic
	for i := 0; i <= n; i++ {
		ExtID := xid.New("prav")
		clinics = append(clinics, model.Clinic{
			ExtID:      ExtID,
			ProfilePic: avatar.DicebearShapeAvatarURL(ExtID),
			Name:       faker.Company().Name(),
			Email:      faker.Internet().Email(),
			Phone:      faker.PhoneNumber().CellPhone(),
			UserID:     cu.ID,
		})
	}

	result := s.DB.CreateInBatches(&clinics, 1000)
	if err := result.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendString("Rowsaffected:" + strconv.Itoa(int(result.RowsAffected)))
}

func (s *service) respondWithRedirect(c fiber.Ctx, redirectPath string, htmxStatus int) error {
	if s.NotHTMX(c) {
		return s.Redirect(c, redirectPath)
	}

	c.Set("HX-Location", redirectPath)
	return c.SendStatus(htmxStatus)
}

func (s *service) respondWithClinicPage(c fiber.Ctx, clinic model.Clinic) error {
	if s.NotHTMX(c) {
		return s.Redirect(c, "/clinics/"+clinic.ID)
	}

	c.Set("HX-Push-Url", "/clinics/"+clinic.ID)
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

func (s *service) attachClinicProfilePicBestEffort(c fiber.Ctx, userID string, clinic *model.Clinic) {
	path, picErr := SaveProfilePicToDisk(c, *clinic)
	if errors.Is(picErr, ErrProfilePicNotProvided) {
		return
	}
	if picErr != nil {
		log.Error(picErr)
		return
	}

	clinic.ProfilePic = path
	profilePicUpdate := model.Clinic{ProfilePic: path}
	err := s.UpdateClinicByID(c.Context(), userID, clinic.ID, profilePicUpdate)
	if err != nil {
		log.Error(err)
	}
}
