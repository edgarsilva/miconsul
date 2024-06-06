package patient

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/util"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
	"syreclabs.com/go/faker"
)

// HandlePatientsPage renders the patients page HTML
//
// GET: /patients
func (s *service) HandlePatientsPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	id := c.Params("id", "")
	patient := model.Patient{}
	patient.ID = id
	if id != "" && id != "new" {
		s.DB.Model(&model.Patient{}).First(&patient)
		return view.Render(c, view.PatientFormPage(patient, layoutProps))
	}

	patients := []model.Patient{}
	s.DB.Model(&cu).Limit(25).Association("Patients").Find(&patients)
	return view.Render(c, view.PatientsPage(patients, layoutProps))
}

// HandlePatientFormPage renders the patients page HTML
//
// GET: /patients/:id
func (s *service) HandlePatientFormPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	id := c.Params("id", "")
	if id == "" {
		return c.Redirect("/patients?toast=That user does not exist")
	}

	patient := model.Patient{}
	if id != "new" {
		patient.ID = id
		s.DB.Model(&model.Patient{}).First(&patient)
	}

	return view.Render(c, view.PatientFormPage(patient, layoutProps))
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
		}
	}

	if err := result.Error; err != nil {
		redirectPath := "/patients/new?toast=Failed to create new patient&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath)
		}
	}

	if !s.IsHTMX(c) {
		return c.Redirect("/patients/" + patient.ID)
	}

	c.Set("HX-Push-Url", "/patients/"+patient.ID)
	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(
		c,
		view.WithTheme(theme),
		view.WithCurrentUser(cu),
		view.WithToast("New patient created", "", ""),
	)
	return view.Render(c, view.PatientFormPage(patient, layoutProps))
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
	}

	res := s.DB.Where("id = ? AND user_id = ?", patientID, cu.ID).Updates(&patient)
	if err := res.Error; err != nil {
		redirectPath := "/patients?err=failed to update patient&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	if !s.IsHTMX(c) {
		return c.Redirect("/patients/"+patientID, fiber.StatusSeeOther)
	}

	c.Set("HX-Location", "/patients/"+patientID+"?toast=Patient changes saved&level=success")
	return c.SendStatus(fiber.StatusOK)
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
	layoutProps, _ := view.NewLayoutProps(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientFormPage(patient, layoutProps))
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

	dbquery := s.DB.Model(&cu)
	if queryStr != "" {
		dbquery.Scopes(model.GlobalFTS(queryStr))
	} else {
		dbquery.Order("created_at desc")
	}

	dbquery.Limit(10).Association("Patients").Find(&patients)

	// time.Sleep(time.Second * 2)
	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientSearchResults(patients, layoutProps))
}

func (s *service) HandleMockManyPatients(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	n, err := strconv.Atoi(c.Query("n", "100000"))
	if err != nil {
		n = 100000
	}

	var patients []model.Patient
	for i := 0; i <= n; i++ {
		ExtID := xid.New("prav")
		patients = append(patients, model.Patient{
			ExtID:      ExtID,
			ProfilePic: util.PravatarURL(ExtID),
			FirstName:  faker.Name().FirstName(),
			LastName:   faker.Name().LastName(),
			Email:      faker.Internet().Email(),
			Phone:      faker.PhoneNumber().CellPhone(),
			Age:        25,
			UserID:     cu.ID,
		})
	}

	result := s.DB.CreateInBatches(&patients, 1000)
	if err := result.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendString("Rowsaffected:" + strconv.Itoa(int(result.RowsAffected)))
}
