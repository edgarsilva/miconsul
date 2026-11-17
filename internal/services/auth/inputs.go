package auth

import (
	"errors"
	"strings"
)

// signinInput is the API sign-in request body.
type signinInput struct {
	Email      string `json:"email" form:"email"`
	Password   string `json:"password" form:"password"`
	RememberMe bool   `json:"remember_me" form:"remember_me"`
}

// signupInput is the signup form.
type signupInput struct {
	Email    string `form:"email"`
	Password string `form:"password"`
	Confirm  string `form:"confirm"`
}

// trim returns the input with the spaces around the email removed.
func (in signupInput) trim() signupInput {
	in.Email = strings.TrimSpace(in.Email)

	return in
}

// validate returns the error to show the user when the form is blank, the
// password confirmation differs or the email is malformed. A password of only
// spaces is blank.
func (in signupInput) validate() error {
	if in.Email == "" || strings.TrimSpace(in.Password) == "" {
		return errors.New("email and password can't be blank")
	}

	if in.Confirm == "" || in.Password != in.Confirm {
		return errors.New("passwords don't match")
	}

	if !isEmail(in.Email) {
		return errors.New("email address is invalid")
	}

	return nil
}
