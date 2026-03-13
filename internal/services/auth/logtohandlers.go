package auth

import (
	"miconsul/internal/view"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
)

// HandleLogtoSignin redirects to Logto sign-in page
// GET: /logto/signin
func (s *service) HandleLogtoSignin(c fiber.Ctx) error {
	sess, err := s.Session(c)
	if err != nil {
		log.Error("failed to load session in logto signin:", err)
		return s.Redirect(c, "/signin?logto_error=session")
	}
	logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(s.Env))
	defer deferLogtoSessionSave("logto signin", saveSess)

	if logtoClient.IsAuthenticated() {
		return c.Redirect().Status(fiber.StatusTemporaryRedirect).To("/")
	}

	// The sign-in request is handled by Logto.
	// The user will be redirected to the RedirectURI after successful sign in.
	callbackURI, err := logtoRedirectURI(s.Env, "/logto/callback")
	if err != nil {
		log.Error("failed to compose logto callback redirect uri:", err)
		return s.Redirect(c, "/signin?logto_error=config")
	}

	signInUri, err := logtoClient.SignIn(callbackURI)
	if err != nil {
		log.Error("failed to build logto signin url:", err)
		return s.Redirect(c, "/signin?logto_error=signin")
	}

	// Redirect the user to the Logto sign-in page.
	return c.Redirect().Status(fiber.StatusTemporaryRedirect).To(signInUri)
}

// HandleLogtoCallback handles the Logto callback/webhook after login
// GET: /logto/callback
func (s *service) HandleLogtoCallback(c fiber.Ctx) error {
	sess, err := s.Session(c)
	if err != nil {
		log.Error("failed to load session in logto callback:", err)
		return s.Redirect(c, "/signin?logto_error=session")
	}
	logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(s.Env))
	defer deferLogtoSessionSave("logto callback", saveSess)

	req, err := adaptor.ConvertRequest(c, true)
	if err != nil {
		log.Error("failed to convert fiber request to http request with adaptor on logto callback:", err)
		return s.Redirect(c, "/signin?logto_error=request")
	}

	err = logtoClient.HandleSignInCallback(req)
	if err != nil {
		log.Error("failed to verify signin in logto callback handler:", err)
		return s.Redirect(c, "/signin?logto_error=callback")
	}

	// Identity-critical fields come from ID token claims after callback verification.
	claims, err := logtoClient.GetIdTokenClaims()
	if err != nil {
		log.Error("failed to get id token claims in logto callback handler:", err)
		return s.Redirect(c, "/signin?logto_error=id_token")
	}

	logtoUser, err := NewLogtoUser(claims)
	if err != nil {
		log.Error("failed to build user from id token claims:", err)
		return s.Redirect(c, "/signin?logto_error=claims")
	}

	customClaims, err := logtoCustomClaims(logtoClient, s.Env.LogtoResource)
	if err != nil {
		log.Warn("failed to decode custom access token claims, continuing with id token claims only:", err)
	} else {
		logtoUser.Identities = customClaims.Identities
	}

	err = s.saveLogtoUser(c.Context(), logtoUser)
	if err != nil {
		log.Error(err)
		return s.Redirect(c, "/signin?logto_error=user_sync")
	}

	// This example takes the user back to the home page.
	return s.Redirect(c, "/")
}

// HandleLogtoSignout redirects to Logto sign-out flow.
// GET: /logto/signout
func (s *service) HandleLogtoSignout(c fiber.Ctx) error {
	if err := s.SessionWrite(c, "logto_skip_redirect", "1"); err != nil {
		log.Warn("failed to mark login redirect skip before logto signout:", err)
	}

	sess, err := s.Session(c)
	if err != nil {
		log.Error("failed to load session in logto signout:", err)
		return s.Redirect(c, "/logto/signin")
	}
	logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(s.Env))
	defer deferLogtoSessionSave("logto signout", saveSess)

	// The sign-out request is handled by Logto.
	// The user will be redirected to the Post Sign-out Redirect URI on signed out.
	postSignOutURI, err := logtoRedirectURI(s.Env, "/logto/signin")
	if err != nil {
		log.Error("failed to compose post logout redirect uri:", err)
		return s.Redirect(c, "/logto/signin")
	}

	signOutUri, err := logtoClient.SignOut(postSignOutURI)
	if err != nil {
		log.Error("failed to build logto signout url:", err)
		return s.Redirect(c, "/logto/signin")
	}
	if signOutUri == "" {
		log.Warn("empty logto signout url, redirecting to /logto/signin")
		return s.Redirect(c, "/logto/signin")
	}

	return c.Redirect().Status(fiber.StatusTemporaryRedirect).To(signOutUri)
}

// HandleLogtoPage renders the Logto page with two links to sign in and sign out
// GET: /logto
func (s *service) HandleLogtoPage(c fiber.Ctx) error {
	sess, err := s.Session(c)
	if err != nil {
		log.Error("failed to load session in logto page:", err)
		return s.Redirect(c, "/logto/signout")
	}
	logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(s.Env))
	defer deferLogtoSessionSave("logto page", saveSess)

	notAuthenticated := !logtoClient.IsAuthenticated()
	if notAuthenticated {
		vc, _ := view.NewCtx(c)
		return view.Render(c, view.LogtoPage(
			vc,
			"You are not logged in to this website. :(",
			"{}",
			"{}",
		))
	}

	authState := "You are logged in to this website! :)"
	idTokenClaimsJSON := logtoIDTokenClaimsJSON(logtoClient)
	customClaimsJSON := logtoCustomClaimsJSON(logtoClient, s.Env.LogtoResource)

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.LogtoPage(vc, authState, idTokenClaimsJSON, customClaimsJSON))
}

func deferLogtoSessionSave(route string, saveSess func() error) {
	if err := saveSess(); err != nil {
		log.Error("failed to save session in "+route+":", err)
	}
}
