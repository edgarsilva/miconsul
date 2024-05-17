package patient

import (
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
)

// handlePatientsPage renders the patients page HTML
//
// GET: /patients
func (s *service) HandlePatientsPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	id := c.Params("id", "")
	patient := model.Patient{}
	patient.ID = id
	if id != "" && id != "new" {
		s.DB.Model(&model.Patient{}).First(&patient)
	}
	patients := []model.Patient{}
	s.DB.Model(&model.Patient{}).Find(&patients)

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientsPage(patients, patient, layoutProps))
}

// HandleCreatePatient inserts a new clinic record for the given user
//
// POST: /clinics
func (s *service) HandleCreatePatient(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	c.BodyParser(&patient)

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))

	res := s.DB.Create(&patient)
	if err := res.Error; err != nil {
		return view.Render(c, view.PatientsPage([]model.Patient{}, patient, layoutProps))
	}

	patients := []model.Patient{}
	s.DB.Model(&model.Patient{}).Find(&patients)

	return view.Render(c, view.PatientsPage(patients, patient, layoutProps))
}

// HandleDeletePatient deletes a patient record from the DB
//
// DELETE: /patients/:id
func (s *service) HandleDeletePatient(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	patientID := c.Params("ID", "")
	if patientID == "" {
		return c.Redirect("/patients?msg=can't delete without an id")
	}

	patient := model.Patient{
		UserID: cu.ID,
	}

	res := s.DB.Where("id = ? AND user_id = ?", patientID, cu.ID).Delete(&patient)
	if err := res.Error; err != nil {
		return c.Redirect("/patients?msg=failed to delete that patient")
	}

	return c.Redirect("/patients")
}
