package patient

import (
	"errors"
	"fmt"
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
		return s.RedirectWithToast(c, "/patients", "Failed to load patients", view.LevelError)
	}

	return view.Render(c, view.PatientIndexPage(vc, patients))
}

// HandleIndexSearch runs the index search on patients.
// GET: /patients/search
func (s *service) HandleIndexSearch(c fiber.Ctx) error {
	searchTerm, searchErr := normalizeSearchTerm(c.Query("searchTerm", ""))
	if searchErr != nil {
		return view.Render(c, view.CmpSearchTooShortToast())
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

// HandleNewPage renders the new patient page.
// GET: /patients/new
func (s *service) HandleNewPage(c fiber.Ctx) error {
	vc := view.NewCtx(c)
	return view.Render(c, view.PatientFormPage(vc, models.Patient{}))
}

// HandleEditPage renders the patient edit page.
// GET: /patients/:id
func (s *service) HandleEditPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	vc := view.NewCtx(c)

	patient, err := s.TakePatientByID(c.Context(), cu.ID, c.Params("id", ""))
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "That patient does not exist", view.LevelWarning)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patient", view.LevelError)
	}

	return view.Render(c, view.PatientFormPage(vc, patient))
}

// HandleCreate inserts a new patient record for the CurrentUser
// POST: /patients
func (s *service) HandleCreate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	input := patientUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/patients/new", "Invalid patient input", view.LevelError)
	}

	patient := input.toPatient(0, "", cu.ID)

	patient.Sanitize()

	err = s.CreatePatient(c.Context(), &patient)
	if err != nil {
		return s.RedirectWithToast(c, "/patients/new", "Failed to create new patient", view.LevelError)
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
		view.CmpToastPartial("New patient created", "", view.LevelSuccess),
	))
}

// HandleUpdate updates a patient record for the CurrentUser
// PATCH: /patients/:id
func (s *service) HandleUpdate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		patientID = strings.TrimSpace(c.FormValue("id", ""))
	}

	if patientID == "" {
		return s.RedirectWithToast(c, "/patients", "Can't update without an id", view.LevelError)
	}

	input := patientUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/patients/"+patientID, "Invalid patient input", view.LevelError)
	}

	existingPatient, findErr := s.TakePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", view.LevelWarning)
	}
	if findErr != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patient", view.LevelError)
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
		return s.RedirectWithToast(c, "/patients", "Can't update without an id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to update patient", view.LevelError)
	}

	return s.RedirectWithToast(c, "/patients/"+patientID, "Patient changes saved", view.LevelSuccess)
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
		return s.RedirectWithToast(c, "/patients", "Can't remove profile picture without an id", view.LevelError)
	}

	patient := models.Patient{
		UserID: cu.ID,
	}
	err := s.ClearPatientProfilePic(c.Context(), cu.ID, patientID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "Can't remove profile picture without an id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to remove profile picture", view.LevelError)
	}

	patient, err = s.TakePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patient", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.PatientAvatar(vc, patient))
}

// HandleDelete deletes a patient record from the DB.
// DELETE: /patients/:id
func (s *service) HandleDelete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		return s.RedirectWithToast(c, "/patients", "Can't delete without an id", view.LevelError)
	}

	err := s.DeletePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/patients", "Can't delete without an id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/patients", "Patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to delete patient", view.LevelError)
	}

	searchTerm := strings.TrimSpace(c.Query("searchTerm", ""))
	patients, err := s.SearchPatientsByUser(c.Context(), cu, searchTerm, QUERY_LIMIT)
	if err != nil {
		return s.RedirectWithToast(c, "/patients", "Failed to load patients", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.PatientList(vc, patients))
}

func normalizeSearchTerm(raw string) (string, error) {
	term := strings.TrimSpace(raw)
	if models.SearchTermTooShort(term) {
		return "", fmt.Errorf("search term must be at least %d chars", models.MinSearchTermLen)
	}

	return term, nil
}

// HandleProfilePic serves a patient's profile picture file.
// GET: /patients/:id/profilepic/:filename
func (s *service) HandleProfilePic(c fiber.Ctx) error {
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
