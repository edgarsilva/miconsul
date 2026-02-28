package appointment

import (
	"fmt"
	"miconsul/internal/lib/handlerutils"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// HandleIndexPage renders the appointments page HTML
//
// GET: /appointments
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	patientID := c.Query("patientId", "")
	patient, _ := s.GetPatientByID(c, patientID)
	c.Locals("patient", patient)

	clinicID := c.Query("clinicId", "")
	clinic, _ := s.GetClinicByID(c, clinicID)
	c.Locals("clinic", clinic)

	timeframe := c.Query("timeframe", "day")
	appointments, _ := s.GetAppointmentsBy(c, cu, patientID, clinicID, timeframe)

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.AppointmentsPage(vc, appointments))
}

// HandleNewAppointmentPage renders the new appointments form page
//
// GET: /appointments/new
func (s *service) HandleShowPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	id := c.Params("id", "")
	appointment := model.Appointment{}
	appointment.ID = id
	if id != "" && id != "new" {
		appointment, _ = gorm.G[model.Appointment](s.DB.DB).Where("id = ?", id).Take(c.Context())
	}

	clinics := []model.Clinic{}
	s.DB.Model(&cu).Order("created_at desc").Limit(10).Association("Clinics").Find(&clinics)

	patients := []model.Patient{}
	s.DB.Model(&cu).Order("created_at desc").Limit(10).Association("Patients").Find(&patients)

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)

	return view.Render(c, view.AppointmentPage(vc, appointment, patients, clinics))
}

// HandleCommencePage renders the appointments page HTML
//
// GET: /appointments/:id/commence
func (s *service) HandleCommencePage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	id := c.Params("id", "")
	appointment := model.Appointment{
		UserID: cu.ID,
	}
	appointment, _ = gorm.G[model.Appointment](s.DB.DB).
		Where("id = ? AND user_id = ?", id, cu.ID).
		Take(c.Context())
	if id == "" || appointment.ID == "" {
		c.Set("HX-Location", "/appointments?toast=The appointment does not exist&level=warning")
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments?toast=The appointment does not exist&level=warning")
	}

	patient := model.Patient{
		UserID: cu.ID,
	}
	patient.ID = appointment.PatientID
	s.DB.Model(patient).Preload("Appointments", func(tx *gorm.DB) *gorm.DB {
		return tx.Limit(1).Where("status = ?", model.ApntStatusDone).Order("booked_at desc")
	}).Take(&patient)

	appointment.Patient = patient

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentCommencePage(appointment, vc))
}

// HandleConclude handles the request to mark an appointment as concluded/done
//
// GET: /appointments/:id/conclude
func (s *service) HandleConclude(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		c.Set("HX-Location", redirectPath)
		return c.Redirect().Status(fiber.StatusSeeOther).To(redirectPath)
	}

	appointment := model.Appointment{
		UserID: cu.ID,
	}
	c.Bind().Body(&appointment)

	_, err = gorm.G[model.Appointment](s.DB.DB).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		c.Set("HX-Location", redirectPath)
		return c.Redirect().Status(fiber.StatusSeeOther).To(redirectPath)
	}

	if !s.IsHTMX(c) {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments")
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandleCreate inserts a new appointment record for the CurrentUser
//
// POST: /appointments
func (s *service) HandleCreate(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
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
	c.Bind().Body(&appointment)

	err = gorm.G[model.Appointment](s.DB.DB).Create(c.Context(), &appointment)
	if err != nil {
		redirectPath := "/appointments?toast=failed to create appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect().To(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	s.DB.Model(&appointment).Preload("Clinic").Preload("Patient").Take(&appointment)
	s.SendBookedAlert(appointment)

	if !s.IsHTMX(c) {
		return c.Redirect().To("/appointments?toast=New appointment created")
	}

	c.Set("HX-Location", "/appointments?toast=New appointment created")
	return c.SendStatus(fiber.StatusOK)
}

// HandleUpdateAppointment updates a appointment record for the CurrentUser
//
// PATCH: /appointments/:id
// POST: /appointments/:id/patch
func (s *service) HandleUpdate(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments?msg=can't update without an id")
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
	c.Bind().Body(&appointment)

	_, err = gorm.G[model.Appointment](s.DB.DB).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/appointments/" + appointment.ID + "?toast=failed to update appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect().To(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	if !s.IsHTMX(c) {
		return c.Redirect().To("/appointments?toast=Appointment saved")
	}

	c.Set("HX-Location", "/appointments?toast=Appointment saved")
	return c.SendStatus(fiber.StatusOK)
}

// HandleCancel cancels an appointment
//
// PATCH: /appointments/:id/cancel
func (s *service) HandleCancel(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect().Status(fiber.StatusSeeOther).To(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusNotFound)
	}

	appointment := model.Appointment{
		UserID:     cu.ID,
		Status:     model.ApntStatusCanceled,
		CanceledAt: time.Now(),
	}
	_, err = gorm.G[model.Appointment](s.DB.DB).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect().Status(fiber.StatusSeeOther).To(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	if !s.IsHTMX(c) {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments")
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandleDelete deletes a appointment record from the DB
//
// DELETE: /appointments/:id
// POST: /appointments/:id/delete
func (s *service) HandleDelete(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments?msg=can't delete without an id")
	}

	_, err = gorm.G[model.Appointment](s.DB.DB).
		Where("id = ? AND user_id = ?", appointmentID, cu.ID).
		Delete(c.Context())
	if err != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments?msg=failed to delete that appointment")
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/appointments")
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandlePatientConfirm lets a patient mark an appointment as confirmed
//
//	GET: /appointments/:id/patient/confirm/:token
func (s *service) HandlePatientConfirm(c fiber.Ctx) error {
	apptID := c.Params("id", "")
	token := c.Params("token", "")
	appt := model.Appointment{
		// Token:       "",
		ConfirmedAt: time.Now(),
		Status:      model.ApntStatusConfirmed,
	}
	_, err := gorm.G[model.Appointment](s.DB.DB).
		Select("ConfirmedAt", "Status").
		Where("id = ? AND token = ?", apptID, token).
		Updates(c.Context(), appt)
	if err != nil {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
		return c.Redirect().Status(fiber.StatusSeeOther).To(redirectPath)
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", ""),
	)

	if appt.ID == "" {
		return view.Render(c, view.AppointmentNotFoundPage(vc))
	}

	return view.Render(c, view.AppointmentConfirmPage(vc))
}

// HandlePatientCancelPage lets a patient cancel an appointment
//
//	GET: /appointments/:id/patient/cancel/:token
func (s *service) HandlePatientCancelPage(c fiber.Ctx) error {
	apptID := c.Params("id", "")
	token := c.Params("token", "")
	appt := model.Appointment{}
	s.DB.Preload("Clinic").Preload("User").Where("id = ? AND token = ?", apptID, token).Take(&appt)

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
//
//	POST: /appointments/:id/patient/cancel/:token
func (s *service) HandlePatientCancel(c fiber.Ctx) error {
	apptID := c.Params("id", "")
	token := c.Params("token", "")
	apptUpds := model.Appointment{
		CanceledAt: time.Now(),
		Status:     model.ApntStatusCanceled,
	}
	rowsAffected, err := gorm.G[model.Appointment](s.DB.DB).
		Select("CanceledAt", "Status").
		Where("id = ? AND token = ?", apptID, token).
		Updates(c.Context(), apptUpds)
	if err != nil || rowsAffected != 1 {
		fmt.Println("------>", err, rowsAffected)
	}

	appt := model.Appointment{}
	s.DB.Preload("Clinic").Where("id = ? AND token = ?", apptID, token).Take(&appt)

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "Success!")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", "sucess"),
	)
	return view.Render(c, view.AppointmentCancelPage(vc, appt))
}

// HandlePatientChangeDate lets a patient mark an appointment as needs
// to change the date
//
// GET: /appointments/:id/patient/changedate/:token
func (s *service) HandlePatientChangeDate(c fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return c.Redirect().To("/login?toast=Can't find that appointment&level=error")
	}

	appointment := model.Appointment{
		PendingAt: time.Now(),
		Status:    model.ApntStatusPending,
	}
	_, err := gorm.G[model.Appointment](s.DB.DB).
		Select("Token", "PendingAt", "Status").
		Where("id = ? AND token = ?", appointmentID, token).
		Updates(c.Context(), appointment)
	if err != nil {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
		return c.Redirect().Status(fiber.StatusSeeOther).To(redirectPath)
	}

	return c.SendString("A new date for your appointment has been requested.")
}

// HandlePriceFrg renders the price input based on clinic selected
//
// GET: /appointments/new/pricefrg/:id
func (s *service) HandlePriceFrg(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	id := c.Params("id", "")
	clinic := model.Clinic{
		UserID: cu.ID,
		ID:     id,
	}

	clinic, _ = gorm.G[model.Clinic](s.DB.DB).
		Select("id", "price").
		Where("id = ? AND user_id = ?", clinic.ID, clinic.UserID).
		Take(c.Context())

	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c, view.WithToast(toast, "", ""))

	return view.Render(c, view.ApptPrice(vc, model.Appointment{}, clinic, false))
}

// HandleSearchClinics searches clinics and returns an HTML fragment to be
// replacesd in the HTMX active search
//
// POST: /appointments/search/clinics
func (s *service) HandleSearchClinics(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect().To("/login")
	}

	queryStr := c.FormValue("query", "")
	clinics := []model.Clinic{}

	dbquery := s.DB.Model(&cu)
	if queryStr != "" {
		dbquery.Scopes(model.GlobalFTS(queryStr))
	} else {
		dbquery.Order("created_at desc")
	}
	dbquery.Limit(10).Association("Clinics").Find(&clinics)

	// time.Sleep(time.Second * 2)
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))

	return view.Render(c, view.ApptSearchClinicsFrg(vc, clinics))
}
