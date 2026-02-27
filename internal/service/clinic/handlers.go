package clinic

import (
	"cmp"
	"errors"
	"miconsul/internal/lib/handlerutils"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

const (
	QUERY_LIMIT int = 10
)

// HandleClinicsPage renders the clinics page HTML
//
//	GET: /clinics
func (s *service) HandleClinicsIndexPage(c fiber.Ctx) error {
	clinics, err := s.FindClinicsByTerm(c, "")
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicsPage(vc, clinics))
}

// HandleClinicsNewPage renders the new clinic HTML page
//
//	GET: /clinics/new
func (s *service) HandleClinicsNewPage(c fiber.Ctx) error {
	clinic := model.Clinic{}
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsShowPage renders the clinics page HTML
//
//	GET: /clinics/:id
func (s *service) HandleClinicsShowPage(c fiber.Ctx) error {
	id := c.Params("id", "")
	if id == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics?err=failed to load Clinic without ID")
	}

	clinic, err := s.TakeClinicByID(c, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsCreate inserts a new clinic record for the given user
//
//	POST: /clinics
func (s *service) HandleClinicsCreate(c fiber.Ctx) error {
	cu, _ := s.CurrentUser(c)
	clinic := model.Clinic{
		UserID: cu.ID,
		Price:  handlerutils.StrToAmount(c.FormValue("price", "")),
	}

	c.Bind().Body(&clinic)

	result := s.DB.Create(&clinic)
	if result.Error == nil {
		path, err := SaveProfilePicToDisk(c, clinic)
		if err == nil {
			clinic.ProfilePic = path
		}
	}

	if s.NotHTMX(c) {
		if result.Error != nil {
			return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics?err=failed to create Clinic")
		}

		return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics/" + clinic.ID)
	}

	c.Set("HX-Push-Url", "/clinics/"+clinic.ID)
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsUpdate updates a clinic record for the CurrentUser
//
//	PATCH: /clinics/:id
//	POST: /clinics/:id/patch
func (s *service) HandleClinicsUpdate(c fiber.Ctx) error {
	clinicID := cmp.Or(c.Params("id", ""), c.FormValue("id", ""))
	if clinicID == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics?msg=can't update without an id")
	}

	clinic, err := s.TakeClinicByID(c, clinicID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}

	// If found parse the form values into the model struct
	c.Bind().Body(&clinic)
	clinic.Price = handlerutils.StrToAmount(c.FormValue("price", ""))

	path, err := SaveProfilePicToDisk(c, clinic)
	if err == nil {
		clinic.ProfilePic = path
	}

	result := s.DB.Model(&clinic).Where("user_id = ?", clinic.UserID).Updates(&clinic)

	if s.NotHTMX(c) {
		if result.Error != nil {
			return c.Redirect().To("/clinics?err=failed to update clinic")
		}

		return c.Redirect().To("/clinics/" + clinic.ID)
	}

	c.Set("HX-Push-Url", "/clinics/"+clinicID)
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleClinicsDelete deletes a clinic record from the DB
//
// DELETE: /clinics/:id
// POST: /clinics/:id/delete
func (s *service) HandleClinicsDelete(c fiber.Ctx) error {
	clinicID := c.Params("ID", "")
	if clinicID == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics?msg=can't delete without an id")
	}

	clinic, err := s.TakeClinicByID(c, clinicID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.ErrNotFound
	}

	res := s.DB.Model(&clinic).Where("user_id = ?", clinic.UserID).Delete(&clinic)
	if err := res.Error; err != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics?msg=failed to delete that clinic")
	}

	if s.NotHTMX(c) {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/clinics")
	}

	c.Set("HX-Location", "/clinics")
	return c.SendStatus(fiber.StatusOK)
}

// HandleClinicIndexSearch searches patients and returns an HTML fragment to be
// replacesd in the HTMX active search
//
// POST: /clinics/search
func (s *service) HandleClinicsIndexSearch(c fiber.Ctx) error {
	term := c.Query("term", "")
	if len(term) > 0 && len(term) < 3 {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	clinics, err := s.FindClinicsByTerm(c, term)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ClinicsList(vc, clinics))
}

func (s *service) HandleMockManyClinics(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	n, err := strconv.Atoi(c.Query("n", "100000"))
	if err != nil {
		n = 100000
	}

	var clinics []model.Clinic
	for i := 0; i <= n; i++ {
		ExtID := xid.New("prav")
		clinics = append(clinics, model.Clinic{
			ExtID:      ExtID,
			ProfilePic: handlerutils.DicebearShapeAvatarURL(ExtID),
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
