package patient

import (
	"fmt"

	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
)

// HandlePatientsPage renders the patients page HTML
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
	s.DB.Model(&cu).Association("Patients").Find(&patients)

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientsPage(patients, patient, layoutProps))
}

// HandleCreatePatient inserts a new patient record for the CurrentUser
//
// POST: /patients
func (s *service) HandleCreatePatient(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	c.BodyParser(&patient)

	result := s.DB.Create(&patient)
	if err := result.Error; err == nil {
		path, err := SaveProfilePicToDisk(c, patient)
		if err == nil {
			patient.ProfilePic = path
		} else {
			fmt.Println("Error ---->", err)
		}
	}

	if !s.IsHTMX(c) {
		if err := result.Error; err != nil {
			return c.Redirect("/patients?err=failed to create patient")
		}

		return c.Redirect("/patients/" + patient.ID)
	}

	c.Set("HX-Push-Url", "/patients/"+patient.ID)
	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientsPage([]model.Patient{}, patient, layoutProps))
}

// HandleUpdatePatient updates a patient record for the CurrentUser
//
// PATCH: /patients/:id
// POST: /patients/:id/patch
func (s *service) HandleUpdatePatient(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	patientID := c.Params("ID", "")
	if patientID == "" {
		patientID = c.FormValue("id", "")
	}

	if patientID == "" {
		return c.Redirect("/patients?msg=can't update without an id", fiber.StatusSeeOther)
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	c.BodyParser(&patient)
	path, err := SaveProfilePicToDisk(c, patient)
	if err == nil {
		patient.ProfilePic = path
	} else {
		fmt.Println("Error ---->", err)
	}

	res := s.DB.Where("id = ? AND user_id = ?", patientID, cu.ID).Updates(&patient)
	if err := res.Error; err != nil {
		return c.Redirect("/patients?err=failed to update patient")
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect("/patients/" + patient.ID)
	}

	return c.Redirect("/patients/"+patientID, fiber.StatusSeeOther)
}

// HandleRemovePic removes the ProfilePic from the patient
//
// PATCH: /patients/:id/removepic
// POST: /patients/:id/removepic
func (s *service) HandleRemovePic(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	patientID := c.Params("ID", "")
	if patientID == "" {
		patientID = c.FormValue("id", "")
	}

	if patientID == "" {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	res := s.DB.Model(&patient).Where("id = ? AND user_id = ?", patientID, cu.ID).Update("profile_pic", "")
	s.DB.Where("id = ?", patientID).Take(&patient)

	if !s.IsHTMX(c) {
		if err := res.Error; err != nil {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		return c.Redirect("/patients/"+patient.ID, fiber.StatusSeeOther)
	}

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientsPage([]model.Patient{}, patient, layoutProps))
}

// HandleDeletePatient deletes a patient record from the DB
//
// DELETE: /patients/:id
// POST: /patients/:id/delete
func (s *service) HandleDeletePatient(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	patientID := c.Params("ID", "")
	if patientID == "" {
		return c.Redirect("/patients?msg=can't delete without an id", fiber.StatusSeeOther)
	}

	patient := model.Patient{
		UserID: cu.ID,
	}

	res := s.DB.Where("id = ? AND user_id = ?", patientID, cu.ID).Delete(&patient)
	if err := res.Error; err != nil {
		return c.Redirect("/patients?msg=failed to delete that patient", fiber.StatusSeeOther)
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect("/patients", fiber.StatusSeeOther)
	}

	c.Set("HX-Location", "/patients")
	return c.SendStatus(fiber.StatusOK)
}

// HandlePatientSearch searches patients and returns an HTML fragment to be
// replacesd in the HTMX active search
//
// POST: /patients/search
func (s *service) HandlePatientSearch(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	queryStr := c.FormValue("query", "")
	patients := []model.Patient{}

	query := s.DB.Model(&cu)
	if queryStr != "" {
		query = query.Where("first_name LIKE ?", "%"+queryStr+"%")
	}
	query.Limit(5).Association("Patients").Find(&patients)

	// time.Sleep(time.Second * 2)
	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientSearchResults(patients, layoutProps))
}
