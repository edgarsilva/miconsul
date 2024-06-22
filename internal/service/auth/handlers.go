package auth

import (
	"errors"
	"fmt"
	"miconsul/internal/lib/xid"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// HandleLoginPage returns the login page html
//
// GET: /login
func (s *service) HandleLoginPage(c *fiber.Ctx) error {
	if s.LogtoEnabled() {
		return c.Redirect("/logto/signin", fiber.StatusSeeOther)
	}

	cu, _ := s.CurrentUser(c)
	if cu.IsLoggedIn() {
		return c.Redirect("/")
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme))
	email := c.Query("email", "")
	msg := c.Query("msg", "")
	return view.Render(c, view.LoginPage(email, msg, nil, vc))
}

// HandleLogin compares hash and password and sets the user Auth session cookie
// if the email & password combination are valid
//
// POST: /login
func (s *service) HandleLogin(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme))
	respErr := errors.New("incorrect email and password combination")

	email, password, err := authParams(c)
	if err != nil {
		return view.Render(c, view.LoginPage(email, "", respErr, vc))
	}

	user, err := s.userFetch(email)
	if err != nil {
		return view.Render(c, view.LoginPage(email, "", respErr, vc))
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return view.Render(c, view.LoginPage(email, "", respErr, vc))
	}

	if user.ConfirmEmailToken != "" {
		err := errors.New("email pending confirmation, check your inbox")
		return view.Render(c, view.LoginPage(email, "", err, vc))
	}

	validFor := time.Duration(24)
	rememberMe := c.FormValue("remember_me", "") != ""
	if rememberMe {
		validFor *= 7
	}

	switch c.Accepts("text/plain", "text/html", "application/json") {
	case "application/json":
		// TODO: HandleLogin maybe accept JWT for application/json
		return c.SendStatus(fiber.StatusServiceUnavailable)
	default:
		jwt, err := JWTCreateToken(user.Email, user.ID)
		if err != nil {
			return c.Redirect("/?msg=Failed to login, please try again")
		}
		c.Cookie(newCookie("Auth", jwt, time.Hour*validFor))
		return c.Redirect("/?timeframe=day")
	}
}

// HandleSignupPage returns the Signup form page html
//
// GET: /signup
func (s *service) HandleSignupPage(c *fiber.Ctx) error {
	cu, _ := s.CurrentUser(c)

	if cu.IsLoggedIn() {
		return c.Redirect("/todos")
	}

	msg := c.Query("msg", "")
	err := errors.New(msg)
	if msg == "" {
		err = nil
	}
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme))

	return view.Render(c, view.SignupPage(vc, "", err))
}

// HandleSignup creates a new user if email and password are valid
//
// POST: /signup
func (s *service) HandleSignup(c *fiber.Ctx) error {
	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme))
	email, password, err := authParams(c)
	if err != nil {
		return view.Render(c, view.SignupPage(vc, email, err))
	}

	confirm := c.FormValue("confirm", "")
	if confirm == "" || password != confirm {
		err := errors.New("passwords don't match")
		return view.Render(c, view.SignupPage(vc, email, err))
	}

	err = s.userPendingConfirmation(email)
	if err != nil {
		token := randToken()
		s.userUpdateConfirmToken(email, token)
		go mailer.ConfirmEmail(email, token)
		return c.Redirect("/login?msg=check your inbox, we'll re-send a confirmation link")
	}

	if err := s.signup(email, password); err != nil {
		return view.Render(c, view.SignupPage(vc, email, err))
	}

	return c.Redirect("/login?msg=check your inbox to confirm your email")
}

// HandleEmailConfirmation creates a new user if email and password are valid
//
// POST: /signup/confirmemail
func (s *service) HandleSignupConfirmEmail(c *fiber.Ctx) error {
	token := c.Params("token", "")
	if token == "" {
		return c.Redirect("/login?msg=unable to confirm email, try login instead")
	}

	user := model.User{}
	err := s.DB.
		Model(&model.User{}).
		Select("id, email, confirm_email_token").
		Where("confirm_email_token = ? AND confirm_email_expires_at > ?", token, time.Now()).
		Take(&user).Error
	if err != nil {
		return c.Redirect("/login?msg=we couldn't verify your account, pls try again")
	}

	result := s.DB.
		Model(&user).
		Select("ConfirmEmailToken", "ConfirmEmailExpiresAt").
		Where("confirm_email_token = ? AND confirm_email_expires_at > ?", token, time.Now()).
		Updates(&model.User{})
	if result.Error != nil {
		return c.Redirect("/login?msg=Email confirmed, you should be able to login now")
	}

	jwt, err := JWTCreateToken(user.Email, user.ID)
	if err != nil {
		return c.Redirect("/login?msg=Email confirmed, you should be able to login now")
	}

	c.Cookie(newCookie("Auth", jwt, time.Hour*24))
	return c.Redirect("/login?msg=Email confirmed, you should be able to login now")
}

// HandleLogout calles sessionDestroy and invalidateCookies then redirects to
// /login
//
// ALL: /logout
func (s *service) HandleLogout(c *fiber.Ctx) error {
	s.SessionDestroy(c)
	invalidateSessionCookies(c)

	redirectURL := "/login"
	if s.LogtoEnabled() {
		redirectURL = "/logto/signout"
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", redirectURL)
		return c.SendStatus(fiber.StatusTemporaryRedirect)
	}

	return c.Redirect(redirectURL)
}

// HandlePageResetPassword renders the HTML reset password page/form
//
// GET: /resetpassword
func (s *service) HandleResetPasswordPage(c *fiber.Ctx) error {
	vc, _ := view.NewCtx(c)

	msg := s.SessionGet(c, "msg", "")
	if msg == "" {
		msg = c.Query("msg", "")
	}

	return view.Render(c, view.ResetPasswordPage(vc, "", msg, "", nil))
}

// HandleResetPasswordForm sends a new reset password link to the email provided
// as query param, url param or body param
//
// POST: /resetpassword
func (s *service) HandleResetPassword(c *fiber.Ctx) error {
	vc, _ := view.NewCtx(c)
	email, err := resetPasswordEmailParam(c)
	if err != nil {
		errView := errors.New("email can't be blank")
		return view.Render(c, view.ResetPasswordPage(vc, email, "", "", errView))
	}

	user := model.User{}
	err = s.DB.Model(&model.User{}).Select("id", "name").Where("email = ?", email).Take(&user).Error
	if err != nil {
		errView := errors.New("user not found with that email")
		return view.Render(c, view.ResetPasswordPage(vc, email, "", "", errView))
	}

	token, err := resetPasswordToken()
	if err != nil {
		return c.Redirect("/resetpassword")
	}

	fmt.Println("Test USER ->", user)
	user.ResetToken = token
	user.ResetTokenExpiresAt = time.Now().Add(time.Hour * 1)
	s.DB.Model(&user).Select("ResetToken", "ResetTokenExpiresAt").Updates(&user)

	go mailer.ResetPassword(email, token)

	return view.Render(c, view.ResetPasswordPage(vc, email, "", "check your email for a reset password link", nil))
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
	vc, _ := view.NewCtx(c)
	return view.Render(c, view.ResetPasswordChangePage(email, token, nonce, nil, vc))
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

	vc, _ := view.NewCtx(c)
	password := c.FormValue("password", "")
	if password == "" {
		err := errors.New("password can't be blank")
		return view.Render(c, view.ResetPasswordChangePage(email, token, nonce, err, vc))
	}

	confirm := c.FormValue("confirm", "")
	if confirm == "" || password != confirm {
		err := errors.New("passwords don't match")
		return view.Render(c, view.ResetPasswordChangePage(email, token, nonce, err, vc))
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

// HandleShowUser returns a JSON model.User if the session is valid
//
// GET: /auth/show
func (s *service) HandleShowUser(c *fiber.Ctx) error {
	id := c.Locals("uid")
	if id == nil {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	if len(id.(string)) == 0 {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	var user model.User
	if result := s.DB.Where("id = ?", id).Take(&user); result.Error != nil {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	res := struct{ User model.User }{
		User: user,
	}

	return c.JSON(res)
}
