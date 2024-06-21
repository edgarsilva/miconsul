package auth

import (
	"fmt"
	"miconsul/internal/view"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func (s *service) HandleLogtoPage(c *fiber.Ctx) error {
	// Init LogtoClient
	logtoClient, saveSess := s.LogtoClient(c)
	defer saveSess()

	// Use Logto to control the content of the home page
	authState := "You are not logged in to this website. :("

	if logtoClient.IsAuthenticated() {
		authState = "You are logged in to this website! :)"
	}

	vc, _ := view.NewCtx(c)
	return view.Render(c, view.LogtoPage(vc, authState))
}

func (s *service) HandleLogtoSignin(c *fiber.Ctx) error {
	logtoClient, saveSess := s.LogtoClient(c)
	defer saveSess()

	// The sign-in request is handled by Logto.
	// The user will be redirected to the Redirect URI on signed in.
	signInUri, err := logtoClient.SignIn("http://localhost:3000/logto/callback")
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// Redirect the user to the Logto sign-in page.
	return c.Redirect(signInUri, fiber.StatusTemporaryRedirect)
}

func (s *service) HandleLogtoCallback(c *fiber.Ctx) error {
	logtoClient, saveSess := s.LogtoClient(c)
	defer saveSess()

	req, _ := adaptor.ConvertRequest(c, true)

	err := logtoClient.HandleSignInCallback(req)
	if err != nil {
		fmt.Println("--------->", err)
		return nil
	}

	// Jump to the page specified by the developer.
	// This example takes the user back to the home page.
	return c.Redirect("/logto", http.StatusTemporaryRedirect)
}

func (s *service) HandleLogtoSignout(c *fiber.Ctx) error {
	logtoClient, saveSess := s.LogtoClient(c)
	defer saveSess()

	// The sign-out request is handled by Logto.
	// The user will be redirected to the Post Sign-out Redirect URI on signed out.
	signOutUri, err := logtoClient.SignOut("http://localhost:3000/logto")
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	return c.Redirect(signOutUri, fiber.StatusTemporaryRedirect)
}
