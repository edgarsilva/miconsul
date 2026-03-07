package patient

import (
	"errors"
	"miconsul/internal/lib/avatar"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
	"syreclabs.com/go/faker"
)

const (
	QUERY_LIMIT int = 10
)

// HandleIndexPage renders the patients index page HTML.
// GET: /patients
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	patients, err := s.FindRecentPatientsByUser(c.Context(), cu, QUERY_LIMIT)
	if err != nil {
		return s.Redirect(c, "/patients?toast=Failed to load patients&level=error")
	}

	return view.Render(c, view.PatientsPage(vc, patients))
}

// HandlePatientsIndexSearch runs the index search on patients.
// GET: /patients/search
func (s *service) HandlePatientsIndexSearch(c fiber.Ctx) error {
	term := c.Query("term", "")
	if len(term) < 3 {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	cu := s.CurrentUser(c)
	patients, err := s.Patients(cu, term)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.PatientsList(vc, patients))
}

// HandlePatientFormPage renders the patient create/edit page.
// GET: /patients/:id
func (s *service) HandlePatientFormPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	id := c.Params("id", "")
	if id == "" {
		return s.Redirect(c, "/patients?toast=That user does not exist")
	}

	patient := model.Patient{}
	if id != "new" {
		var err error
		patient, err = s.TakePatientByID(c.Context(), id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.Redirect(c, "/patients?toast=Patient does not exist&level=warning")
		}
		if err != nil {
			return s.Redirect(c, "/patients?toast=Failed to load patient&level=error")
		}
	}

	return view.Render(c, view.PatientFormPage(patient, vc))
}

// HandleCreatePatient inserts a new patient record for the CurrentUser
// POST: /patients
func (s *service) HandleCreatePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	var err error

	patient := model.Patient{
		UserID: cu.ID,
	}
	err = c.Bind().Body(&patient)
	if err != nil {
		redirectPath := "/patients/new?toast=Invalid patient input&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	patient.Sanitize()

	err = s.CreatePatient(c.Context(), &patient)
	if err == nil {
		path, picErr := SaveProfilePicToDisk(c, patient)
		if picErr != nil {
			log.Error(picErr)
		} else {
			patient.ProfilePic = path
		}
	}

	if err != nil {
		redirectPath := "/patients/new?toast=Failed to create new patient&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/patients/"+patient.ID)
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
// PATCH: /patients/:id
// POST: /patients/:id/patch
func (s *service) HandleUpdatePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	var err error

	patientID := c.Params("id", "")
	if patientID == "" {
		patientID = c.FormValue("id", "")
	}

	if patientID == "" {
		return s.Redirect(c, "/patients?msg=can't update without an id")
	}

	patient := model.Patient{ID: patientID, UserID: cu.ID}
	err = c.Bind().Body(&patient)
	if err != nil {
		redirectPath := "/patients/" + patientID + "?toast=Invalid patient input&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	patient.Sanitize()

	path, picErr := SaveProfilePicToDisk(c, patient)
	if picErr != nil {
		log.Error(picErr)
	} else {
		patient.ProfilePic = path
	}

	err = s.UpdatePatientByIDAndUserID(c.Context(), cu.ID, patientID, patient)
	if err != nil {
		redirectPath := "/patients?err=failed to update patient&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/patients/"+patientID)
	}

	c.Set("HX-Location", "/patients/"+patientID+"?toast=Patient changes saved&level=success")
	return c.SendStatus(fiber.StatusOK)
}

// HandleRemovePic removes the ProfilePic from the patient
// PATCH: /patients/:id/removepic
// POST: /patients/:id/removepic
func (s *service) HandleRemovePic(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	var err error

	patientID := c.Params("id", "")
	if patientID == "" {
		patientID = c.FormValue("id", "")
	}

	if patientID == "" {
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	err = s.ClearPatientProfilePic(c.Context(), cu.ID, patientID)
	if err != nil {
		redirectPath := "/patients?toast=Failed to remove profile picture&level=error"
		if s.NotHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	patient, err = s.TakePatientByIDAndUserID(c.Context(), cu.ID, patientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/patients?toast=Patient does not exist&level=warning"
		if s.NotHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/patients?toast=Failed to load patient&level=error"
		if s.NotHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/patients/"+patient.ID)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientFormPage(patient, vc))
}

// HandleDeletePatient deletes a patient record from the DB.
// DELETE: /patients/:id
// POST: /patients/:id/delete
func (s *service) HandleDeletePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	var err error

	patientID := c.Params("id", "")
	if patientID == "" {
		return s.Redirect(c, "/patients?msg=can't delete without an id")
	}

	err = s.DeletePatientByIDAndUserID(c.Context(), cu.ID, patientID)
	if err != nil {
		return s.Redirect(c, "/patients?msg=failed to delete that patient")
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/patients")
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
// POST: /patients/search
func (s *service) HandlePatientSearch(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	queryStr := c.FormValue("query", "")
	patients, err := s.SearchPatientsByUser(c.Context(), cu, queryStr, QUERY_LIMIT)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientSearchResults(patients, vc))
}

// HandleMockManyPatients creates many mock patients for admin/testing flows.
// GET: /patients/makeaton
func (s *service) HandleMockManyPatients(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	n, err := strconv.Atoi(c.Query("n", "100000"))
	if err != nil {
		n = 100000
	}

	var patients []model.Patient
	for i := 0; i <= n; i++ {
		ExtID := xid.New("prav")
		patients = append(patients, model.Patient{
			ExtID:      ExtID,
			ProfilePic: avatar.PravatarURL(ExtID),
			Name:       faker.Name().Name(),
			Email:      faker.Internet().Email(),
			Phone:      faker.PhoneNumber().CellPhone(),
			Age:        25,
			UserID:     cu.ID,
		})
	}

	rowsAffected, err := s.CreatePatientsInBatches(c.Context(), patients, 1000)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendString("Rowsaffected:" + strconv.Itoa(int(rowsAffected)))
}

// HandlePatientProfilePicImgSrc serves a patient's profile picture file.
// GET: /patients/:id/profilepic/:filename
func (s *service) HandlePatientProfilePicImgSrc(c fiber.Ctx) error {
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
