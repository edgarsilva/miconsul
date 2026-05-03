package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"gopkg.in/gomail.v2"
)

func SendAppointmentBookedEmail(env *appenv.Env, appointment models.Appointment) error {
	if env == nil {
		return errors.New("appointment booked mailer requires non-nil env")
	}

	if appointment.Patient.ID == 0 || appointment.Clinic.ID == 0 {
		fmt.Println(errors.New("appointment Clinic or Patient association is missing, they must be Preloaded"))
	}

	m := gomail.NewMessage()
	m.SetHeader("From", env.EmailFromAddress)
	m.SetHeader("To", appointment.Patient.Email)
	m.SetAddressHeader("Bcc", "edgarsilva.dev@gmail.com", "edgarsilva")
	m.SetHeader("Subject", "Miconsul:"+l("es-MX", "email.confirm_appointment_title"))

	emailHTML := bytes.Buffer{}
	if err := AppointmentBookedEmail(env, appointment).Render(context.Background(), &emailHTML); err != nil {
		fmt.Println(errors.New("couldn't create HTML from templ comp to send email"))
	}
	m.SetBody("text/html", emailHTML.String())

	d := gomail.NewDialer(env.EmailSMTPURL, 587, dialerUsername(env), dialerPassword(env))

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		fmt.Println(err)
	}

	return nil
}

func SendAppointmentReminderEmail(env *appenv.Env, appointment models.Appointment) error {
	if env == nil {
		return errors.New("appointment reminder mailer requires non-nil env")
	}

	if appointment.Patient.ID == 0 || appointment.Clinic.ID == 0 {
		return errors.New("appointment Clinic or Patient association is missing, they must be Preloaded")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", env.EmailFromAddress)
	m.SetHeader("To", appointment.Patient.Email)
	m.SetAddressHeader("Bcc", "edgarsilva.dev@gmail.com", "edgarsilva")
	m.SetHeader("Subject", "Miconsul:"+l("es-MX", "email.confirm_appointment_title"))

	emailHTML := bytes.Buffer{}
	if err := AppointmentReminderEmail(env, appointment).Render(context.Background(), &emailHTML); err != nil {
		return errors.New("couldn't create HTML from templ comp to send email")
	}
	m.SetBody("text/html", emailHTML.String())

	d := gomail.NewDialer(env.EmailSMTPURL, 587, dialerUsername(env), dialerPassword(env))

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
