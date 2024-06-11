package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/gomail.v2"
)

func ConfirmEmail(email, token string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("EMAIL_SENDER"))
	m.SetHeader("To", email)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Scaffold: Confirm your email to login to Miconsul!")

	url := "https://" + os.Getenv("APP_DOMAIN") + "/signup/confirm/" + token
	emailHTML := bytes.Buffer{}
	if err := ConfirmEmailTpl(email, url).Render(context.Background(), &emailHTML); err != nil {
		return errors.New("couldn't create HTML from templ comp to send email")
	}
	m.SetBody("text/html", emailHTML.String())

	emailsecret := os.Getenv("EMAIL_SECRET")
	emailsecret = strings.Trim(emailsecret, "\"")
	dialer := gomail.NewDialer("smtp.gmail.com", 587, os.Getenv("EMAIL_SENDER"), emailsecret)

	// Send Email
	if err := dialer.DialAndSend(m); err != nil {
		fmt.Println("-------> Failed to send email:", err)
		return nil
	}

	return nil
}
