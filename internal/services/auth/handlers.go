package auth

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/models"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// HandleSigninPage returns the signin page html
//
// GET: /signin
func (s *service) HandleSigninPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)
	if cu.IsLoggedIn() {
		return s.Redirect(c, "/")
	}

	email := c.Query("email", "")
	msg := c.Query("msg", "")

	redirectURL, nextMsg := s.providerSigninRedirect(c, msg)
	if redirectURL != "" {
		return s.Redirect(c, redirectURL)
	}

	msg = nextMsg

	vc := view.NewCtx(c)
	return view.Render(c, view.LoginPage(email, msg, nil, vc))
}

// HandleSignin compares hash and password and sets the user Auth session cookie
// if the email & password combination are valid
//
// POST: /signin
func (s *service) HandleSignin(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "auth/handlers:HandleSignin")
	defer span.End()

	vc := view.NewCtx(c)
	respErr := errors.New("incorrect email and password combination")

	email, password, err := credentialsFromRequest(c)
	if err != nil {
		return view.Render(c, view.LoginPage(email, "", respErr, vc))
	}

	user, err := s.authenticateCredentials(ctx, email, password)
	if errors.Is(err, errAuthInvalidCredentials) {
		return view.Render(c, view.LoginPage(email, "", respErr, vc))
	}
	if errors.Is(err, errAuthPendingConfirmation) {
		err := errors.New("email pending confirmation, check your inbox")
		return view.Render(c, view.LoginPage(email, "", err, vc))
	}
	if err != nil {
		return s.respondWithRedirect(c, "/", "Failed to login, please try again")
	}

	rememberMe := c.FormValue("remember_me", "") != ""
	if err := s.issueAuthCookie(c, user, rememberMe); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create session failed")
		return s.respondWithRedirect(c, "/", "Failed to login, please try again")
	}

	return s.Redirect(c, "/?timeframe=day")
}

// HandleAPISignin authenticates a user and creates an auth cookie for API clients.
// POST: /api/auth/signin
func (s *service) HandleAPISignin(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "auth/handlers:HandleAPISignin")
	defer span.End()

	input := signinInput{}
	if err := c.Bind().Body(&input); err != nil {
		return apiError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if input.Email == "" || input.Password == "" {
		return apiError(c, fiber.StatusBadRequest, "email and password can't be blank")
	}

	user, err := s.authenticateCredentials(ctx, input.Email, input.Password)
	if errors.Is(err, errAuthInvalidCredentials) {
		return apiError(c, fiber.StatusUnauthorized, "invalid credentials")
	}
	if errors.Is(err, errAuthPendingConfirmation) {
		return apiError(c, fiber.StatusForbidden, "email pending confirmation")
	}
	if err != nil {
		return apiError(c, fiber.StatusInternalServerError, "authentication failed")
	}

	if err := s.issueAuthCookie(c, user, input.RememberMe); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create session failed")
		return apiError(c, fiber.StatusInternalServerError, "failed to create session")
	}

	return apiOK(c, fiber.Map{"ok": true})
}

// HandleSignupPage returns the Signup form page html
//
// GET: /signup
func (s *service) HandleSignupPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	if cu.IsLoggedIn() {
		return s.Redirect(c, "/todos")
	}

	msg := c.Query("msg", "")
	err := errors.New(msg)
	if msg == "" {
		err = nil
	}
	vc := view.NewCtx(c)

	return view.Render(c, view.SignupPage(vc, "", err))
}

// HandleSignup creates a new user if email and password are valid
//
// POST: /signup
func (s *service) HandleSignup(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "auth/handlers:HandleSignup")
	defer span.End()
	span.SetAttributes(attribute.String("auth.provider", "local"))

	vc := view.NewCtx(c)
	email, password, err := credentialsFromRequest(c)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid signup credentials input")
		return view.Render(c, view.SignupPage(vc, email, err))
	}
	span.SetAttributes(attribute.String("auth.signup.email", strings.ToLower(strings.TrimSpace(email))))

	confirm := c.FormValue("confirm", "")
	if confirm == "" || password != confirm {
		err := errors.New("passwords don't match")
		span.RecordError(err)
		span.SetStatus(codes.Error, "signup password confirmation mismatch")
		return view.Render(c, view.SignupPage(vc, email, err))
	}

	err = s.userPendingConfirmation(ctx, email)
	if err != nil {
		if resendErr := s.resendPendingConfirmation(ctx, email); resendErr != nil {
			span.RecordError(resendErr)
			span.SetStatus(codes.Error, "resend confirmation failed")
			return view.Render(c, view.SignupPage(vc, email, errors.New("failed to resend confirmation email, please try again")))
		}
		span.SetStatus(codes.Ok, "pending confirmation resent")
		return s.respondWithRedirect(c, "/signin", "check your inbox, we'll re-send a confirmation link")
	}

	if err := s.signup(ctx, email, password); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "signup failed")
		if errors.Is(err, errAuthConfirmEmailSend) {
			err = errors.New("failed to send confirmation email, please try again")
		}
		return view.Render(c, view.SignupPage(vc, email, err))
	}
	span.SetStatus(codes.Ok, "signup confirmation email sent")

	return s.respondWithRedirect(c, "/signin", "check your inbox to confirm your email")
}

// HandleSignupConfirmEmail validates an email confirmation token.
// GET: /signup/confirm/:token
func (s *service) HandleSignupConfirmEmail(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "auth/handlers:HandleSignupConfirmEmail")
	defer span.End()
	span.SetAttributes(attribute.String("auth.provider", "local"))

	token := c.Params("token", "")
	if token == "" {
		span.SetStatus(codes.Error, "missing confirmation token")
		return s.respondWithRedirect(c, "/signin", "unable to confirm email, try signin instead")
	}

	user, err := gorm.G[models.User](s.DB.GormDB()).
		Select("uid, email, confirm_email_token").
		Where("confirm_email_token = ? AND confirm_email_expires_at > ?", token, time.Now()).
		Take(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid or expired confirmation token")
		return s.respondWithRedirect(c, "/signin", "we couldn't verify your account, pls try again")
	}
	span.SetAttributes(attribute.String("auth.signup.confirm_email", strings.ToLower(strings.TrimSpace(user.Email))))

	_, err = gorm.G[models.User](s.DB.GormDB()).
		Select("confirm_email_token", "confirm_email_expires_at").
		Where("confirm_email_token = ? AND confirm_email_expires_at > ?", token, time.Now()).
		Updates(ctx, models.User{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "confirmation token clear failed")
		return s.respondWithRedirect(c, "/signin", "Email confirmed, you should be able to signin now")
	}

	jwt, err := JWTCreateToken(s.AppEnv(), user.Email, user.UID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "jwt creation after confirmation failed")
		return s.respondWithRedirect(c, "/signin", "Email confirmed, you should be able to signin now")
	}

	c.Cookie(s.NewCookie("Auth", jwt, time.Hour*24))
	span.SetStatus(codes.Ok, "email confirmed")
	return s.respondWithRedirect(c, "/signin", "Email confirmed, you should be able to signin now")
}

func apiError(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

func apiOK(c fiber.Ctx, payload fiber.Map) error {
	return c.Status(fiber.StatusOK).JSON(payload)
}

func (s *service) respondWithRedirect(c fiber.Ctx, path, message string) error {
	if strings.TrimSpace(message) == "" {
		return s.Redirect(c, path)
	}

	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}

	return s.Redirect(c, path+sep+"msg="+url.QueryEscape(message))
}

// HandleLogout destroys session state and redirects to login.
// ALL: /logout
func (s *service) HandleLogout(c fiber.Ctx) error {
	s.SessionDestroy(c)
	s.InvalidateCookies(c, "Auth", "JWT")

	redirectURL := "/signin"
	if logtoEnabled(s.Env) {
		redirectURL = "/logto/signout"
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", redirectURL)
		return c.SendStatus(fiber.StatusTemporaryRedirect)
	}

	return s.Redirect(c, redirectURL)
}

// HandleResetPasswordPage renders the reset password request page.
// GET: /resetpassword
func (s *service) HandleResetPasswordPage(c fiber.Ctx) error {
	vc := view.NewCtx(c)

	msg := s.SessionRead(c, "msg", "")
	if msg == "" {
		msg = c.Query("msg", "")
	}

	return view.Render(c, view.ResetPasswordPage(vc, "", msg, "", nil))
}

// HandleResetPassword sends a reset password link to the provided email.
// as query param, url param or body param
// POST: /resetpassword
func (s *service) HandleResetPassword(c fiber.Ctx) error {
	vc := view.NewCtx(c)
	email, err := resetPasswordEmailFromRequest(c)
	if err != nil {
		errView := errors.New("email can't be blank")
		return view.Render(c, view.ResetPasswordPage(vc, email, "", "", errView))
	}

	user, err := gorm.G[models.User](s.DB.GormDB()).Select("id", "name").Where("email = ?", email).Take(c.Context())
	if err != nil {
		errView := errors.New("user not found with that email")
		return view.Render(c, view.ResetPasswordPage(vc, email, "", "", errView))
	}

	token, err := newResetPasswordToken()
	if err != nil {
		return s.Redirect(c, "/resetpassword")
	}

	user.ResetToken = token
	user.ResetTokenExpiresAt = time.Now().Add(time.Hour * 1)
	rowsAffected, err := gorm.G[models.User](s.DB.GormDB()).
		Select("reset_token", "reset_token_expires_at").
		Where("id = ?", user.ID).
		Updates(c.Context(), user)
	if err != nil || rowsAffected != 1 {
		errView := errors.New("failed to create reset password token")
		return view.Render(c, view.ResetPasswordPage(vc, email, "", "", errView))
	}

	s.sendResetPasswordEmailAsync(c.Context(), email, token)

	return view.Render(c, view.ResetPasswordPage(vc, email, "", "check your email for a reset password link", nil))
}

// HandleResetPasswordChange renders the password change page for a valid token.
// GET: /resetpassword/change/:token
func (s *service) HandleResetPasswordChange(c fiber.Ctx) error {
	token := c.Params("token", "")
	if token == "" {
		return s.Redirect(c, "/resetpassword?msg=token can't be blank")
	}

	email, err := s.resetPasswordVerifyToken(c.Context(), token)
	if err != nil {
		return s.Redirect(c, "/resetpassword?msg=invalid email or token")
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.ResetPasswordChangePage(email, token, nil, vc))
}

// HandleResetPasswordUpdate updates the user password in the DB
// POST: /resetpassword/change
func (s *service) HandleResetPasswordUpdate(c fiber.Ctx) error {
	email, err := resetPasswordEmailFromRequest(c)
	if err != nil {
		return s.Redirect(c, "/resetpassword?msg=something went wrong with the email, try again!")
	}

	token := c.FormValue("token", "")
	if token == "" {
		return s.Redirect(c, "/resetpassword?msg=something went wrong with the token, try again!")
	}

	_, err = s.resetPasswordVerifyToken(c.Context(), token)
	if err != nil {
		return s.Redirect(c, "/resetpassword?msg=seems like your token has expired, try again!")
	}

	vc := view.NewCtx(c)
	password := c.FormValue("password", "")
	if password == "" {
		err := errors.New("password can't be blank")
		return view.Render(c, view.ResetPasswordChangePage(email, token, err, vc))
	}

	confirm := c.FormValue("confirm", "")
	if confirm == "" || password != confirm {
		err := errors.New("passwords don't match")
		return view.Render(c, view.ResetPasswordChangePage(email, token, err, vc))
	}

	_, err = s.userUpdatePassword(c.Context(), email, password, token)
	if err != nil {
		return s.Redirect(c, "/resetpassword?msg=something went wrong, try again!")
	}

	return s.Redirect(c, "/signin")
}

// HandleValidate validates the current authentication session.
// POST: /api/auth/validate
func (s *service) HandleValidate(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

// HandleShowUser returns the current user's public fields as JSON.
// GET: /api/auth/protected
func (s *service) HandleShowUser(c fiber.Ctx) error {
	_, span := s.Trace(c.Context(), "auth/handlers:HandleShowUser")
	defer span.End()

	out := newShowUserOutput(s.CurrentUser(c))
	return c.JSON(out)
}

func (s *service) providerSigninRedirect(c fiber.Ctx, msg string) (redirectURL string, nextMsg string) {
	meta := selectAuthenticator(s).Metadata()
	if !meta.Enabled {
		return "", msg
	}

	providerError := c.Query(meta.ErrorQueryKey, "")
	loggedOut := c.Query(meta.LoggedOutQueryKey, "")
	skipAutoRedirect := s.SessionRead(c, meta.SkipRedirectSessionKey, "") == "1"
	maxErrorRedirects := meta.ErrorRedirectMax
	if maxErrorRedirects <= 0 {
		maxErrorRedirects = 3
	}
	if skipAutoRedirect {
		_ = s.SessionWrite(c, meta.SkipRedirectSessionKey, "")
		if meta.ErrorCountSessionKey != "" {
			_ = s.SessionWrite(c, meta.ErrorCountSessionKey, "0")
		}
	}

	errorCount := 0
	if meta.ErrorCountSessionKey != "" {
		raw := s.SessionRead(c, meta.ErrorCountSessionKey, "0")
		parsed, err := strconv.Atoi(strings.TrimSpace(raw))
		if err == nil && parsed > 0 {
			errorCount = parsed
		}
	}

	if providerError != "" {
		errorCount++
		if meta.ErrorCountSessionKey != "" {
			_ = s.SessionWrite(c, meta.ErrorCountSessionKey, strconv.Itoa(errorCount))
		}
		if errorCount < maxErrorRedirects {
			return meta.SigninPath, msg
		}
	} else if meta.ErrorCountSessionKey != "" {
		_ = s.SessionWrite(c, meta.ErrorCountSessionKey, "0")
	}

	if providerError == "" && loggedOut == "" && !skipAutoRedirect {
		return meta.SigninPath, msg
	}

	if msg != "" {
		return "", msg
	}

	switch {
	case providerError != "":
		return "", meta.ErrorMessage
	case loggedOut == "1" || skipAutoRedirect:
		return "", meta.SignedOutMessage
	default:
		return "", msg
	}
}
