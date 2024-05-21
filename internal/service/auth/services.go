package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/mailer"
	"github.com/edgarsilva/go-scaffold/internal/model"
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
	if err := s.signupIsEmailValid(email); err != nil {
		return err
	}

	// if err := s.signupIsPasswordValid(password); err != nil {
	// 	return err
	// }

	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return errors.New("failed to save email or password, try again")
	}

	token := randToken()

	_, err = s.userCreate(email, string(pwd), token)
	if err != nil {
		return errors.New("failed to save email or password, try again")
	}

	go mailer.ConfirmEmail(email, token)

	return nil
}

// IsEmailValidForSignup returns nil if valid, otherwise returns an error
func (s service) signupIsEmailValid(email string) error {
	validEmail := govalidator.IsEmail(email)
	if !validEmail {
		return errors.New("email address is invalid")
	}

	user := model.User{Email: email}
	if result := s.DB.Where(user, "Email").Take(&user); result.RowsAffected != 0 {
		return errors.New("email already exists, try login instead")
	}

	return nil
}

// signupIsPasswordValid returns nil if valid, otherwise returns an error
func (s service) signupIsPasswordValid(pwd string) error {
	if len(pwd) < 8 {
		return errors.New("password is too short")
	}

	if strings.ContainsAny(pwd, "1234567890") {
		return errors.New("password must contain at least 1 digit (numbers from 0 to 9)")
	}

	if strings.ContainsAny(pwd, "!@#$%^&*()") {
		return errors.New("password must contain at least 1 special character e.g. !@#$%^&*")
	}

	return nil
}

// userCreate creates a new row in the users table
func (s service) userCreate(email, password, token string) (model.User, error) {
	user := model.User{
		Email:                 email,
		Password:              password,
		ConfirmEmailToken:     token,
		ConfirmEmailExpiresAt: time.Now().Add(time.Hour * 1),
		Role:                  model.UserRoleUser,
	}

	result := s.DB.Create(&user) // pass pointer of data to Create
	if result.Error != nil {
		err := errors.New("faild to save email or password, try again")
		return model.User{}, err
	}

	return user, nil
}

// userFetch returns a User by email
func (s service) userFetch(email string) (model.User, error) {
	user := model.User{Email: email}
	s.DB.Where(user, "Email").Take(&user)
	if user.ID == "" {
		return user, errors.New("user not found")
	}

	return user, nil
}

// userUpdatePassword updates a user password
func (s service) userUpdatePassword(email, password, token string) (model.User, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return model.User{}, errors.New("failed to update password")
	}

	user := model.User{
		Email:    email,
		Password: string(pwd),
	}

	result := s.DB.
		Model(&user).
		Select("Password", "ResetToken", "ResetTokenExpiresAt").
		Where("email = ? AND reset_token = ? AND reset_token_expires_at > ?", email, token, time.Now()).
		Limit(1).
		Updates(&user)
	if result.Error != nil || result.RowsAffected != 1 {
		return model.User{}, errors.New("failed to update password")
	}

	return user, nil
}

func (s service) userUpdateConfirmToken(email, token string) error {
	user := model.User{
		ConfirmEmailToken:     token,
		ConfirmEmailExpiresAt: time.Now().Add(time.Hour * 24),
	}
	result := s.DB.
		Model(&user).
		Select("ConfirmEmailToken", "ConfirmEmailExpiresAt").
		Where("email = ?", email).
		Limit(1).
		Updates(&user)
	if result.Error != nil || result.RowsAffected != 1 {
		return errors.New("failed to update confirm token")
	}

	return nil
}

func (s service) userPendingConfirmation(email string) error {
	user := model.User{}
	result := s.DB.
		Select("id, email, confirm_email_token").
		Where("email = ? AND confirm_email_token IS NOT null AND confirm_email_token != ''", email).
		Take(&user)
	if result.RowsAffected != 0 || user.ID != "" { // If a row/record exists it means confirmation is pending and we should re-send
		return errors.New("user pending confirmation")
	}

	return nil
}

// Authenticate an user based on Req Ctx Cookie 'Auth'
// cookies
func Authenticate(DB *database.Database, c *fiber.Ctx) (model.User, error) {
	uid := c.Cookies("Auth", "")
	if uid == "" {
		uid = strings.TrimPrefix(c.Get("Authorization", ""), "Bearer ")
	}

	user, err := authenticateWithUID(DB, uid)
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

func (s service) resetPasswordVerifyToken(token string) (email string, err error) {
	user := model.User{}
	result := s.DB.
		Select("id, email").
		Where("reset_token != '' AND reset_token IS NOT null AND reset_token = ? AND reset_token_expires_at > ?", token, time.Now()).
		Take(&user)
	if result.Error != nil {
		return "", errors.New("password reset token not found or expired")
	}

	return user.Email, nil
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
