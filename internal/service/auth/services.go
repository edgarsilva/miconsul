package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/server"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}

// Signup creates a new user record if req.body Email & Password are valid
func (s service) signup(email string, password string) error {
	if email == "" || password == "" {
		return errors.New("incorrect email or password")
	}

	if err := s.signupIsEmailValid(email); err != nil {
		return err
	}

	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %q", err)
	}

	_, err = s.userCreate(email, string(pwd))
	if err != nil {
		return err
	}

	return nil
}

// IsEmailValidForSignup returns nil if valid, otherwise returns an error
func (s service) signupIsEmailValid(email string) error {
	validEmail := govalidator.IsEmail(email)
	if !validEmail {
		return errors.New("email address is invalid")
	}

	user := database.User{Email: email}
	if result := s.DB.Where(user, "Email").Take(&user); result.RowsAffected != 0 {
		return errors.New("email already exists")
	}

	return nil
}

// signupIsPasswordValid returns nil if valid, otherwise returns an error
func (s service) signupIsPasswordValid(pwd string) error {
	if len(pwd) < 8 {
		return errors.New("password is too short")
	}

	if strings.ContainsAny(pwd, "1234567890") {
		return errors.New("password must contain at least 1 digit e.g. one of '1234567890'")
	}

	if strings.ContainsAny(pwd, "!@#$%^&*()") {
		return errors.New("password must contain at least 1 special character e.g. one of !@#$%^&*()")
	}

	return nil
}

// userCreate creates a new row in the users table
func (s service) userCreate(email string, password string) (database.User, error) {
	user := database.User{
		Email:    email,
		Password: password,
		Role:     database.UserRoleUser,
	}
	result := s.DB.Create(&user) // pass pointer of data to Create
	if result.Error != nil {
		return database.User{}, fmt.Errorf("failed to write user row to the DB: %q", result.Error)
	}

	return user, nil
}

// userFetch returns a User by email
func (s service) userFetch(email string) (database.User, error) {
	user := database.User{Email: email}
	s.DB.Where(user, "Email").Take(&user)
	if user.ID == 0 {
		return user, errors.New("user not found")
	}

	return user, nil
}

// userUpdatePassword updates a user password
func (s service) userUpdatePassword(email, password, token string) (database.User, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return database.User{}, errors.New("failed to update password")
	}

	user := database.User{
		Email:    email,
		Password: string(pwd),
	}

	result := s.DB.
		Model(&user).
		Select("Password", "ResetToken", "ResetTokenExpiresAt").
		Where("email = ? AND reset_token = ? AND reset_token_expires_at > ?", email, token, time.Now()).
		Updates(&user)
	fmt.Println("RowsAffected ---->", result.RowsAffected, email, password, token, time.Now())
	if result.Error != nil || result.RowsAffected != 1 {
		return database.User{}, errors.New("failed to update password")
	}

	return user, nil
}

// CurrentUser returns currently logged-in(or not) user from fiber Req Ctx
// cookies
func Authenticate(DB *database.Database, c *fiber.Ctx) (database.User, error) {
	uid := c.Cookies("Auth", "")
	user, err := AuthenticateWithUID(DB, uid)
	if err == nil {
		return user, nil
	}

	tokenStr := c.Cookies("JWT", "")
	if tokenStr == "" {
		tokenStr = strings.TrimPrefix(c.Get("Authorization", ""), "Bearer ")
	}
	user, err = AuthenticateWithJWT(DB, tokenStr)
	if err == nil {
		return user, nil
	}

	return user, errors.New("invalid authentication, both methods are missing [Auth, JWT]")
}

func AuthenticateWithUID(DB *database.Database, uid string) (database.User, error) {
	user := database.User{}
	if uid == "" {
		return user, errors.New("user UID is blank")
	}

	result := DB.Where("uid = ?", uid).Take(&user)
	if result.Error != nil {
		return user, errors.New("user NOT FOUND with UID found in Auth cookie")
	}

	return user, nil
}

func AuthenticateWithJWT(DB *database.Database, tokenStr string) (database.User, error) {
	user := database.User{}
	if tokenStr == "" {
		return user, errors.New("JWT token is blank")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the algorithm is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return user, errors.New("JWT token can't be parsed")
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return user, errors.New("JWT token can't be parsed")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return user, errors.New("JWT token claims can't be parsed")
	}

	result := DB.Where("uid = ?", claims["uid"]).Take(&user)
	if result.Error != nil {
		return user, errors.New("user NOT FOUND with UID in JWT token")
	}

	return user, nil
}

func (s service) resetPasswordVerifyToken(email, token string) error {
	user := database.User{}
	result := s.DB.
		Select("id").
		Where("email = ? AND reset_token != '' AND reset_token IS NOT null AND reset_token = ? AND reset_token_expires_at > ?", email, token, time.Now()).
		Take(&user)
	if result.Error != nil {
		return errors.New("password reset token not found or expired")
	}

	return nil
}
