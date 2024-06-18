package mailer

import (
	"bytes"
	"context"
	"errors"
	"os"

	"miconsul/internal/model"
	"gopkg.in/gomail.v2"
)

func SendAppointmentBookedEmail(appointment model.Appointment) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_SENDER"))
	m.SetHeader("To", appointment.Patient.Email)
	m.SetAddressHeader("Bcc", "edgarsilva.dev@gmail.com", "edgarsilva")
	m.SetHeader("Subject", "Miconsul:"+l("es-MX", "email.confirm_appointment_title"))

	emailHTML := bytes.Buffer{}
	if err := AppointmentBookedEmail(appointment).Render(context.Background(), &emailHTML); err != nil {
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

func SendAppointmentReminderEmail(appointment model.Appointment) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_SENDER"))
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
