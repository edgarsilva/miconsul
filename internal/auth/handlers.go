package auth

import (
	"fmt"
	"os"
	"rtx-blog/internal/database"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HandleSignup creates a new user if email and password are valid
// POST: /auth/signup
func (s *service) HandleSignup(c *fiber.Ctx) error {
	email, password, err := bodyParams(c)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("incorrect email or password")
	}

	if err := s.signup(email, password); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("failed to signup user")
	}

	return c.SendStatus(fiber.StatusOK)
}

// HandleLogin compares hash and password and sets the user Auth session cookie
// if the email & password combination are valid
//
// POST: /auth/login
func (s *service) HandleLogin(c *fiber.Ctx) error {
	email, password, err := bodyParams(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("incorrect email or password")
	}

	user, err := s.fetchUser(email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("incorrect email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("incorrect email or password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": user.Email,
		"uid": user.UID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("incorrect email or password")
	}

	c.Cookie(newCookie("Auth", user.UID, time.Minute*5))
	c.Cookie(newCookie("JWT", tokenStr, time.Minute*5))

	return c.JSON(user)
}

// HandleValidate validates the uses auth session is still valid
//
// POST: /auth/validate
func (s *service) HandleLogout(c *fiber.Ctx) error {
	s.SessionDestroy(c)
	invalidateCookies(c)

	return c.Redirect("/")
}

// HandleValidate validates the uses auth session is still valid
//
// POST: /auth/validate
func (s *service) HandleValidate(c *fiber.Ctx) error {
	type Cookies struct {
		Auth string `cookie:"Auth"`
		JWT  string `cookie:"JWT"`
	}

	cookies := Cookies{}
	if err := c.CookieParser(&cookies); err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Can't Authenticate")
	}

	return c.SendStatus(fiber.StatusOK)
}

// HandleShowUser returns a JSON database.User if the session is valid
// GET: /auth/show
func (s *service) HandleShowUser(c *fiber.Ctx) error {
	uid := c.Locals("userID")
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

// AuthParams extracts email and password from the request body
func bodyParams(c *fiber.Ctx) (email, password string, err error) {
	type params struct {
		Email    string `json:"name" xml:"name" form:"email"`
		Password string `json:"password" xml:"pass" form:"password"`
	}

	p := params{}
	if err := c.BodyParser(&p); err != nil {
		return "", "", fmt.Errorf("couldn't parse email or password from body: %q", err)
	}

	email = p.Email
	password = p.Password

	return email, password, nil
}

// newCookie creates a new cookie and returns a pointer to the cookie
func newCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{
		Name:    name,
		Value:   value,
		Expires: time.Now().Add(validFor),
		// MaxAge:   60 * 5,
		Secure:   os.Getenv("env") == "production",
		HTTPOnly: true,
	}
}

func invalidateCookies(c *fiber.Ctx) {
	c.Cookie(newCookie("Auth", "", time.Hour*24))
	c.Cookie(newCookie("JWT", "", time.Hour*24))
}
