package middleware

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/edgarsilva/miconsul/internal/database"
	"github.com/edgarsilva/miconsul/internal/model"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type MWService interface {
	DBClient() *database.Database
}

// Authenticate an user based on Req Ctx Cookie 'Auth'
// cookies
func Authenticate(DB *database.Database, c *fiber.Ctx) (model.User, error) {
	uid := c.Cookies("Auth", "")
	if uid == "" {
		uid = strings.TrimPrefix(c.Get("Authorization", ""), "Bearer ")
	}

	// user, err := authenticateWithUID(DB, uid)
	user, err := authenticateWithJWT(DB, uid)
	if err == nil {
		return user, nil
	}

	return user, errors.New("failed to authenticate user")
}

func authenticateWithUID(DB *database.Database, uid string) (model.User, error) {
	user := model.User{}
	if uid == "" {
		return user, errors.New("user ID is blank")
	}

	result := DB.Where("id = ?", uid).Take(&user)
	if result.Error != nil {
		return user, errors.New("user NOT FOUND with ID in Auth cookie")
	}

	return user, nil
}

func authenticateWithJWT(DB *database.Database, tokenStr string) (model.User, error) {
	user := model.User{}
	if tokenStr == "" {
		return user, errors.New("JWT token is blank")
	}

	uid, err := JWTValidateToken(tokenStr)
	if err != nil {
		return user, errors.New("failed to validase JWT token")
	}

	result := DB.Where("id = ?", uid).Take(&user)
	if result.Error != nil {
		return user, errors.New("user NOT FOUND with UID in JWT token")
	}

	return user, nil
}

// JWTCreateToken returns a JWT token string for the sub and uid, optionally error
func JWTCreateToken(sub, uid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": sub,
		"uid": uid,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// JWTValidateToken returns the uid string from the JWT claims if valid, and an
// error if not valid or able to parse the token
func JWTValidateToken(tokenStr string) (uid string, err error) {
	tokenJWT, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the algorithm is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", errors.New("JWT token can't be parsed")
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := tokenJWT.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("JWT token claims can't be parsed")
	}

	uid, ok = claims["uid"].(string)
	if !ok || uid == "" {
		return "", errors.New("uid not found in token claims")
	}

	return uid, nil
}

func MustAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := Authenticate(s.DBClient(), c)
		if err != nil {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				if c.Get("HX-Request") != "true" {
					return c.Redirect("/login")
				}
				c.Set("HX-Redirect", "/login")
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)
		return c.Next()
	}
}

func MustBeAdmin(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := Authenticate(s.DBClient(), c)
		if err != nil || cu.Role != model.UserRoleAdmin {
			switch c.Accepts("*/*", "text/html", "text/plain", "application/json") {
			case "*/*", "text/html":
				c.Set("HX-Redirect", "/login")
				return c.Redirect("/login")
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				c.Set("HX-Redirect", "/login")
				return c.Redirect("/login")
			}
		}

		return c.Next()
	}
}

func MaybeAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, _ := Authenticate(s.DBClient(), c)
		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}
