package auth

import (
	"errors"
	"os"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/view"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HandleLoginPage returns the login page html
//
// GET: /login
func (s *service) HandleLoginPage(c *fiber.Ctx) error {
	cu, _ := s.CurrentUser(c)

	if cu.IsLoggedIn() {
		return c.Redirect("/todos")
	}

	return s.RenderLoginPage(c, nil)
}

// HandleLogin compares hash and password and sets the user Auth session cookie
// if the email & password combination are valid
//
// POST: /auth/login
func (s *service) HandleLogin(c *fiber.Ctx) error {
	email, password, err := bodyParams(c)
	respErr := errors.New("incorrect email and password combination")
	if err != nil {
		return s.RenderLoginPage(c, respErr)
	}

	user, err := s.fetchUser(email)
	if err != nil {
		return s.RenderLoginPage(c, respErr)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return s.RenderLoginPage(c, respErr)
	}

	switch c.Accepts("text/plain", "text/html", "application/json") {
	case "text/html":
		c.Cookie(newCookie("Auth", user.UID, time.Hour*24))
		return c.Redirect("/todos")
	case "application/json":
		token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
			"sub": user.Email,
			"uid": user.UID,
			"exp": time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			return s.RenderLoginPage(c, err)
		}
		c.Cookie(newCookie("JWT", tokenStr, time.Minute*5))
		resp := map[string]string{
			"token": tokenStr,
		}
		return c.JSON(resp)
	default:
		c.Cookie(newCookie("Auth", user.UID, time.Hour*24))
		return c.SendString("Login Successful")
	}
}

// HandleSignup creates a new user if email and password are valid
//
// POST: /auth/signup
func (s *service) HandleSignup(c *fiber.Ctx) error {
	email, password, err := bodyParams(c)
	if err != nil {
		theme := s.SessionGet(c, "theme", "light")
		layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme))
		email := c.Query("email")

		return view.Render(c, view.LoginPage(email, err, layoutProps))
	}

	if err := s.signup(email, password); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString(err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}

// HandleValidate validates the uses auth session is still valid
//
// POST: /auth/validate
func (s *service) HandleLogout(c *fiber.Ctx) error {
	s.SessionDestroy(c)
	invalidateCookies(c)

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/")
		return c.SendStatus(fiber.StatusTemporaryRedirect)
	}

	return c.Redirect("/")
}

// HandleValidate validates the uses auth session is still valid
//
// POST: /auth/validate
func (s *service) HandleValidate(c *fiber.Ctx) error {
	_, err := s.CurrentUser(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	return c.SendStatus(fiber.StatusOK)
}

// HandleShowUser returns a JSON database.User if the session is valid
//
// GET: /auth/show
func (s *service) HandleShowUser(c *fiber.Ctx) error {
	uid := c.Locals("uid")
	if uid == nil {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	if len(uid.(string)) == 0 {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	var user database.User
	if result := s.DB.Where("uid = ?", uid).Take(&user); result.Error != nil {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	res := struct{ User database.User }{
		User: user,
	}

	return c.JSON(res)
}
