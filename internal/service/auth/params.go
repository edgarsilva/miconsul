package auth

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

// credentialsFromRequest extracts email and password from request form values.
func credentialsFromRequest(c fiber.Ctx) (email, password string, err error) {
	email = c.FormValue("email", "")
	password = c.FormValue("password", "")
	if email == "" || password == "" {
		err = errors.New("email and password can't be blank")
		return email, password, err
	}

	return email, password, nil
}

// resetPasswordEmailFromRequest returns email from form, params, or query.
func resetPasswordEmailFromRequest(c fiber.Ctx) (string, error) {
	email := c.FormValue("email", "")
	if email != "" {
		return email, nil
	}

	email = c.Params("email", "")
	if email != "" {
		return email, nil
	}

	email = c.Query("email", "")
	if email != "" {
		return email, nil
	}

	return "", errors.New("email can't be blank")
}
