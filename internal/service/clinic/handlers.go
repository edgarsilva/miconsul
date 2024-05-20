package clinic

import (
	"strconv"

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

	id := c.Params("id", "")
	clinic := model.Clinic{}
	clinic.ID = id
	if id != "" && id != "new" {
		s.DB.Model(&model.Clinic{}).First(&clinic)
	}
	clinics := []model.Clinic{}
	s.DB.Model(&model.Clinic{}).Find(&clinics)

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.ClinicsPage(clinics, clinic, layoutProps))
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
	}
	c.BodyParser(&clinic)

	result := s.DB.Create(&clinic)
	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX != "" {
		c.Set("HX-Push-Url", "/clinics/"+clinic.ID)
		theme := s.SessionUITheme(c)
		layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
		return view.Render(c, view.ClinicsPage([]model.Clinic{}, clinic, layoutProps))
	}

	if err := result.Error; err != nil {
		return c.Redirect("/clinics?err=failed to create Clinic", fiber.StatusSeeOther)
	}
	return c.Redirect("/clinics/"+clinic.ID, fiber.StatusSeeOther)
}

func (s *service) HandleCreateMockClinic(c *fiber.Ctx) error {
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

	res := s.DB.Create(&users)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
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
	}
	c.BodyParser(&clinic)

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
