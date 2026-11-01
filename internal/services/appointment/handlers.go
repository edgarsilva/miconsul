package appointment

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"strings"
	"time"

	"miconsul/internal/lib/amount"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
)

// HandleIndexPage renders the appointments page HTML
// GET: /appointments
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	filter := appointmentFilterFromQuery(c)

	patient, err := s.takePatientOrZero(c.Context(), cu.ID, filter.PatientUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "Selected patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load selected patient", view.LevelError)
	}

	clinic, err := s.takeClinicOrZero(c.Context(), cu.ID, filter.ClinicUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "Selected clinic does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load selected clinic", view.LevelError)
	}
	appointments, err := s.FindAppointmentsBy(c.Context(), cu.ID, filter)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load appointments", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AppointmentIndexPage(vc, appointments, patient, clinic))
}

// HandleIndexSearch runs the appointments index search.
// GET: /appointments/search
func (s *service) HandleIndexSearch(c fiber.Ctx) error {
	filter := appointmentFilterFromQuery(c)
	if models.SearchTermTooShort(filter.SearchTerm) {
		return view.Render(c, view.CmpSearchTooShortToast())
	}

	cu := s.CurrentUser(c)
	patient, err := s.takePatientOrZero(c.Context(), cu.ID, filter.PatientUID)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	clinic, err := s.takeClinicOrZero(c.Context(), cu.ID, filter.ClinicUID)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	appointments, err := s.FindAppointmentsBy(c.Context(), cu.ID, filter)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AppointmentIndexSection(vc, appointments, patient, clinic))
}

// HandleShowPage renders the appointment create/edit page.
// GET: /appointments/new
// GET: /appointments/:id
func (s *service) HandleShowPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	appointment, err := s.AppointmentForShowPage(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/appointments", "Invalid appointment id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load appointment", view.LevelError)
	}

	clinics, err := s.FindRecentClinicsByUserID(c.Context(), cu.ID, 10)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load clinics", view.LevelError)
	}

	patients, err := s.FindRecentPatientsByUserID(c.Context(), cu.ID, 10)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load patients", view.LevelError)
	}

	clinics = ensureClinicListed(clinics, appointment.Clinic)
	patients = ensurePatientListed(patients, appointment.Patient)
	vc := view.NewCtx(c)

	return view.Render(c, view.AppointmentPage(vc, appointment, patients, clinics))
}

// HandleOpenPage renders the appointment start page HTML
// GET: /appointments/:id/open
func (s *service) HandleOpenPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}

	appointment, err := s.TakeAppointmentByUID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, ErrIDRequired) {
		return s.RedirectWithToast(c, "/appointments", "Invalid appointment id", view.LevelError)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || appointment.ID == 0 {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load appointment", view.LevelError)
	}

	patient, err := s.PatientForStartPage(c.Context(), cu.ID, appointment.PatientID)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment patient does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to load appointment patient", view.LevelError)
	}

	appointment.Patient = patient

	vc := view.NewCtx(c)
	return view.Render(c, view.AppointmentStartPage(vc, appointment))
}

// HandleStartSession handles starting an appointment session.
// POST: /appointments/:id/start-session
func (s *service) HandleStartSession(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "Can't find that appointment", view.LevelError)
	}

	err := s.StartAppointmentSessionByID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to start appointment session", view.LevelError)
	}

	vc := view.NewCtx(c)
	appointment := models.Appointment{UID: appointmentID, Status: models.AppointmentInProgress}
	return view.Render(c, view.AppointmentSessionToggle(vc, appointment))
}

// HandlePauseSession handles pausing an appointment session.
// POST: /appointments/:id/pause-session
func (s *service) HandlePauseSession(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "Can't find that appointment", view.LevelError)
	}

	err := s.PauseAppointmentSessionByID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to pause appointment session", view.LevelError)
	}

	vc := view.NewCtx(c)
	appointment := models.Appointment{UID: appointmentID, Status: models.AppointmentPending}
	return view.Render(c, view.AppointmentSessionToggle(vc, appointment))
}

// HandleSaveSession handles saving in-progress session notes.
// POST: /appointments/:id/save-session
func (s *service) HandleSaveSession(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "Can't find that appointment", view.LevelError)
	}

	input := appointmentCompleteInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID+"/open", "Invalid appointment input", view.LevelError)
	}

	updates := appointmentSessionUpdates{
		Observations: input.Observations,
		Conclusions:  input.Conclusions,
		Summary:      input.Summary,
		Notes:        input.Notes,
	}

	err = s.SaveAppointmentSessionByID(c.Context(), cu.ID, appointmentID, updates)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID+"/open", "Failed to save appointment session", view.LevelError)
	}

	return s.RedirectWithToast(c, "/appointments/"+appointmentID+"/open", "Session saved", view.LevelSuccess)
}

// HandleComplete handles the request to mark an appointment as completed/done.
// POST: /appointments/:id/complete
func (s *service) HandleComplete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "Can't find that appointment", view.LevelError)
	}

	input := appointmentCompleteInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID, "Invalid appointment input", view.LevelError)
	}

	appointment := appointmentCompleteUpdates{
		Status:       models.AppointmentDone,
		Observations: input.Observations,
		Conclusions:  input.Conclusions,
		Summary:      input.Summary,
		Notes:        input.Notes,
	}

	err = s.CompleteAppointmentByID(c.Context(), cu.ID, appointmentID, appointment)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to update appointment", view.LevelError)
	}

	return s.HXRedirect(c, "/appointments")
}

// HandleCreate inserts a new appointment record for the CurrentUser
// POST: /appointments
func (s *service) HandleCreate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	bookedAtValue := c.FormValue("bookedAt", "")
	bookedAt, err := time.Parse(view.FormTimeFormat, bookedAtValue)
	if err != nil {
		bookedAt = time.Now()
	}

	priceValue := c.FormValue("price", "")

	bookedAt = libtime.NewInTimezone(bookedAt, models.DefaultTimezone)
	bookedAt = bookedAt.UTC()
	input := appointmentUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/new", "Invalid appointment input", view.LevelError)
	}

	appointment := models.Appointment{
		Token:        rand.Text(),
		UserID:       cu.ID,
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Timezone:     models.DefaultTimezone,
		Price:        amount.StrToAmount(priceValue),
		ClinicID:     0,
		PatientID:    0,
		Duration:     input.Duration,
	}

	clinic, err := s.TakeClinicByUID(c.Context(), cu.ID, input.ClinicUID)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/new", "Invalid clinic selected", view.LevelError)
	}
	patient, err := s.TakePatientByUID(c.Context(), cu.ID, input.PatientUID)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/new", "Invalid patient selected", view.LevelError)
	}
	appointment.ClinicID = clinic.ID
	appointment.PatientID = patient.ID

	err = s.CreateAppointment(c.Context(), &appointment)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "failed to create appointment", view.LevelError)
	}

	appointment, err = s.TakeAppointmentByUID(c.Context(), cu.ID, appointment.UID)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Appointment created, but failed to load related records", view.LevelWarning)
	}

	err = s.DispatchBookedAlert(appointment)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Appointment created, but failed to queue alert", view.LevelWarning)
	}

	return s.RedirectWithToast(c, "/appointments?patientId="+url.QueryEscape(patient.UID), "New appointment created", view.LevelSuccess)
}

// HandleUpdate updates an appointment record for the current user.
// PATCH: /appointments/:id
func (s *service) HandleUpdate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "can't update without an id", view.LevelError)
	}

	_, err := s.TakeAppointmentByUID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID, "Failed to load appointment", view.LevelError)
	}

	bookedAtStr := c.FormValue("bookedAt", "")
	bookedAt, err := time.Parse(view.FormTimeFormat, bookedAtStr)
	if err != nil {
		bookedAt = time.Now()
	}

	bookedAt = libtime.NewInTimezone(bookedAt, models.DefaultTimezone)
	bookedAt = bookedAt.UTC()
	input := appointmentUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID, "Invalid appointment input", view.LevelError)
	}

	appointment := appointmentPatchUpdates{
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Price:        amount.StrToAmount(c.FormValue("price", "")),
		ClinicID:     0,
		PatientID:    0,
		Duration:     input.Duration,
	}

	clinic, err := s.TakeClinicByUID(c.Context(), cu.ID, input.ClinicUID)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID, "Invalid clinic selected", view.LevelError)
	}
	patient, err := s.TakePatientByUID(c.Context(), cu.ID, input.PatientUID)
	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID, "Invalid patient selected", view.LevelError)
	}
	appointment.ClinicID = clinic.ID
	appointment.PatientID = patient.ID

	err = s.UpdateAppointmentByID(c.Context(), cu.ID, appointmentID, appointment)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}

	if err != nil {
		return s.RedirectWithToast(c, "/appointments/"+appointmentID, "failed to update appointment", view.LevelError)
	}

	return s.RedirectWithToast(c, "/appointments?patientId="+url.QueryEscape(patient.UID), "Appointment saved", view.LevelSuccess)
}

// HandleCancel cancels an appointment
// POST: /appointments/:id/cancel
func (s *service) HandleCancel(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "Can't find that appointment", view.LevelError)
	}

	appointment := appointmentCancelUpdates{
		Status:     models.AppointmentCanceled,
		CanceledAt: time.Now().UTC(),
	}
	err := s.CancelAppointmentByID(c.Context(), cu.ID, appointmentID, appointment)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "The appointment does not exist", view.LevelWarning)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "Failed to update appointment", view.LevelError)
	}

	return s.HXRedirect(c, "/appointments")
}

// HandleDelete deletes an appointment record from the DB.
// DELETE: /appointments/:id
func (s *service) HandleDelete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		return s.RedirectWithToast(c, "/appointments", "can't delete without an id", view.LevelError)
	}

	err := s.DeleteAppointmentByID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.RedirectWithToast(c, "/appointments", "can't find that appointment", view.LevelError)
	}
	if err != nil {
		return s.RedirectWithToast(c, "/appointments", "failed to delete that appointment", view.LevelError)
	}

	return s.HXRedirect(c, "/appointments")
}

// HandlePatientConfirmPage shows a patient the appointment to confirm
// GET: /p/a/:token/confirm
func (s *service) HandlePatientConfirmPage(c fiber.Ctx) error {
	appointment, err := s.TakeAppointmentByToken(c.Context(), c.Params("token", ""))
	if err != nil {
		return s.renderPatientLinkError(c, err)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AppointmentConfirmPage(vc, appointment))
}

// HandlePatientConfirm lets a patient mark an appointment as confirmed
// POST: /p/a/:token/confirm
func (s *service) HandlePatientConfirm(c fiber.Ctx) error {
	token := c.Params("token", "")
	appointment, err := s.TakeAppointmentByToken(c.Context(), token)
	if err != nil {
		return s.renderPatientLinkError(c, err)
	}
	if !appointment.PatientCanConfirm() {
		return s.renderPatientWarning(c, appointment, view.AppointmentConfirmPage)
	}

	appointment, err = s.ConfirmAppointment(c.Context(), appointment)
	if errors.Is(err, ErrAppointmentChanged) {
		return s.renderReloadedPatientWarning(c, token, view.AppointmentConfirmPage)
	}
	if err != nil {
		return s.renderAppointmentNotFoundPage(c, "Failed to confirm appointment", view.LevelError)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AppointmentConfirmPage(vc, appointment))
}

// HandlePatientCancelPage lets a patient cancel an appointment
// GET: /p/a/:token/cancel
func (s *service) HandlePatientCancelPage(c fiber.Ctx) error {
	appointment, err := s.TakeAppointmentByToken(c.Context(), c.Params("token", ""))
	if err != nil {
		return s.renderPatientLinkError(c, err)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AppointmentCancelPage(vc, appointment))
}

// HandlePatientCancel lets a patient cancel an appointment
// POST: /p/a/:token/cancel
func (s *service) HandlePatientCancel(c fiber.Ctx) error {
	token := c.Params("token", "")
	appointment, err := s.TakeAppointmentByToken(c.Context(), token)
	if err != nil {
		return s.renderPatientLinkError(c, err)
	}
	if !appointment.PatientCanCancel() {
		return s.renderPatientWarning(c, appointment, view.AppointmentCancelPage)
	}

	appointment, err = s.CancelAppointment(c.Context(), appointment)
	if errors.Is(err, ErrAppointmentChanged) {
		return s.renderReloadedPatientWarning(c, token, view.AppointmentCancelPage)
	}
	if err != nil {
		return s.renderAppointmentNotFoundPage(c, "Failed to cancel appointment", view.LevelError)
	}

	vc := view.NewCtx(c, view.WithToast("Success!", "", view.LevelSuccess))
	return view.Render(c, view.AppointmentCancelPage(vc, appointment))
}

// HandlePatientChangeDate lets a patient mark an appointment as needs
// to change the date
// POST: /p/a/:token/changedate
func (s *service) HandlePatientChangeDate(c fiber.Ctx) error {
	appointment, err := s.TakeAppointmentByToken(c.Context(), c.Params("token", ""))
	if err != nil {
		return s.renderPatientLinkError(c, err)
	}
	if !appointment.PatientCanConfirm() {
		return c.Status(fiber.StatusConflict).SendString("This appointment can no longer be changed.")
	}

	_, err = s.RequestAppointmentDateChange(c.Context(), appointment)
	if errors.Is(err, ErrAppointmentChanged) {
		return c.Status(fiber.StatusConflict).SendString("This appointment can no longer be changed.")
	}
	if err != nil {
		return s.renderAppointmentNotFoundPage(c, "Failed to request appointment date change", view.LevelError)
	}

	return c.SendString("A new date for your appointment has been requested.")
}

// HandlePriceFrg renders the price input based on clinic selected
// GET: /appointments/new/pricefrg/:id
func (s *service) HandlePriceFrg(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	clinicID := c.Params("id", "")
	if clinicID == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}

	clinic, err := gorm.G[models.Clinic](s.DB.GormDB()).
		Select("id", "uid", "price").
		Where("uid = ? AND user_id = ?", clinicID, cu.ID).
		Take(c.Context())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	vc := view.NewCtx(c)

	return view.Render(c, view.AppointmentPrice(vc, models.Appointment{}, clinic, false))
}

// HandleSearchClinics searches clinics and returns an HTML fragment to be
// replaced in the HTMX active search. The selected clinic stays listed and
// checked, and its price is left as is. A 1-2 character term or a search
// error only shows a toast, so the current list stays.
// POST: /appointments/search/clinics
func (s *service) HandleSearchClinics(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	searchTerm := strings.TrimSpace(c.FormValue("searchTerm", ""))
	if models.SearchTermTooShort(searchTerm) {
		return view.Render(c, view.CmpSearchTooShortToast())
	}

	clinics, err := s.FindClinicsBySearchTerm(c.Context(), cu.ID, searchTerm)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	clinicUID := strings.TrimSpace(c.FormValue("clinicId", ""))
	clinic, err := s.takeClinicOrZero(c.Context(), cu.ID, clinicUID)
	if err != nil {
		log.Warn("load selected clinic for search:", err)
	}
	clinics = ensureClinicListed(clinics, clinic)

	vc := view.NewCtx(c)

	return view.Render(c, view.AppointmentClinicSearchResults(vc, clinics, clinicUID))
}

// HandleSearchPatients searches patients and returns an HTML fragment to be
// replaced in the HTMX active search. The selected patient stays listed and
// checked. A 1-2 character term or a search error only shows a toast, so the
// current list stays.
// POST: /appointments/search/patients
func (s *service) HandleSearchPatients(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	searchTerm := strings.TrimSpace(c.FormValue("searchTerm", ""))
	if models.SearchTermTooShort(searchTerm) {
		return view.Render(c, view.CmpSearchTooShortToast())
	}

	patients, err := s.FindPatientsBySearchTerm(c.Context(), cu.ID, searchTerm)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return view.Render(c, view.CmpSearchFailedToast())
	}

	patientUID := strings.TrimSpace(c.FormValue("patientId", ""))
	patient, err := s.takePatientOrZero(c.Context(), cu.ID, patientUID)
	if err != nil {
		log.Warn("load selected patient for search:", err)
	}
	patients = ensurePatientListed(patients, patient)

	vc := view.NewCtx(c)

	return view.Render(c, view.AppointmentPatientPickerList(vc, patients, patientUID))
}

func (s *service) AppointmentForShowPage(ctx context.Context, userID uint, appointmentID string) (models.Appointment, error) {
	appointment := models.Appointment{UID: appointmentID}
	if appointmentID == "" || appointmentID == "new" {
		return appointment, nil
	}

	return s.TakeAppointmentByUID(ctx, userID, appointmentID)
}

func (s *service) PatientForStartPage(ctx context.Context, userID, patientID uint) (models.Patient, error) {
	patient := models.Patient{}
	err := s.DB.WithContext(ctx).
		Model(&models.Patient{}).
		Where("id = ? AND user_id = ?", patientID, userID).
		Preload("Appointments", func(tx *gorm.DB) *gorm.DB {
			return tx.Limit(1).Where("status = ?", models.AppointmentDone).Order("booked_at desc")
		}).
		Take(&patient).Error
	if err != nil {
		return models.Patient{}, err
	}

	return patient, nil
}

// renderPatientLinkError renders the not-found page for a wrong, blank or
// expired patient link token, or a load error.
func (s *service) renderPatientLinkError(c fiber.Ctx, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrTokenRequired) {
		return s.renderAppointmentNotFoundPage(c, "", view.LevelInfo)
	}

	return s.renderAppointmentNotFoundPage(c, "Failed to load appointment", view.LevelError)
}

// renderPatientWarning renders a patient link page for an appointment that can
// no longer be changed.
func (s *service) renderPatientWarning(c fiber.Ctx, appointment models.Appointment, page func(*view.Ctx, models.Appointment) templ.Component) error {
	vc := view.NewCtx(c, view.WithToast("This appointment can no longer be changed", "", view.LevelWarning))
	return view.Render(c, page(vc, appointment))
}

// renderReloadedPatientWarning reloads an appointment that changed during the
// request and renders its page with the warning.
func (s *service) renderReloadedPatientWarning(c fiber.Ctx, token string, page func(*view.Ctx, models.Appointment) templ.Component) error {
	appointment, err := s.TakeAppointmentByToken(c.Context(), token)
	if err != nil {
		return s.renderPatientLinkError(c, err)
	}

	return s.renderPatientWarning(c, appointment, page)
}

func (s *service) renderAppointmentNotFoundPage(c fiber.Ctx, toast string, level view.Level) error {
	vc := view.NewCtx(c, view.WithToast(toast, "", level))
	return view.Render(c, view.AppointmentNotFoundPage(vc))
}

// appointmentFilterFromQuery reads the appointments index filters from the
// query string.
func appointmentFilterFromQuery(c fiber.Ctx) appointmentFilter {
	return appointmentFilter{
		PatientUID: c.Query("patientId", ""),
		ClinicUID:  c.Query("clinicId", ""),
		Timeframe:  libtime.ParseTimeframe(c.Query("timeframe", "")),
		SearchTerm: strings.TrimSpace(c.Query("searchTerm", "")),
		Statuses:   models.ParseAppointmentStatuses(c.Query("status", "")),
	}
}
