package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"miconsul/internal/model"
	"os"

	"gopkg.in/gomail.v2"
)

func SendAppointmentBookedEmail(appointment model.Appointment) error {
	if appointment.Patient.ID == "" || appointment.Clinic.ID == "" {
		fmt.Println(errors.New("appointment Clinic or Patient association is missing, they must be Preloaded"))
	}

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_FROM_ADDR"))
	m.SetHeader("To", appointment.Patient.Email)
	m.SetAddressHeader("Bcc", "edgarsilva.dev@gmail.com", "edgarsilva")
	m.SetHeader("Subject", "Miconsul:"+l("es-MX", "email.confirm_appointment_title"))

	emailHTML := bytes.Buffer{}
	if err := AppointmentBookedEmail(appointment).Render(context.Background(), &emailHTML); err != nil {
		fmt.Println(errors.New("couldn't create HTML from templ comp to send email"))
	}
	m.SetBody("text/html", emailHTML.String())

	smtpURL := os.Getenv("EMAIL_SMTP_URL")
	d := gomail.NewDialer(smtpURL, 587, dialerUsername(), dialerPassword())

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		fmt.Println(err)
	}

	return nil
}

func SendAppointmentReminderEmail(appointment model.Appointment) error {
	if appointment.Patient.ID == "" || appointment.Clinic.ID == "" {
		return errors.New("appointment Clinic or Patient association is missing, they must be Preloaded")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_FROM_ADDR"))
	m.SetHeader("To", appointment.Patient.Email)
	m.SetAddressHeader("Bcc", "edgarsilva.dev@gmail.com", "edgarsilva")
	m.SetHeader("Subject", "Miconsul:"+l("es-MX", "email.confirm_appointment_title"))

	emailHTML := bytes.Buffer{}
	if err := AppointmentReminderEmail(appointment).Render(context.Background(), &emailHTML); err != nil {
		return errors.New("couldn't create HTML from templ comp to send email")
	}
	m.SetBody("text/html", emailHTML.String())

	d := gomail.NewDialer("smtp.gmail.com", 587, dialerUsername(), dialerPassword())

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
