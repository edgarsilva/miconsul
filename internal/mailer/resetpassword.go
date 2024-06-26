package mailer

import (
	"bytes"
	"context"
	"errors"
	"os"

	"gopkg.in/gomail.v2"
)

func ResetPassword(email, token string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_FROM_ADDR"))
	m.SetHeader("To", "edgarsilva.dev@gmail.com")
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Scaffold: Reset Your Password!")

	url := "http://localhost:8080/resetpassword/change/" + token
	emailHTML := bytes.Buffer{}
	if err := ResetPasswordTpl(email, url).Render(context.Background(), &emailHTML); err != nil {
		return errors.New("couldn't create HTML from templ comp to send email")
	}
	m.SetBody("text/html", emailHTML.String())

	// cwd, err := os.Getwd()
	// if err != nil {
	// return err
	// }
	// log.Info("sent CWD ->", cwd)
	// m.Attach(cwd + "/public/images/ripple-pic.jpg")

	d := gomail.NewDialer("smtp.gmail.com", 587, dialerUsername(), dialerPassword())

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
