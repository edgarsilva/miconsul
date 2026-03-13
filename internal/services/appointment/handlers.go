package appointment

import (
	"context"
	"errors"
	"strings"
	"time"

	"miconsul/internal/lib/amount"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HandleIndexPage renders the appointments page HTML
// GET: /appointments
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	patientID := c.Query("patientId", "")
	patient, err := s.selectedPatientFromQuery(c, cu.ID, patientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?toast=Selected patient does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load selected patient&level=error")
	}

	clinicID := c.Query("clinicId", "")
	clinic, err := s.selectedClinicFromQuery(c, cu.ID, clinicID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?toast=Selected clinic does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load selected clinic&level=error")
	}
	timeframe := c.Query("timeframe", "day")
	appointments, err := s.FindAppointmentsBy(c.Context(), cu.ID, patientID, clinicID, timeframe)
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load appointments&level=error")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.AppointmentsPage(vc, appointments, patient, clinic))
}

// HandleShowPage renders the appointment create/edit page.
// GET: /appointments/new
// GET: /appointments/:id
func (s *service) HandleShowPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	appointment, err := s.AppointmentForShowPage(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, ErrIDRequired) {
		return s.Redirect(c, "/appointments?toast=Invalid appointment id&level=error")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?toast=The appointment does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load appointment&level=error")
	}

	clinics, err := s.FindRecentClinicsByUserID(c.Context(), cu.ID, 10)
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load clinics&level=error")
	}

	patients, err := s.FindRecentPatientsByUserID(c.Context(), cu.ID, 10)
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load patients&level=error")
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)

	return view.Render(c, view.AppointmentPage(vc, appointment, patients, clinics))
}

// HandleStartPage renders the appointment start page HTML
// GET: /appointments/:id/start
func (s *service) HandleStartPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}

	appointment, err := s.TakeAppointmentByID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, ErrIDRequired) {
		redirectPath := "/appointments?toast=Invalid appointment id&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || appointment.ID == "" {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/appointments?toast=Failed to load appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusInternalServerError)
	}

	patient, err := s.PatientForStartPage(c.Context(), cu.ID, appointment.PatientID)
	if errors.Is(err, ErrIDRequired) || errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/appointments?toast=The appointment patient does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/appointments?toast=Failed to load appointment patient&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusInternalServerError)
	}

	appointment.Patient = patient

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentStartPage(appointment, vc))
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
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}

	input := appointmentCompleteInput{}
	err := c.Bind().Body(&input)
	if err != nil {
		redirectPath := "/appointments/" + appointmentID + "?toast=Invalid appointment input&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}

	appointment := appointmentCompleteUpdates{
		Status:       model.ApntStatusDone,
		Observations: input.Observations,
		Conclusions:  input.Conclusions,
		Summary:      input.Summary,
		Notes:        input.Notes,
	}

	err = s.CompleteAppointmentByID(c.Context(), cu.ID, appointmentID, appointment)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	return s.respondWithRedirect(c, "/appointments", fiber.StatusOK)
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

	bookedAt = libtime.NewInTimezone(bookedAt, model.DefaultTimezone)
	bookedAt = bookedAt.UTC()
	input := appointmentUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		redirectPath := "/appointments/new?toast=Invalid appointment input&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}

	appointment := model.Appointment{
		Token:        xid.New("tkn_"),
		UserID:       cu.ID,
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Timezone:     model.DefaultTimezone,
		Price:        amount.StrToAmount(priceValue),
		ClinicID:     input.ClinicID,
		PatientID:    input.PatientID,
		Duration:     input.Duration,
	}

	err = s.CreateAppointment(c.Context(), &appointment)
	if err != nil {
		redirectPath := "/appointments?toast=failed to create appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	appointment, err = s.TakeAppointmentByID(c.Context(), cu.ID, appointment.ID)
	if err != nil {
		redirectPath := "/appointments?toast=Appointment created, but failed to load related records"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusOK)
	}

	err = s.SendBookedAlert(appointment)
	if err != nil {
		redirectPath := "/appointments?toast=Appointment created, but failed to queue alert"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusOK)
	}

	redirectPath := "/appointments?toast=New appointment created"
	return s.respondWithRedirect(c, redirectPath, fiber.StatusOK)
}

// HandleUpdate updates an appointment record for the current user.
// PATCH: /appointments/:id
// POST: /appointments/:id/patch
func (s *service) HandleUpdate(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return s.Redirect(c, "/appointments?msg=can't update without an id")
	}

	bookedAtStr := c.FormValue("bookedAt", "")
	bookedAt, err := time.Parse(view.FormTimeFormat, bookedAtStr)
	if err != nil {
		bookedAt = time.Now()
	}

	bookedAt = libtime.NewInTimezone(bookedAt, model.DefaultTimezone)
	bookedAt = bookedAt.UTC()
	input := appointmentUpsertInput{}
	err = c.Bind().Body(&input)
	if err != nil {
		redirectPath := "/appointments/" + appointmentID + "?toast=Invalid appointment input&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusBadRequest)
	}

	appointment := appointmentPatchUpdates{
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Price:        amount.StrToAmount(c.FormValue("price", "")),
		ClinicID:     input.ClinicID,
		PatientID:    input.PatientID,
		Duration:     input.Duration,
	}

	err = s.UpdateAppointmentByID(c.Context(), cu.ID, appointmentID, appointment)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}

	if err != nil {
		redirectPath := "/appointments/" + appointmentID + "?toast=failed to update appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	return s.respondWithRedirect(c, "/appointments?toast=Appointment saved", fiber.StatusOK)
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
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}

	appointment := appointmentCancelUpdates{
		Status:     model.ApntStatusCanceled,
		CanceledAt: time.Now(),
	}
	err := s.CancelAppointmentByID(c.Context(), cu.ID, appointmentID, appointment)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusNotFound)
	}
	if err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		return s.respondWithRedirect(c, redirectPath, fiber.StatusUnprocessableEntity)
	}

	return s.respondWithRedirect(c, "/appointments", fiber.StatusOK)
}

// HandleDelete deletes an appointment record from the DB.
// DELETE: /appointments/:id
// POST: /appointments/:id/delete
func (s *service) HandleDelete(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		return s.Redirect(c, "/appointments?msg=can't delete without an id")
	}

	err := s.DeleteAppointmentByID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?msg=can't find that appointment")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?msg=failed to delete that appointment")
	}

	return s.respondWithRedirect(c, "/appointments", fiber.StatusOK)
}

// HandlePatientConfirm lets a patient mark an appointment as confirmed
// GET: /appointments/:id/patient/confirm/:token
func (s *service) HandlePatientConfirm(c fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return s.Redirect(c, "/signin?toast=Failed to confirm appointment&level=error")
	}

	err := s.ConfirmAppointmentByIDAndToken(c.Context(), appointmentID, token)
	if err != nil {
		redirectPath := "/signin?toast=Failed to confirm appointment&level=error"
		return s.Redirect(c, redirectPath)
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", ""),
	)

	return view.Render(c, view.AppointmentConfirmPage(vc))
}

// HandlePatientCancelPage lets a patient cancel an appointment
// GET: /appointments/:id/patient/cancel/:token
func (s *service) HandlePatientCancelPage(c fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return s.renderAppointmentNotFoundPage(c, c.Query("toast", ""), "")
	}

	appointment, err := s.TakeAppointmentByIDAndToken(c.Context(), appointmentID, token)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.renderAppointmentNotFoundPage(c, c.Query("toast", ""), "")
	}
	if err != nil {
		return s.Redirect(c, "/signin?toast=Failed to load appointment&level=error")
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentCancelPage(vc, appointment))
}

// HandlePatientCancel lets a patient cancel an appointment
// POST: /appointments/:id/patient/cancel/:token
func (s *service) HandlePatientCancel(c fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return s.Redirect(c, "/signin?toast=Failed to cancel appointment&level=error")
	}

	err := s.CancelAppointmentByIDAndToken(c.Context(), appointmentID, token)
	if err != nil {
		return s.Redirect(c, "/signin?toast=Failed to cancel appointment&level=error")
	}

	appointment, err := s.TakeAppointmentByIDAndToken(c.Context(), appointmentID, token)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.renderAppointmentNotFoundPage(c, "Appointment not found", "warning")
	}
	if err != nil {
		return s.Redirect(c, "/signin?toast=Failed to load appointment&level=error")
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "Success!")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", "success"),
	)
	return view.Render(c, view.AppointmentCancelPage(vc, appointment))
}

// HandlePatientChangeDate lets a patient mark an appointment as needs
// to change the date
// GET: /appointments/:id/patient/changedate/:token
func (s *service) HandlePatientChangeDate(c fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return s.Redirect(c, "/signin?toast=Can't find that appointment&level=error")
	}

	err := s.RequestAppointmentDateChangeByIDAndToken(c.Context(), appointmentID, token)
	if err != nil {
		redirectPath := "/signin?toast=Failed to request appointment date change&level=error"
		return s.Redirect(c, redirectPath)
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

	clinic, err := gorm.G[model.Clinic](s.DB.GormDB()).
		Select("id", "price").
		Where("id = ? AND user_id = ?", clinicID, cu.ID).
		Take(c.Context())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c, view.WithToast(toast, "", ""))

	return view.Render(c, view.ApptPrice(vc, model.Appointment{}, clinic, false))
}

// HandleSearchClinics searches clinics and returns an HTML fragment to be
// replacesd in the HTMX active search
// POST: /appointments/search/clinics
func (s *service) HandleSearchClinics(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	searchTerm := c.FormValue("searchTerm", "")
	clinics, err := s.FindClinicsBySearchTerm(c.Context(), cu.ID, searchTerm)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	return view.Render(c, view.ApptSearchClinicsFrg(vc, clinics))
}

func (s *service) AppointmentForShowPage(ctx context.Context, userID, appointmentID string) (model.Appointment, error) {
	appointment := model.Appointment{ID: appointmentID}
	if appointmentID == "" || appointmentID == "new" {
		return appointment, nil
	}

	return s.TakeAppointmentByID(ctx, userID, appointmentID)
}

func (s *service) PatientForStartPage(ctx context.Context, userID, patientID string) (model.Patient, error) {
	return s.TakePatientByIDWithLastDoneAppointment(ctx, userID, patientID)
}

func (s *service) selectedPatientFromQuery(c fiber.Ctx, userID, patientID string) (model.Patient, error) {
	patientID = strings.TrimSpace(patientID)
	if patientID == "" {
		return model.Patient{}, nil
	}

	patient, err := s.TakePatientByID(c.Context(), userID, patientID)
	if err != nil {
		return model.Patient{}, err
	}

	return patient, nil
}

func (s *service) selectedClinicFromQuery(c fiber.Ctx, userID, clinicID string) (model.Clinic, error) {
	clinicID = strings.TrimSpace(clinicID)
	if clinicID == "" {
		return model.Clinic{}, nil
	}

	clinic, err := s.TakeClinicByID(c.Context(), userID, clinicID)
	if err != nil {
		return model.Clinic{}, err
	}

	return clinic, nil
}

func (s *service) renderAppointmentNotFoundPage(c fiber.Ctx, toast, level string) error {
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithToast(toast, "", level))
	return view.Render(c, view.AppointmentNotFoundPage(vc))
}

func (s *service) respondWithRedirect(c fiber.Ctx, redirectPath string, htmxStatus int) error {
	if s.NotHTMX(c) {
		return s.Redirect(c, redirectPath)
	}

	c.Set("HX-Location", redirectPath)
	return c.SendStatus(htmxStatus)
}
