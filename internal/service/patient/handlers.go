package patient

import (
	"errors"
	"miconsul/internal/lib/avatar"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"os"
	"strconv"
	"strings"

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
	searchTerm, searchErr := normalizeSearchTerm(c.Query("searchTerm", ""))
	if searchErr != nil {
		return s.respondWithRedirect(c, "/patients?toast=Search term must be at least 3 characters&level=warning", fiber.StatusBadRequest)
	}

	cu := s.CurrentUser(c)
	patients, err := s.SearchPatientsByUser(c.Context(), cu, searchTerm, QUERY_LIMIT)
	if err != nil {
		redirectPath := "/patients?toast=Failed to search patients&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusInternalServerError)
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

	id := strings.TrimSpace(c.Params("id", ""))
	if id == "" {
		return s.Redirect(c, "/patients?toast=That user does not exist")
	}

	patient, err := s.PatientForShowPage(c.Context(), cu.ID, id)
	if errors.Is(err, ErrIDRequired) {
		return s.Redirect(c, "/patients?toast=That user does not exist&level=warning")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/patients?toast=Patient does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/patients?toast=Failed to load patient&level=error")
	}

	return view.Render(c, view.PatientFormPage(patient, vc))
}

// HandleCreatePatient inserts a new patient record for the CurrentUser
// POST: /patients
func (s *service) HandleCreatePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	input := patientUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		redirectPath := "/patients/new?toast=Invalid patient input&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}

	patient := input.toPatient("", cu.ID)

	patient.Sanitize()

	err = s.CreatePatient(c.Context(), &patient)
	if err != nil {
		redirectPath := "/patients/new?toast=Failed to create new patient&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	path, picErr := SaveProfilePicToDisk(c, patient, s.Env.AssetsDir)
	if picErr != nil {
		if !errors.Is(picErr, ErrProfilePicNotProvided) {
			log.Error(picErr)
		}
	} else {
		patient.ProfilePic = path
		profilePicUpdate := model.Patient{ProfilePic: path}
		err = s.UpdatePatientByID(c.Context(), cu.ID, patient.ID, profilePicUpdate)
		if err != nil {
			log.Error(err)
		}
	}

	if s.NotHTMX(c) {
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

func (s *service) respondWithRedirect(c fiber.Ctx, redirectPath string, htmxStatus int) error {
	if s.NotHTMX(c) {
		return s.Redirect(c, redirectPath)
	}

	c.Set("HX-Location", redirectPath)
	return c.SendStatus(htmxStatus)
}

// HandleUpdatePatient updates a patient record for the CurrentUser
// PATCH: /patients/:id
// POST: /patients/:id/patch
func (s *service) HandleUpdatePatient(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		patientID = strings.TrimSpace(c.FormValue("id", ""))
	}

	if patientID == "" {
		return s.respondWithRedirect(c, "/patients?toast=Can't update without an id&level=error", fiber.StatusBadRequest)
	}

	input := patientUpsertInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		redirectPath := "/patients/" + patientID + "?toast=Invalid patient input&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}

	patient := input.toPatient(patientID, cu.ID)

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
		redirectPath := "/patients?toast=Can't update without an id&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/patients?toast=Patient does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/patients?toast=Failed to update patient&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	if s.NotHTMX(c) {
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

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		patientID = strings.TrimSpace(c.FormValue("id", ""))
	}

	if patientID == "" {
		redirectPath := "/patients?toast=Can't remove profile picture without an id&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	err := s.ClearPatientProfilePic(c.Context(), cu.ID, patientID)
	if errors.Is(err, ErrIDRequired) {
		redirectPath := "/patients?toast=Can't remove profile picture without an id&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/patients?toast=Patient does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/patients?toast=Failed to remove profile picture&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	patient, err = s.TakePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/patients?toast=Patient does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/patients?toast=Failed to load patient&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusInternalServerError)
	}

	if s.NotHTMX(c) {
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

	patientID := strings.TrimSpace(c.Params("id", ""))
	if patientID == "" {
		return s.respondWithRedirect(c, "/patients?toast=Can't delete without an id&level=error", fiber.StatusBadRequest)
	}

	err := s.DeletePatientByID(c.Context(), cu.ID, patientID)
	if errors.Is(err, ErrIDRequired) {
		return s.respondWithRedirect(c, "/patients?toast=Can't delete without an id&level=error", fiber.StatusBadRequest)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.respondWithRedirect(c, "/patients?toast=Patient does not exist&level=warning", fiber.StatusNotFound)
	}
	if err != nil {
		return s.respondWithRedirect(c, "/patients?toast=Failed to delete patient&level=error", fiber.StatusUnprocessableEntity)
	}

	if s.NotHTMX(c) {
		return s.Redirect(c, "/patients")
	}

	searchTerm := strings.TrimSpace(c.Query("searchTerm", ""))
	patients, err := s.SearchPatientsByUser(c.Context(), cu, searchTerm, QUERY_LIMIT)
	if err != nil {
		return s.respondWithRedirect(c, "/patients?toast=Failed to load patients&level=error", fiber.StatusInternalServerError)
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

	searchTerm, searchErr := normalizeSearchTerm(c.FormValue("searchTerm", ""))
	if searchErr != nil {
		return s.respondWithRedirect(c, "/patients?toast=Search term must be at least 3 characters&level=warning", fiber.StatusBadRequest)
	}

	patients, err := s.SearchPatientsByUser(c.Context(), cu, searchTerm, QUERY_LIMIT)
	if err != nil {
		redirectPath := "/patients?toast=Failed to search patients&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusInternalServerError)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.PatientSearchResults(patients, vc))
}

func normalizeSearchTerm(raw string) (string, error) {
	term := strings.TrimSpace(raw)
	if len(term) > 0 && len(term) < 3 {
		return "", errors.New("search term must be at least 3 chars")
	}

	return term, nil
}

// HandleMockManyPatients creates many mock patients for admin/testing flows.
// GET: /patients/makeaton
func (s *service) HandleMockManyPatients(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	const (
		defaultMockPatients = 1000
		maxMockPatients     = 10000
	)

	n, err := strconv.Atoi(c.Query("n", strconv.Itoa(defaultMockPatients)))
	if err != nil || n <= 0 {
		n = defaultMockPatients
	}
	if n > maxMockPatients {
		n = maxMockPatients
	}

	patients := []model.Patient{}
	for i := 0; i < n; i++ {
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
