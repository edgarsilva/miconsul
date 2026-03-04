package appointment

import (
	"errors"
	"time"

	"miconsul/internal/lib/handlerutils"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HandleIndexPage renders the appointments page HTML
// GET: /appointments
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	patientID := c.Query("patientId", "")
	patient, err := s.GetPatientByID(c, patientID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?toast=Selected patient does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load selected patient&level=error")
	}
	c.Locals("patient", patient)

	clinicID := c.Query("clinicId", "")
	clinic, err := s.GetClinicByID(c, clinicID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?toast=Selected clinic does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load selected clinic&level=error")
	}
	c.Locals("clinic", clinic)

	timeframe := c.Query("timeframe", "day")
	appointments, err := s.GetAppointmentsBy(c, cu, patientID, clinicID, timeframe)
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load appointments&level=error")
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.AppointmentsPage(vc, appointments))
}

// HandleShowPage renders the appointment create/edit page.
// GET: /appointments/new
// GET: /appointments/:id
func (s *service) HandleShowPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	appointmentID := c.Params("id", "")
	appointment, err := s.AppointmentForShowPage(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Redirect(c, "/appointments?toast=The appointment does not exist&level=warning")
	}
	if err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load appointment&level=error")
	}

	clinics := []model.Clinic{}
	if err := s.DB.Model(&cu).Order("created_at desc").Limit(10).Association("Clinics").Find(&clinics); err != nil {
		return s.Redirect(c, "/appointments?toast=Failed to load clinics&level=error")
	}

	patients := []model.Patient{}
	if err := s.DB.Model(&cu).Order("created_at desc").Limit(10).Association("Patients").Find(&patients); err != nil {
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
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		c.Set("HX-Location", "/appointments?toast=The appointment does not exist&level=warning")
		return s.Redirect(c, "/appointments?toast=The appointment does not exist&level=warning")
	}

	appointment, err := s.TakeAppointmentByID(c.Context(), cu.ID, appointmentID)
	if errors.Is(err, gorm.ErrRecordNotFound) || appointment.ID == "" {
		c.Set("HX-Location", "/appointments?toast=The appointment does not exist&level=warning")
		return s.Redirect(c, "/appointments?toast=The appointment does not exist&level=warning")
	}
	if err != nil {
		c.Set("HX-Location", "/appointments?toast=Failed to load appointment&level=error")
		return s.Redirect(c, "/appointments?toast=Failed to load appointment&level=error")
	}

	patient, err := s.PatientForStartPage(c.Context(), cu.ID, appointment.PatientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Set("HX-Location", "/appointments?toast=The appointment patient does not exist&level=warning")
			return s.Redirect(c, "/appointments?toast=The appointment patient does not exist&level=warning")
		}

		c.Set("HX-Location", "/appointments?toast=Failed to load appointment patient&level=error")
		return s.Redirect(c, "/appointments?toast=Failed to load appointment patient&level=error")
	}

	appointment.Patient = patient

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentStartPage(appointment, vc))
}

// HandleConclude handles the request to mark an appointment as concluded/done
// POST: /appointments/:id/conclude
func (s *service) HandleConclude(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		c.Set("HX-Location", redirectPath)
		return s.Redirect(c, redirectPath)
	}

	appointment := model.Appointment{
		UserID: cu.ID,
	}
	if err := c.Bind().Body(&appointment); err != nil {
		redirectPath := "/appointments/" + appointmentID + "?toast=Invalid appointment input&level=error"
		c.Set("HX-Location", redirectPath)
		return s.Redirect(c, redirectPath)
	}

	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		c.Set("HX-Location", redirectPath)
		return s.Redirect(c, redirectPath)
	}
	if rowsAffected != 1 {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		c.Set("HX-Location", redirectPath)
		return s.Redirect(c, redirectPath)
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/appointments")
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandleCreate inserts a new appointment record for the CurrentUser
// POST: /appointments
func (s *service) HandleCreate(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	bookedAtValue := c.FormValue("bookedAt", "")
	bookedAt, err := time.Parse(view.FormTimeFormat, bookedAtValue)
	if err != nil {
		bookedAt = time.Now()
	}

	priceValue := c.FormValue("price", "")

	bookedAt = libtime.NewInTimezone(bookedAt, model.DefaultTimezone)
	bookedAt = bookedAt.UTC()
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
		Price:        handlerutils.StrToAmount(priceValue),
	}
	if err := c.Bind().Body(&appointment); err != nil {
		redirectPath := "/appointments/new?toast=Invalid appointment input&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	err = gorm.G[model.Appointment](s.DB.GormDB()).Create(c.Context(), &appointment)
	if err != nil {
		redirectPath := "/appointments?toast=failed to create appointment&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	toastMsg := "New appointment created"
	if err := s.DB.Model(&appointment).Preload("Clinic").Preload("Patient").Take(&appointment).Error; err != nil {
		toastMsg = "Appointment created, but failed to load related records"
	} else if err := s.SendBookedAlert(appointment); err != nil {
		toastMsg = "Appointment created, but failed to queue alert"
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/appointments?toast="+toastMsg)
	}

	c.Set("HX-Location", "/appointments?toast="+toastMsg)
	return c.SendStatus(fiber.StatusOK)
}

// HandleUpdate updates an appointment record for the current user.
// PATCH: /appointments/:id
// POST: /appointments/:id/patch
func (s *service) HandleUpdate(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

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
	appointment := model.Appointment{
		UserID:       cu.ID,
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Price:        handlerutils.StrToAmount(c.FormValue("price", "")),
	}
	if err := c.Bind().Body(&appointment); err != nil {
		redirectPath := "/appointments/" + appointmentID + "?toast=Invalid appointment input&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/appointments/" + appointmentID + "?toast=failed to update appointment&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}
	if rowsAffected != 1 {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/appointments?toast=Appointment saved")
	}

	c.Set("HX-Location", "/appointments?toast=Appointment saved")
	return c.SendStatus(fiber.StatusOK)
}

// HandleCancel cancels an appointment
// POST: /appointments/:id/cancel
func (s *service) HandleCancel(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusNotFound)
	}

	appointment := model.Appointment{
		UserID:     cu.ID,
		Status:     model.ApntStatusCanceled,
		CanceledAt: time.Now(),
	}
	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}
	if rowsAffected != 1 {
		redirectPath := "/appointments?toast=The appointment does not exist&level=warning"
		if !s.IsHTMX(c) {
			return s.Redirect(c, redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusNotFound)
	}

	if !s.IsHTMX(c) {
		return s.Redirect(c, "/appointments")
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandleDelete deletes an appointment record from the DB.
// DELETE: /appointments/:id
// POST: /appointments/:id/delete
func (s *service) HandleDelete(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	appointmentID := c.Params("id", "")
	if appointmentID == "" {
		return s.Redirect(c, "/appointments?msg=can't delete without an id")
	}

	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Delete(c.Context())
	if err != nil {
		return s.Redirect(c, "/appointments?msg=failed to delete that appointment")
	}
	if rowsAffected != 1 {
		return s.Redirect(c, "/appointments?msg=can't find that appointment")
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return s.Redirect(c, "/appointments")
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandlePatientConfirm lets a patient mark an appointment as confirmed
// GET: /appointments/:id/patient/confirm/:token
func (s *service) HandlePatientConfirm(c fiber.Ctx) error {
	apptID := c.Params("id", "")
	token := c.Params("token", "")
	appt := model.Appointment{
		// Token:       "",
		ConfirmedAt: time.Now(),
		Status:      model.ApntStatusConfirmed,
	}
	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Select("ConfirmedAt", "Status").
		Where("id = ? AND token = ?", apptID, token).
		Updates(c.Context(), appt)
	if err != nil || rowsAffected != 1 {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
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
	apptID := c.Params("id", "")
	token := c.Params("token", "")
	appt := model.Appointment{}
	if err := s.DB.Preload("Clinic").Preload("User").Where("id = ? AND token = ?", apptID, token).Take(&appt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			theme := s.SessionUITheme(c)
			toast := c.Query("toast", "")
			vc, _ := view.NewCtx(c,
				view.WithTheme(theme), view.WithToast(toast, "", ""),
			)
			return view.Render(c, view.AppointmentNotFoundPage(vc))
		}

		return s.Redirect(c, "/login?toast=Failed to load appointment&level=error")
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", ""),
	)

	if appt.ID == "" {
		return view.Render(c, view.AppointmentNotFoundPage(vc))
	}

	return view.Render(c, view.AppointmentCancelPage(vc, appt))
}

// HandlePatientCancel lets a patient cancel an appointment
// POST: /appointments/:id/patient/cancel/:token
func (s *service) HandlePatientCancel(c fiber.Ctx) error {
	apptID := c.Params("id", "")
	token := c.Params("token", "")
	apptUpds := model.Appointment{
		CanceledAt: time.Now(),
		Status:     model.ApntStatusCanceled,
	}
	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Select("CanceledAt", "Status").
		Where("id = ? AND token = ?", apptID, token).
		Updates(c.Context(), apptUpds)
	if err != nil || rowsAffected != 1 {
		return s.Redirect(c, "/login?toast=Failed to cancel appointment&level=error")
	}

	appt := model.Appointment{}
	if err := s.DB.Preload("Clinic").Where("id = ? AND token = ?", apptID, token).Take(&appt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			theme := s.SessionUITheme(c)
			vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithToast("Appointment not found", "", "warning"))
			return view.Render(c, view.AppointmentNotFoundPage(vc))
		}

		return s.Redirect(c, "/login?toast=Failed to load appointment&level=error")
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "Success!")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", "sucess"),
	)
	return view.Render(c, view.AppointmentCancelPage(vc, appt))
}

// HandlePatientChangeDate lets a patient mark an appointment as needs
// to change the date
// GET: /appointments/:id/patient/changedate/:token
func (s *service) HandlePatientChangeDate(c fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return s.Redirect(c, "/login?toast=Can't find that appointment&level=error")
	}

	appointment := model.Appointment{
		PendingAt: time.Now(),
		Status:    model.ApntStatusPending,
	}
	rowsAffected, err := gorm.G[model.Appointment](s.DB.GormDB()).
		Select("Token", "PendingAt", "Status").
		Where("id = ? AND token = ?", appointmentID, token).
		Updates(c.Context(), appointment)
	if err != nil || rowsAffected != 1 {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
		return s.Redirect(c, redirectPath)
	}

	return c.SendString("A new date for your appointment has been requested.")
}

// HandlePriceFrg renders the price input based on clinic selected
// GET: /appointments/new/pricefrg/:id
func (s *service) HandlePriceFrg(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	id := c.Params("id", "")
	clinic := model.Clinic{
		UserID: cu.ID,
		ID:     id,
	}

	clinic, err = gorm.G[model.Clinic](s.DB.GormDB()).
		Select("id", "price").
		Where("id = ? AND user_id = ?", clinic.ID, clinic.UserID).
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
	cu, err := s.CurrentUser(c)
	if err != nil {
		return s.Redirect(c, "/login")
	}

	queryStr := c.FormValue("query", "")
	clinics := []model.Clinic{}

	dbquery := s.DB.Model(&cu)
	if queryStr != "" {
		dbquery = dbquery.Scopes(model.GlobalFTS(queryStr))
	} else {
		dbquery = dbquery.Order("created_at desc")
	}
	if err := dbquery.Limit(10).Association("Clinics").Find(&clinics); err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// time.Sleep(time.Second * 2)
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	return view.Render(c, view.ApptSearchClinicsFrg(vc, clinics))
}
