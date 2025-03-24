package auth

import (
	"context"
	"errors"
	"fmt"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"golang.org/x/crypto/bcrypt"

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

	err = s.logtoSaveUser(c.UserContext(), logtoUser)
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

func (s *service) logtoSaveUser(ctx context.Context, claims LogtoUser) error {
	ctx, span := s.Trace(ctx, "auth/logto:logtoSaveUser")
	defer span.End()

	user := model.User{Email: claims.Email}
	s.DB.WithContext(ctx).Model(&user).Where(user, "Email").Take(&user)

	if user.ID != "" && user.ExtID == claims.Sub {
		return nil
	}

	if user.Password == "" {
		rndPwd, err := bcrypt.GenerateFromPassword([]byte(xid.New("rpwd")), 10)
		if err != nil {
			return errors.New("failed to generate password placeholder for user")
		}
		user.Password = string(rndPwd)
	}

	user.Name = claims.Name
	user.ExtID = claims.Sub
	user.Email = claims.Email
	user.ProfilePic = claims.Picture
	if claims.Picture == "" && claims.Identities.Google.Details.Avatar != "" {
		user.ProfilePic = claims.Identities.Google.Details.Avatar
	}
	user.Phone = claims.PhoneNumber
	user.Role = model.UserRoleUser

	if result := s.DB.WithContext(ctx).Save(&user); result.Error != nil {
		return fmt.Errorf("failed to create or update user from logto claims, GORM error: %w", result.Error)
	}

	return nil
}

func logtoCustomJWTClaims(logtoClient *logto.LogtoClient) (LogtoUser, error) {
	accessToken, err := logtoClient.GetAccessToken("https://app.miconsul.xyz/api")
	if err != nil {
		return LogtoUser{}, err
	}

	claims, err := logtoDecodeAccessToken(accessToken.Token)
	if err != nil {
		return LogtoUser{}, err
	}

	return claims, nil
}

func logtoDecodeAccessToken(token string) (LogtoUser, error) {
	jwtObject, err := logtocore.ParseSignedJwt(token)
	if err != nil {
		return LogtoUser{}, err
	}

	var accessTokenClaims LogtoUser
	err = jwtObject.UnsafeClaimsWithoutVerification(&accessTokenClaims)
	if err != nil {
		return LogtoUser{}, err
	}

	return accessTokenClaims, nil
}
