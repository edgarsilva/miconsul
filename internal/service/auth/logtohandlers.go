package auth

import (
	"miconsul/internal/view"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	logto "github.com/logto-io/go/client"
	logtocore "github.com/logto-io/go/core"
)

// HandleLogtoSignin redirects to Logto sign-in page
func (s service) HandleLogtoSignin(c *fiber.Ctx) error {
	logtoClient, saveSess := LogtoClient(s.Session(c))
	defer saveSess()

	if logtoClient.IsAuthenticated() {
		return c.Redirect("/", fiber.StatusTemporaryRedirect)
	}

	// The sign-in request is handled by Logto.
	// The user will be redirected to the RedirectURI after successful sign in.
	signInUri, err := logtoClient.SignIn(redirectURI("/logto/callback"))
	if err != nil {
		return c.Redirect("/logto/signout")
	}

	// Redirect the user to the Logto sign-in page.
	return c.Redirect(signInUri, fiber.StatusTemporaryRedirect)
}

// HandleLogtoCallback handles the Logto callback/webhook after login
func (s *service) HandleLogtoCallback(c *fiber.Ctx) error {
	logtoClient, saveSess := LogtoClient(s.Session(c))
	defer saveSess()

	req, err := adaptor.ConvertRequest(c, true)
	if err != nil {
		log.Error("failed to convert fiber request to http request with adaptor on logto callback:", err)
		return c.Redirect("/logto/signout")
	}

	err = logtoClient.HandleSignInCallback(req)
	if err != nil {
		log.Error("failed to verify signin in logto callback handler:", err)
		return c.Redirect("/logto/signout")
	}

	// claims, err := logtoClient.GetIdTokenClaims()
	// if err != nil {
	// 	log.Error("failed to get IdTokenClaims in logto callback handler")
	// 	return c.Redirect("/logto/signout")
	// }

	logtoUser, err := logtoCustomJWTClaims(logtoClient)
	if err != nil {
		log.Error("failed to get CustomClaims from logto:", err)
		return c.Redirect("/logto/signout")
	}

	err = s.saveLogtoUser(c.UserContext(), logtoUser)
	if err != nil {
		log.Error(err)
		return c.Redirect("/logto/signout")
	}

	// This example takes the user back to the home page.
	return c.Redirect("/", http.StatusSeeOther)
}

func (s *service) HandleLogtoSignout(c *fiber.Ctx) error {
	logtoClient, saveSess := LogtoClient(s.Session(c))
	defer saveSess()

	// The sign-out request is handled by Logto.
	// The user will be redirected to the Post Sign-out Redirect URI on signed out.
	signOutUri, err := logtoClient.SignOut(redirectURI("/login"))
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	return c.Redirect(signOutUri, fiber.StatusTemporaryRedirect)
}

// HandleLogtoPage renders the Logto page with two links to sign in and sign out
func (s *service) HandleLogtoPage(c *fiber.Ctx) error {
	logtoClient, saveSess := LogtoClient(s.Session(c))
	defer saveSess()

	// Use Logto to control the content of the home page
	authState := "You are not logged in to this website. :("
	if logtoClient.IsAuthenticated() {
		authState = "You are logged in to this website! :)"
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.LogtoPage(vc, authState))
}

func logtoCustomJWTClaims(logtoClient *logto.LogtoClient) (LogtoUser, error) {
	accessToken, err := logtoClient.GetAccessToken("https://app.miconsul.xyz/api")
	if err != nil {
		return LogtoUser{}, err
	}

	logtoUser, err := logtoDecodeAccessToken(accessToken.Token)
	if err != nil {
		return LogtoUser{}, err
	}

	return logtoUser, nil
}

func logtoDecodeAccessToken(token string) (LogtoUser, error) {
	jwtObject, err := logtocore.ParseSignedJwt(token)
	if err != nil {
		return LogtoUser{}, err
	}

	var logtoUser LogtoUser
	err = jwtObject.UnsafeClaimsWithoutVerification(&logtoUser)
	if err != nil {
		return LogtoUser{}, err
	}

	return logtoUser, nil
}
