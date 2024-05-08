package auth

import (
	"errors"
	"os"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"github.com/edgarsilva/go-scaffold/internal/mailer"
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
// POST: /login
func (s *service) HandleLogin(c *fiber.Ctx) error {
	email, password, err := authParams(c)
	respErr := errors.New("incorrect email and password combination")
	if err != nil {
		return s.RenderLoginPage(c, respErr)
	}

	user, err := s.userFetch(email)
	if err != nil {
		return s.RenderLoginPage(c, respErr)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return s.RenderLoginPage(c, respErr)
	}

	validFor := time.Duration(24)
	rememberMe := c.FormValue("remember_me", "") != ""
	if rememberMe {
		validFor *= 7
	}
	switch c.Accepts("text/plain", "text/html", "application/json") {
	case "text/html":
		c.Cookie(newCookie("Auth", user.UID, time.Hour*validFor))
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
		c.Cookie(newCookie("JWT", tokenStr, time.Hour*24))
		resp := map[string]string{
			"token": tokenStr,
		}
		return c.JSON(resp)
	default:
		c.Cookie(newCookie("Auth", user.UID, time.Hour*24))
		return c.SendString("Login Successful")
	}
}

// HandleSignupPage returns the Signup form page html
//
// GET: /login
func (s *service) HandleSignupPage(c *fiber.Ctx) error {
	cu, _ := s.CurrentUser(c)

	if cu.IsLoggedIn() {
		return c.Redirect("/todos")
	}

	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme))

	return view.Render(c, view.SignupPage("", nil, layoutProps))
}

// HandleSignup creates a new user if email and password are valid
//
// POST: /signup
func (s *service) HandleSignup(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)
	layoutProps, _ := view.NewLayoutProps(view.WithTheme(theme))

	email, password, err := authParams(c)
	if err != nil {
		return view.Render(c, view.SignupPage(email, err, layoutProps))
	}

	confirm := c.FormValue("confirm", "")
	if confirm == "" || password != confirm {
		err := errors.New("passwords don't match")
		return view.Render(c, view.SignupPage(email, err, layoutProps))
	}

	if err := s.signup(email, password); err != nil {
		return view.Render(c, view.SignupPage(email, err, layoutProps))
	}

	return c.Redirect("/login")
}

// HandleLogout calles sessionDestroy and invalidateCookies then redirects to
// /login
//
// DELETE: /logout
func (s *service) HandleLogout(c *fiber.Ctx) error {
	s.SessionDestroy(c)
	invalidateCookies(c)

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/")
		return c.SendStatus(fiber.StatusTemporaryRedirect)
	}

	return c.Redirect("/")
}

// HandleResetPasswordPage renders the HTML reset password page/form
//
// GET: /resetpassword
func (s *service) HandleResetPasswordPage(c *fiber.Ctx) error {
	layoutProps, _ := view.NewLayoutProps()

	msg := s.SessionGet(c, "msg", "")
	if msg == "" {
		msg = c.Query("msg", "")
	}

	return view.Render(c, view.ResetPasswordPage("", msg, "", nil, layoutProps))
}

// HandleResetPasswordForm sends a new reset password link to the email provided
// as query param, url param or body param
//
// POST: /resetpassword
func (s *service) HandleResetPasswordSend(c *fiber.Ctx) error {
	layoutProps, _ := view.NewLayoutProps()
	email, err := resetPasswordEmailParam(c)
	if err != nil {
		errView := errors.New("email can't be blank")
		return view.Render(c, view.ResetPasswordPage(email, "", "", errView, layoutProps))
	}

	user := database.User{}
	err = s.DB.Model(&database.User{}).Select("id, email").Where("email = ?", email).Take(&user).Error
	if err != nil {
		errView := errors.New("user not found with that email")
		return view.Render(c, view.ResetPasswordPage(email, "", "", errView, layoutProps))
	}

	token, err := resetPasswordGenToken()
	if err != nil {
		return c.Redirect("/resetpassword")
	}

	user.ResetToken = token
	user.ResetTokenExpiresAt = time.Now().Add(time.Hour * 1)
	s.DB.Model(&user).Select("ResetToken", "ResetTokenExpiresAt").Updates(&user)

	go mailer.ResetPasswordSendToken(email, token)

	return view.Render(c, view.ResetPasswordPage(email, "", "check your email for a reset password link", nil, layoutProps))
}

// HandleResetPasswordChange renders the change password form if toke/email
// combo ase valid
//
// GET: /resetpassword/change/:token
func (s *service) HandleResetPasswordChange(c *fiber.Ctx) error {
	token := c.Params("token", "")
	if token == "" {
		return c.Redirect("/resetpassword?msg=token can't be blank")
	}

	email, err := s.resetPasswordVerifyToken(token)
	if err != nil {
		return c.Redirect("/resetpassword?msg=invalid email or token")
	}

	nonce := xid.New("rpnnce")
	s.SessionSet(c, "nonce", nonce)
	layoutProps, _ := view.NewLayoutProps()
	return view.Render(c, view.ResetPasswordChangePage(email, token, nonce, nil, layoutProps))
}

// HandleResetPasswordUpdate updates the user password in the DB
//
// POST: /resetpassword/update
func (s *service) HandleResetPasswordUpdate(c *fiber.Ctx) error {
	email, err := resetPasswordEmailParam(c)
	if err != nil {
		return c.Redirect("/resetpassword?msg=something went wrong with the email, try again!")
	}

	token := c.FormValue("token", "")
	if token == "" {
		return c.Redirect("/resetpassword?msg=something went wrong with the token, try again!")
	}

	nonce := c.FormValue("nonce", "")
	cmpNonce := s.SessionGet(c, "nonce", nonce)
	if nonce == "" || nonce != cmpNonce {
		return c.Redirect("/resetpassword?msg=something went wrong with the nonce, try again!")
	}

	_, err = s.resetPasswordVerifyToken(token)
	if err != nil {
		return c.Redirect("/resetpassword?msg=seems like your token has expired, try again!")
	}

	layoutProps, _ := view.NewLayoutProps()
	password := c.FormValue("password", "")
	if password == "" {
		err := errors.New("password can't be blank")
		return view.Render(c, view.ResetPasswordChangePage(email, token, nonce, err, layoutProps))
	}

	confirm := c.FormValue("confirm", "")
	if confirm == "" || password != confirm {
		err := errors.New("passwords don't match")
		return view.Render(c, view.ResetPasswordChangePage(email, token, nonce, err, layoutProps))
	}

	_, err = s.userUpdatePassword(email, password, token)
	if err != nil {
		return c.Redirect("/resetpassword?msg=something went wrong, try again!")
	}

	return c.Redirect("/login")
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
