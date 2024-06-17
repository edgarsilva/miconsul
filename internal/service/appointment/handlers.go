package appointment

import (
	"strconv"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// HandleAppointmentsPage renders the appointments page HTML
//
// GET: /appointments
func (s *service) HandleAppointmentsPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	patientID := c.Query("patientId", "")
	patient, _ := s.GetPatientByID(c, patientID)
	c.Locals("patient", patient)

	clinicID := c.Query("clinicId", "")
	clinic, _ := s.GetClinicByID(c, clinicID)
	c.Locals("clinic", clinic)

	timeframe := c.Query("timeframe", "day")
	appointments, _ := s.GetAppointmentsBy(c, cu, patientID, clinicID, timeframe)

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)

	return view.Render(c, view.AppointmentsPage(vc, appointments))
}

// HandleNewAppointmentPage renders the new appointments form page
//
// GET: /appointments/new
func (s *service) HandleAppointmentPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	id := c.Params("id", "")
	appointment := model.Appointment{}
	appointment.ID = id
	if id != "" && id != "new" {
		s.DB.Model(&appointment).Where("id", id).Take(&appointment)
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

// HandleBeginAppointmentPage renders the appointments page HTML
//
// GET: /appointments/:id/begin
func (s *service) HandleAppointmentBeginPage(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	id := c.Params("id", "")
	appointment := model.Appointment{
		UserID: cu.ID,
	}
	s.DB.Model(&appointment).Where("id = ?", id).Take(&appointment)
	if id == "" || appointment.ID == "" {
		c.Set("HX-Location", "/appointments?toast=The appointment does not exist&level=warning")
		return c.Redirect("/appointments?toast=The appointment does not exist&level=warning", fiber.StatusSeeOther)
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
	return view.Render(c, view.AppointmentBeginPage(appointment, vc))
}

// HandleCreateAppointment inserts a new appointment record for the CurrentUser
//
// POST: /appointments
func (s *service) HandleCreateAppointment(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	bookedAtValue := c.FormValue("bookedAt", "")
	bookedAt, err := time.Parse(view.FormTimeFormat, bookedAtValue)
	if err != nil {
		bookedAt = time.Now()
	}

	priceValue := c.FormValue("price", "")
	price := 0
	if priceValue != "" {
		pricef, _ := strconv.ParseFloat(priceValue, 64)
		price = int(pricef * 100)
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
		Price:        price,
	}
	c.BodyParser(&appointment)

	result := s.DB.Create(&appointment)
	if err := result.Error; err != nil {
		redirectPath := "/appointments?toast=failed to create appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	s.DB.Model(&appointment).Preload("Clinic").Preload("Patient").Take(&appointment)
	s.SendBookedAlert(appointment)

	if !s.IsHTMX(c) {
		return c.Redirect("/appointments?toast=New appointment created")
	}

	c.Set("HX-Location", "/appointments?toast=New appointment created")
	return c.SendStatus(fiber.StatusOK)
}

// HandleUpdateAppointment updates a appointment record for the CurrentUser
//
// PATCH: /appointments/:id
// POST: /appointments/:id/patch
func (s *service) HandleUpdateAppointment(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		return c.Redirect("/appointments?msg=can't update without an id", fiber.StatusSeeOther)
	}

	bookedAtStr := c.FormValue("bookedAt", "")
	bookedAt, err := time.Parse(view.FormTimeFormat, bookedAtStr)
	if err != nil {
		bookedAt = time.Now()
	}

	priceValue := c.FormValue("price", "")
	price := 0
	if priceValue != "" {
		pricef, _ := strconv.ParseFloat(priceValue, 64)
		price = int(pricef * 100)
	}

	appointment := model.Appointment{
		UserID:   cu.ID,
		BookedAt: bookedAt,
		Price:    price,
	}
	c.BodyParser(&appointment)

	result := s.DB.Where("id = ? AND user_id = ?", appointmentID, cu.ID).Updates(&appointment)
	if err := result.Error; err != nil {
		redirectPath := "/appointments/" + appointment.ID + "?toast=failed to update appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	if !s.IsHTMX(c) {
		return c.Redirect("/appointments?toast=Appointment saved")
	}

	c.Set("HX-Location", "/appointments?toast=Appointment saved")
	return c.SendStatus(fiber.StatusOK)
}

// HandleBeginAppointmentPage renders the appointments page HTML
//
// GET: /appointments/:id/Begin
func (s *service) HandleAppointmentDone(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		c.Set("HX-Location", redirectPath)
		return c.Redirect(redirectPath, fiber.StatusSeeOther)
	}

	appointment := model.Appointment{
		UserID: cu.ID,
	}
	c.BodyParser(&appointment)

	res := s.DB.Where("id = ? AND user_id = ?", appointmentID, cu.ID).Updates(&appointment)
	if err := res.Error; err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		c.Set("HX-Location", redirectPath)
		return c.Redirect(redirectPath, fiber.StatusSeeOther)
	}

	if !s.IsHTMX(c) {
		return c.Redirect("/appointments", fiber.StatusSeeOther)
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandleAppointmentCancel cancels an appointment
//
// POST: /appointments/:id/cancel
func (s *service) HandleAppointmentCancel(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		appointmentID = c.FormValue("id", "")
	}

	if appointmentID == "" {
		redirectPath := "/appointments?toast=Can't find that appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath, fiber.StatusSeeOther)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusNotFound)
	}

	appointment := model.Appointment{
		UserID:     cu.ID,
		Status:     model.ApntStatusCanceled,
		CanceledAt: time.Now(),
	}
	res := s.DB.Where("id = ? AND user_id = ?", appointmentID, cu.ID).Updates(&appointment)
	if err := res.Error; err != nil {
		redirectPath := "/appointments?toast=Failed to update appointment&level=error"
		if !s.IsHTMX(c) {
			return c.Redirect(redirectPath, fiber.StatusSeeOther)
		}

		c.Set("HX-Location", redirectPath)
		return c.SendStatus(fiber.StatusUnprocessableEntity)
	}

	if !s.IsHTMX(c) {
		return c.Redirect("/appointments", fiber.StatusSeeOther)
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandleAppointmentDelete deletes a appointment record from the DB
//
// DELETE: /appointments/:id
// POST: /appointments/:id/delete
func (s *service) HandleDeleteAppointment(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	appointmentID := c.Params("ID", "")
	if appointmentID == "" {
		return c.Redirect("/appointments?msg=can't delete without an id", fiber.StatusSeeOther)
	}

	appointment := model.Appointment{
		UserID: cu.ID,
	}

	res := s.DB.Where("id = ? AND user_id = ?", appointmentID, cu.ID).Delete(&appointment)
	if err := res.Error; err != nil {
		return c.Redirect("/appointments?msg=failed to delete that appointment", fiber.StatusSeeOther)
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect("/appointments", fiber.StatusSeeOther)
	}

	c.Set("HX-Location", "/appointments")
	return c.SendStatus(fiber.StatusOK)
}

// HandlePatientConfirm lets a patient mark an appointment as confirmed
//
//	GET: /patient/:id/confirm/:token
func (s *service) HandlePatientConfirm(c *fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return c.Redirect("/login?toast=Can't find that appointment&level=error")
	}

	appointment := model.Appointment{
		// Token:       "",
		ConfirmedAt: time.Now(),
		Status:      model.ApntStatusConfirmed,
	}
	res := s.DB.Select("ConfirmedAt", "Status").Where("id = ? AND token = ?", appointmentID, token).Updates(&appointment)
	if err := res.Error; err != nil {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
		return c.Redirect(redirectPath, fiber.StatusSeeOther)
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentConfirmPage(vc))
}

// HandlePatientCancel lets a patient cancel an appointment
//
// GET: /patient/:id/cancel/:token
func (s *service) HandlePatientCancel(c *fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return c.Redirect("/login?toast=Can't find that appointment&level=error")
	}

	appointment := model.Appointment{}
	appointmentUpds := model.Appointment{
		// Token:      "",
		CanceledAt: time.Now(),
		Status:     model.ApntStatusCanceled,
	}
	s.DB.Preload("Clinic").Where("id = ? AND token = ?", appointmentID, token).Take(&appointment)
	res := s.DB.
		Select("CanceledAt", "Status").
		Where("id = ? AND token = ?", appointmentID, token).
		Updates(&appointmentUpds)
	if err := res.Error; err != nil {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
		return c.Redirect(redirectPath, fiber.StatusSeeOther)
	}

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "some toast")
	vc, _ := view.NewCtx(c,
		view.WithTheme(theme), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentCancelPage(appointment, vc))
}

// HandlePatientChangeDate lets a patient mark an appointment as needs
// to change the date
//
// GET: /patient/:id/changedate/:token
func (s *service) HandlePatientChangeDate(c *fiber.Ctx) error {
	appointmentID := c.Params("id", "")
	token := c.Params("token", "")
	if appointmentID == "" || token == "" {
		return c.Redirect("/login?toast=Can't find that appointment&level=error")
	}

	appointment := model.Appointment{
		Token:     "",
		PendingAt: time.Now(),
		Status:    model.ApntStatusPending,
	}
	res := s.DB.Select("Token", "PendingAt", "Status").Where("id = ? AND token = ?", appointmentID, token).Updates(&appointment)
	if err := res.Error; err != nil {
		redirectPath := "/login?toast=Failed to confirm appointment&level=error"
		return c.Redirect(redirectPath, fiber.StatusSeeOther)
	}

	return c.SendString("A new date for your appointment has been requested.")
}
