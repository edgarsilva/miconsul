package appointment

import (
	"fmt"

	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
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
	s.DB.Model(&cu).Association("Appointments").Find(&appointments)

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.AppointmentsPage(appointments, appointment, layoutProps))
}

// HandleCreateAppointment inserts a new appointment record for the CurrentUser
//
// POST: /appointments
func (s *service) HandleCreateAppointment(c *fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	if err != nil {
		return c.Redirect("/login")
	}

	appointment := model.Appointment{
		UserID: cu.ID,
	}
	c.BodyParser(&appointment)
	fmt.Println("-------------------->")
	fmt.Printf("-------------------->%#v", appointment.ClinicID)
	fmt.Printf("-------------------->%#v", appointment.PatientID)
	fmt.Println("-------------------->")

	result := s.DB.Create(&appointment)

	if !s.IsHTMX(c) {
		if err := result.Error; err != nil {
			return c.Redirect("/appointments?err=failed to create appointment")
		}

		return c.Redirect("/appointments/" + appointment.ID)
	}

	c.Set("HX-Push-Url", "/appointments/"+appointment.ID)
	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.AppointmentsPage([]model.Appointment{}, appointment, layoutProps))
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

	appointment := model.Appointment{
		UserID: cu.ID,
	}
	c.BodyParser(&appointment)

	res := s.DB.Where("id = ? AND user_id = ?", appointmentID, cu.ID).Updates(&appointment)
	if err := res.Error; err != nil {
		return c.Redirect("/appointments?err=failed to update appointment")
	}

	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	if isHTMX == "" {
		return c.Redirect("/appointments/" + appointment.ID)
	}

	return c.Redirect("/appointments/"+appointmentID, fiber.StatusSeeOther)
}

// HandleDeleteAppointment deletes a appointment record from the DB
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
