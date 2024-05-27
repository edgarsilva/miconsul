package appointment

import (
	"strconv"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/util"
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

	id := c.Params("id", "")
	appointment := model.Appointment{}
	appointment.ID = id
	if id != "" && id != "new" {
		s.DB.Model(&model.Appointment{}).First(&appointment)
	}

	appointments := []model.Appointment{}
	dbquery := s.DB.Model(model.Appointment{}).Where("user_id = ?", cu.ID)

	patient := model.Patient{
		UserID: cu.ID,
	}
	patientID := c.Query("patientId", "")
	if patientID != "" {
		patient.ID = patientID
		result := s.DB.Model(&patient).Take(&patient)
		if result.RowsAffected == 1 {
			c.Locals("patient", patient)
			dbquery.Where("patient_id", patientID)
		}
	}

	timeframe := c.Query("timeframe", "")
	switch timeframe {
	case "day":
		dbquery.Scopes(model.AppointmentsBookedToday)
	case "week":
		dbquery.Scopes(model.AppointmentsBookedThisWeek)
	case "month":
		dbquery.Scopes(model.AppointmentsBookedThisMonth)
	default:
		dbquery.Where("booked_at > ?", util.BoD(time.Now()))
	}

	dbquery.Preload("Clinic").
		Preload("Patient").
		Find(&appointments)

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	layoutProps, _ := view.NewLayoutProps(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)

	return view.Render(c, view.AppointmentsPage(appointments, appointment, layoutProps))
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
		return tx.Limit(1).Order("booked_at desc")
	}).Take(&patient)

	appointment.Patient = patient

	theme := s.SessionUITheme(c)
	toast := c.Query("toast", "")
	layoutProps, _ := view.NewLayoutProps(c,
		view.WithTheme(theme), view.WithCurrentUser(cu), view.WithToast(toast, "", ""),
	)
	return view.Render(c, view.AppointmentBeginPage(appointment, layoutProps))
}

// HandleBeginAppointmentPage renders the appointments page HTML
//
// GET: /appointments/:id/Begin
func (s *service) HandleAppointmentDonePage(c *fiber.Ctx) error {
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

	costValue := c.FormValue("cost", "")
	cost := 0
	if costValue != "" {
		costf, _ := strconv.ParseFloat(costValue, 64)
		cost = int(costf * 100)
	}

	appointment := model.Appointment{
		UserID:   cu.ID,
		BookedAt: bookedAt,
		Cost:     cost,
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

	costValue := c.FormValue("cost", "")
	cost := 0
	if costValue != "" {
		costf, _ := strconv.ParseFloat(costValue, 64)
		cost = int(costf * 100)
	}

	appointment := model.Appointment{
		UserID:   cu.ID,
		BookedAt: bookedAt,
		Cost:     cost,
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

// HandleAppointmentCancel cancels an appointment
//
// POST: /appointments/:id/cancel
func (s *service) HandleAppointmentCancel(c *fiber.Ctx) error {
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
