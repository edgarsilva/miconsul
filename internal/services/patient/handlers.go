package patient

import (
	"errors"
	"os"
	"strings"

	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
)

const (
	QUERY_LIMIT int = 10
)

// HandleIndexPage renders the patients index page HTML.
// GET: /patients
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	vc := view.NewCtx(c)

	patients, err := s.FindRecentPatientsByUser(c.Context(), cu, QUERY_LIMIT)
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patients", "error")
	}

	return view.Render(c, view.PatientIndexPage(vc, patients))
}

// HandlePatientsIndexSearch runs the index search on patients.
// GET: /patients/search
func (s *service) HandlePatientsIndexSearch(c fiber.Ctx) error {
	searchTerm, searchErr := normalizeSearchTerm(c.Query("searchTerm", ""))
	if searchErr != nil {
		return view.Render(c, view.CmpSearchTooShortToast(3))
	}

	cu := s.CurrentUser(c)
	patients, err := s.SearchPatientsByUser(c.Context(), cu, searchTerm, QUERY_LIMIT)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.PatientList(vc, patients))
}

// HandlePatientFormPage renders the patient create/edit page.
// GET: /patients/:id
func (s *service) HandlePatientFormPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	vc := view.NewCtx(c)

	id := strings.TrimSpace(c.Params("id", ""))
	if id == "" {
		return s.RedirectWithToast(c, "/patients", "That user does not exist", "")
	}

	patient, err := s.PatientForShowPage(c.Context(), cu.ID, id)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "That user does not exist", "warning")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", "warning")
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patient", "error")
	}

	return view.Render(c, view.PatientFormPage(vc, patient))
}

// HandleCreatePatient inserts a new patient record for the CurrentUser
// POST: /patients
func (s *service) HandleCreatePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	input := patientUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/patients/new", "Invalid patient input", "error")
	}

	patient := input.toPatient(0, "", cu.ID)

	patient.Sanitize()

	err = s.CreatePatient(c.Context(), &patient)
	if err != nil {
		return s.RedirectWithToast(c, "/patients/new", "Failed to create new patient", "error")
	}

	path, picErr := SaveProfilePicToDisk(c, patient, s.Env.AssetsDir)
	if picErr != nil {
		if !errors.Is(picErr, ErrProfilePicNotProvided) {
			log.Error(picErr)
		}
	} else {
		patient.ProfilePic = path
		profilePicUpdate := models.Patient{ProfilePic: path}
		err = s.UpdatePatientByID(c.Context(), cu.ID, patient.UID, profilePicUpdate)
		if err != nil {
			log.Error(err)
		}
	}

	c.Set("HX-Push-Url", "/patients/"+patient.UID)
	vc := view.NewCtx(c)
	return view.Render(c, templ.Join(
		view.PatientProfileSection(vc, patient),
		view.CmpToastPartial("New patient created", "", ""),
	))
}

// HandleUpdatePatient updates a patient record for the CurrentUser
// PATCH: /patients/:id
func (s *service) HandleUpdatePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		patientID = strings.TrimSpace(c.FormValue("id", ""))
	}

	if patientID == "" {
		return s.RedirectWithToast(c, "/patients", "Can't update without an id", "error")
	}

	input := patientUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/patients/"+patientID, "Invalid patient input", "error")
	}

	existingPatient, findErr := s.TakePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", "warning")
	}
	if findErr != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patient", "error")
	}

	patient := input.toPatient(existingPatient.ID, existingPatient.UID, cu.ID)

	patient.Sanitize()

	path, picErr := SaveProfilePicToDisk(c, patient, s.Env.AssetsDir)
	if picErr != nil {
		if !errors.Is(picErr, ErrProfilePicNotProvided) {
			log.Error(picErr)
		}
	} else {
		patient.ProfilePic = path
	}

	err = s.UpdatePatientByID(c.Context(), cu.ID, patientID, patient)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "Can't update without an id", "error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", "warning")
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to update patient", "error")
	}

	return s.RedirectWithToast(c, "/patients/"+patientID, "Patient changes saved", "success")
}

// HandleRemovePic removes the ProfilePic from the patient
// PATCH: /patients/:id/removepic
func (s *service) HandleRemovePic(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		patientID = strings.TrimSpace(c.FormValue("id", ""))
	}

	if patientID == "" {
		return s.RedirectWithToast(c, "/patients", "Can't remove profile picture without an id", "error")
	}

	patient := models.Patient{
		UserID: cu.ID,
	}
	err := s.ClearPatientProfilePic(c.Context(), cu.ID, patientID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "Can't remove profile picture without an id", "error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", "warning")
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to remove profile picture", "error")
	}

	patient, err = s.TakePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", "warning")
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patient", "error")
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.PatientAvatarPic(vc, patient))
}

// HandleDeletePatient deletes a patient record from the DB.
// DELETE: /patients/:id
func (s *service) HandleDeletePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		return s.RedirectWithToast(c, "/patients", "Can't delete without an id", "error")
	}

	err := s.DeletePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "Can't delete without an id", "error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", "warning")
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to delete patient", "error")
	}

	searchTerm := strings.TrimSpace(c.Query("searchTerm", ""))
	patients, err := s.SearchPatientsByUser(c.Context(), cu, searchTerm, QUERY_LIMIT)
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patients", "error")
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.PatientList(vc, patients))
}

func normalizeSearchTerm(raw string) (string, error) {
	term := strings.TrimSpace(raw)
	if len(term) > 0 && len(term) < 3 {
		return "", errors.New("search term must be at least 3 chars")
	}

	return term, nil
}

// HandlePatientProfilePicImgSrc serves a patient's profile picture file.
// GET: /patients/:id/profilepic/:filename
func (s *service) HandlePatientProfilePicImgSrc(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	id := strings.TrimSpace(c.Params("id", ""))
	filename := strings.TrimSpace(c.Params("filename", ""))
	if id == "" || filename == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if !IsSafeProfilePicFilenameForPatient(id, filename) {
		return c.SendStatus(fiber.StatusNotFound)
	}

	_, err := s.TakePatientByID(c.Context(), cu.ID, id)
	if errors.Is(err, ErrIDRequired) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	path, err := ProfilePicPath(filename, s.Env.AssetsDir)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	fileInfo, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if fileInfo.IsDir() {
		return c.SendStatus(fiber.StatusNotFound)
	}

	return c.SendFile(path)
}
