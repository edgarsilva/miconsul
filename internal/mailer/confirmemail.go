package mailer

import (
	"bytes"
	"context"
	"errors"
	"miconsul/internal/lib/appenv"

	"gopkg.in/gomail.v2"
)

func ConfirmEmail(env *appenv.Env, email, token string) error {
	if env == nil {
		return errors.New("mailer confirm email requires non-nil env")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", env.EmailFromAddress)
	m.SetHeader("To", email)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Scaffold: Confirm your email to login to Miconsul!")

	url := "https://" + env.AppDomain + "/signup/confirm/" + token
	emailHTML := bytes.Buffer{}
	if err := ConfirmEmailTpl(email, url).Render(context.Background(), &emailHTML); err != nil {
		return errors.New("couldn't create HTML from templ comp to send email")
	}
	m.SetBody("text/html", emailHTML.String())

	dialer := gomail.NewDialer(env.EmailSMTPURL, 587, dialerUsername(env), dialerPassword(env))

	// Send Email
	if err := dialer.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
