package mailer

import (
	"bytes"
	"context"
	"errors"
	"miconsul/internal/lib/appenv"

	"gopkg.in/gomail.v2"
)

func ResetPassword(env *appenv.Env, email, token string) error {
	if env == nil {
		return errors.New("mailer reset password requires non-nil env")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", env.EmailFromAddress)
	m.SetHeader("To", email)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Scaffold: Reset Your Password!")

	url := env.AppProtocol + "://" + env.AppDomain + "/resetpassword/change/" + token
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

	d := gomail.NewDialer(env.EmailSMTPURL, 587, dialerUsername(env), dialerPassword(env))

	// Send Email
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
