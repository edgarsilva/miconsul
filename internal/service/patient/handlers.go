package patient

import (
	"miconsul/internal/lib/handlerutils"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"syreclabs.com/go/faker"
)

const (
	QUERY_LIMIT int = 10
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
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	id := c.Params("id", "")
	patient := model.Patient{ID: id}
	if id != "" && id != "new" {
		s.DB.Model(&model.Patient{}).Take(&patient)
		return view.Render(c, view.PatientFormPage(patient, vc))
	}

	patients := []model.Patient{}
	s.DB.Model(&cu).Order("created_at desc").Limit(QUERY_LIMIT).Association("Patients").Find(&patients)
	return view.Render(c, view.PatientsPage(vc, patients))
}

// HandlePatientsIndexSearch GlobalFTS search for patients index page.
//
// GET: /patients/search
func (s *service) HandlePatientsIndexSearch(c *fiber.Ctx) error {
	term := c.Query("term", "")
	if len(term) < 3 {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	cu, _ := s.CurrentUser(c)
	patients, err := s.Patients(cu, term)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.PatientsList(vc, patients))
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
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	id := c.Params("id", "")
	if id == "" {
		return c.Redirect("/patients?toast=That user does not exist")
	}

	patient := model.Patient{}
	if id != "new" {
		patient.ID = id
		s.DB.Model(&model.Patient{}).First(&patient)
	}

	return view.Render(c, view.PatientFormPage(patient, vc))
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
	patient.Sanitize()

	result := s.DB.Create(&patient)
	if err := result.Error; err == nil {
		path, err := SaveProfilePicToDisk(c, patient)
		if err == nil {
			log.Error(err)
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
	vc, _ := view.NewCtx(
		c,
		view.WithTheme(theme),
		view.WithCurrentUser(cu),
		view.WithToast("New patient created", "", ""),
	)
	return view.Render(c, view.PatientFormPage(patient, vc))
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

	patientID := c.Params("id", "")
	if patientID == "" {
		patientID = c.FormValue("id", "")
	}

	if patientID == "" {
		return c.Redirect("/patients?msg=can't update without an id", fiber.StatusSeeOther)
	}

	patient := model.Patient{ID: patientID, UserID: cu.ID}
	c.BodyParser(&patient)
	patient.Sanitize()

	path, err := SaveProfilePicToDisk(c, patient)
	if err == nil {
		log.Error(err)
		patient.ProfilePic = path
	}

	result := s.DB.Where("id = ? AND user_id = ?", patientID, cu.ID).Updates(&patient)
	if err := result.Error; err != nil {
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
	s.DB.WithContext(c.UserContext()).Model(&patient).Where("id = ?", patientID).Take(&patient)

	if !s.IsHTMX(c) {
		if err := res.Error; err != nil {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		return c.Redirect("/patients/"+patient.ID, fiber.StatusSeeOther)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientFormPage(patient, vc))
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

	if s.NotHTMX(c) {
		return c.Redirect("/patients", fiber.StatusSeeOther)
	}

	patients, err := s.Patients(cu, c.Query("term", ""))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientsList(vc, patients))
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

	dbquery := s.DB.WithContext(c.UserContext()).Model(&cu)
	if queryStr != "" {
		dbquery.Scopes(model.GlobalFTS(queryStr))
	} else {
		dbquery.Order("created_at desc")
	}

	dbquery.Limit(QUERY_LIMIT).Association("Patients").Find(&patients)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientSearchResults(patients, vc))
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
			ProfilePic: handlerutils.PravatarURL(ExtID),
			Name:       faker.Name().Name(),
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

func (s *service) HandlePatientProfilePicImgSrc(c *fiber.Ctx) error {
	id := c.Params("id", "")
	filename := c.Params("filename", "")
	if id == "" || filename == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}

	path, err := ProfilePicPath(filename)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.SendFile(path)
}
