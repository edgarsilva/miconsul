package auth

import (
	"errors"
	"miconsul/internal/lib/xid"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"golang.org/x/crypto/bcrypt"

	logto "github.com/edgarsilva/logto-go-client/client"
	logtocore "github.com/edgarsilva/logto-go-client/core"
)

func (s *service) HandleLogtoPage(c *fiber.Ctx) error {
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

	if logtoClient.IsAuthenticated() {
		return c.Redirect("/", fiber.StatusTemporaryRedirect)
	}

	// The sign-in request is handled by Logto.
	// The user will be redirected to the Redirect URI on signed in.
	signInUri, err := logtoClient.SignIn(callbackURL())
	if err != nil {
		return c.Redirect("/logto/signout")
	}

	// Redirect the user to the Logto sign-in page.
	return c.Redirect(signInUri, fiber.StatusTemporaryRedirect)
}

func (s *service) HandleLogtoCallback(c *fiber.Ctx) error {
	logtoClient, saveSess := s.LogtoClient(c)
	defer saveSess()

	req, err := adaptor.ConvertRequest(c, true)
	if err != nil {
		log.Error("failed to convert fiber request to http request with adaptor on logto callback")
		return c.Redirect("/logto/signout")
	}

	err = logtoClient.HandleSignInCallback(req)
	if err != nil {
		log.Error("failed to verify signin in logto callback handler")
		return c.Redirect("/logto/signout")
	}

	claims, err := logtoClient.GetIdTokenClaims()
	if err != nil || claims.Email == "" {
		log.Error("failed to get IdTokenClaims in logto callback handler")
		return c.Redirect("/logto/signout")
	}

	err = s.logtoSaveUser(claims)
	if err != nil {
		log.Error("failed to save user from profile in logto callback handler")
		return c.Redirect("/logto/signout")
	}

	// This example takes the user back to the home page.
	return c.Redirect("/", http.StatusSeeOther)
}

func (s service) logtoSaveUser(claims logtocore.IdTokenClaims) error {
	db := s.DBClient()
	user := model.User{Email: claims.Email}
	db.Where("email = ?", claims.Email).Take(&user)

	if user.ID != "" && user.ExtID == claims.Sub {
		return nil
	}

	if user.Password == "" {
		rndPwd, err := bcrypt.GenerateFromPassword([]byte(xid.New("rpwd")), 10)
		if err != nil {
			return errors.New("failed to save email or password, try again")
		}
		user.Password = string(rndPwd)
	}

	user.Name = claims.Name
	user.ExtID = claims.Sub
	user.Email = claims.Email
	user.ProfilePic = claims.Picture
	user.Phone = claims.PhoneNumber
	user.Role = model.UserRoleUser

	result := s.DB.Save(&user)
	if result.Error != nil {
		return errors.New("failed to save new user from logto profile")
	}

	return nil
}

func (s *service) HandleLogtoSignout(c *fiber.Ctx) error {
	logtoClient, saveSess := s.LogtoClient(c)
	defer saveSess()

	// The sign-out request is handled by Logto.
	// The user will be redirected to the Post Sign-out Redirect URI on signed out.
	signOutUri, err := logtoClient.SignOut("http://localhost:3000/login")
	if err != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	return c.Redirect(signOutUri, fiber.StatusTemporaryRedirect)
}

func logtoCustomJWTClaims(logtoClient *logto.LogtoClient) (logtocore.IdTokenClaims, error) {
	accessToken, err := logtoClient.GetAccessToken("https://app.miconsul.xyz/api")
	if err != nil {
		return logtocore.IdTokenClaims{}, err
	}

	claims, err := logtoDecodeAccessToken(accessToken.Token)
	if err != nil {
		return logtocore.IdTokenClaims{}, err
	}

	return claims, nil
}

func logtoDecodeAccessToken(token string) (logtocore.IdTokenClaims, error) {
	jwtObject, err := logtocore.ParseSignedJwt(token)
	if err != nil {
		return logtocore.IdTokenClaims{}, err
	}

	var accessTokenClaims logtocore.IdTokenClaims
	claimsErr := jwtObject.UnsafeClaimsWithoutVerification(&accessTokenClaims)
	if claimsErr != nil {
		return logtocore.IdTokenClaims{}, claimsErr
	}

	return accessTokenClaims, nil
}

// callbackURL returns the full qualified callbackURL for the path passed
//
//	e.g.
//		url := callbackURL("/logto/callback")
//		-> http://localhost:3000/logto/callback
func callbackURL(path string) string {
	domain := os.Getenv("APP_DOMAIN")
	protocol := os.Getenv("APP_PROTOCOL")
	path = strings.TrimPrefix(path, "/")

	url := protocol + "://" + domain + path
	return url
}
