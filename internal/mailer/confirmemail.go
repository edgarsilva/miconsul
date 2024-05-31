package mailer

import (
	"bytes"
	"context"
	"errors"
	"os"

	"gopkg.in/gomail.v2"
)

func ConfirmEmail(email, token string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_SENDER"))
	m.SetHeader("To", "edgarsilva.dev@gmail.com")
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Scaffold: Confirm your email!")

	url := "http://localhost:8080/signup/confirm/" + token
	emailHTML := bytes.Buffer{}
	if err := ConfirmEmailTpl(email, url).Render(context.Background(), &emailHTML); err != nil {
		return errors.New("couldn't create HTML from templ comp to send email")
	}
	m.SetBody("text/html", emailHTML.String())

	d := gomail.NewDialer("smtp.gmail.com", 587, os.Getenv("EMAIL_SENDER"), os.Getenv("EMAIL_SECRET"))

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
