package clinic

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/common"
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
	"syreclabs.com/go/faker"
)

// HandleClinicsPage renders the clinics page HTML
//
// GET: /clinics
func (s *service) HandleClinicsPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	clinics := []model.Clinic{}
	s.DB.Model(&model.Clinic{}).Limit(20).Find(&clinics)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.ClinicsPage(vc, clinics))
}

// HandleClinicPage renders the clinics page HTML
//
// GET: /clinics/:id
func (s *service) HandleClinicPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	id := c.Params("id", "")
	clinic := model.Clinic{}
	clinic.ID = id
	if id != "" && id != "new" {
		s.DB.Model(&model.Clinic{}).First(&clinic)
	}
	clinics := []model.Clinic{}
	s.DB.Model(&model.Clinic{}).Limit(20).Find(&clinics)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleCreateClinic inserts a new clinic record for the given user
//
// POST: /clinics
func (s *service) HandleCreateClinic(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		c.Redirect("/login")
	}

	clinic := model.Clinic{
		UserID: cu.ID,
		Price:  common.StrToAmount(c.FormValue("price", "")),
	}
	c.BodyParser(&clinic)

	result := s.DB.Create(&clinic)
	if err := result.Error; err == nil {
		path, err := SaveProfilePicToDisk(c, clinic)
		if err == nil {
			clinic.ProfilePic = path
		}
	}
	if !s.IsHTMX(c) {
		if err := result.Error; err != nil {
			return c.Redirect("/clinics?err=failed to create Clinic", fiber.StatusSeeOther)
		}

		return c.Redirect("/clinics/"+clinic.ID, fiber.StatusSeeOther)
	}

	c.Set("HX-Push-Url", "/clinics/"+clinic.ID)
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.ClinicPage(vc, clinic))
}

// HandleUpdateClinic updates a clinic record for the CurrentUser
//
// PATCH: /clinics/:id
// POST: /clinics/:id/patch
func (s *service) HandleUpdateClinic(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	clinicID := c.Params("ID", "")
	if clinicID == "" {
		clinicID = c.FormValue("id", "")
	}

	if clinicID == "" {
		return c.Redirect("/clinics?msg=can't update without an id", fiber.StatusSeeOther)
	}

	clinic := model.Clinic{
		UserID: cu.ID,
		Price:  common.StrToAmount(c.FormValue("price", "")),
	}
	c.BodyParser(&clinic)
	path, err := SaveProfilePicToDisk(c, clinic)
	if err == nil {
		clinic.ProfilePic = path
	}

	res := s.DB.Where("id = ? AND user_id = ?", clinicID, cu.ID).Updates(&clinic)
	if err := res.Error; err != nil {
		return c.Redirect("/clinics?err=failed to update clinic")
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect("/clinics/" + clinic.ID)
	}

	return c.Redirect("/clinics/"+clinicID, fiber.StatusSeeOther)
}

// HandleDeleteClinic deletes a clinic record from the DB
//
// DELETE: /clinics/:id
// POST: /clinics/:id/delete
func (s *service) HandleDeleteClinic(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	clinicID := c.Params("ID", "")
	if clinicID == "" {
		return c.Redirect("/clinics?msg=can't delete without an id", fiber.StatusSeeOther)
	}

	clinic := model.Clinic{
		UserID: cu.ID,
	}

	res := s.DB.Where("id = ? AND user_id = ?", clinicID, cu.ID).Delete(&clinic)
	if err := res.Error; err != nil {
		return c.Redirect("/clinics?msg=failed to delete that clinic", fiber.StatusSeeOther)
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect("/clinics", fiber.StatusSeeOther)
	}

	c.Set("HX-Location", "/clinics")
	return c.SendStatus(fiber.StatusOK)
}

// HandleClinicIndexSearch searches patients and returns an HTML fragment to be
// replacesd in the HTMX active search
//
// POST: /clinics/search
func (s *service) HandleClinicsIndexSearch(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	term := c.Query("term", "")
	clinics := []model.Clinic{}

	s.DB.
		Model(&cu).
		Scopes(model.GlobalFTS(term)).
		Limit(20).
		Association("Clinics").
		Find(&clinics)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.ClinicsList(vc, clinics))
}

func (s *service) HandleMockManyClinics(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
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
			ProfilePic: common.DicebearShapeAvatarURL(ExtID),
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
