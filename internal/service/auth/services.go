package auth

import (
	"errors"
	"fmt"
	"miconsul/internal/database"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"os"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/golang-jwt/jwt/v5"

	logto "github.com/logto-io/go/client"
	"golang.org/x/crypto/bcrypt"
)

type MWService interface {
	DBClient() *database.Database
	Session(*fiber.Ctx) *session.Session
	LogtoClient(c *fiber.Ctx) (client *logto.LogtoClient, save func())
	LogtoEnabled() bool
}

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

	if err := s.signupIsPasswordValid(password); err != nil {
		return err
	}

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
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// Authenticate an user based on Req Ctx Cookie 'Auth'
// cookies
func Authenticate(c *fiber.Ctx, mws MWService) (model.User, error) {
	if mws == nil {
		return model.User{}, errors.New("failed to authenticate user")
	}

	if mws.LogtoEnabled() {
		return logtoStrategy(c, mws)
	}

	return localStrategy(c, mws)
}

func logtoStrategy(c *fiber.Ctx, mws MWService) (model.User, error) {
	logtoClient, saveSess := mws.LogtoClient(c)
	defer saveSess()

	claims, err := logtoClient.GetIdTokenClaims()
	if err != nil {
		return model.User{}, err
	}

	db := mws.DBClient()
	user := model.User{ExtID: claims.Sub}
	fmt.Printf("CLAIMS ---> %#v\n\n", claims)
	result := db.Model(user).Take(&user)
	fmt.Printf("user ---> %#v", user)
	if result.Error != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

func localStrategy(c *fiber.Ctx, mws MWService) (model.User, error) {
	uid := c.Cookies("Auth", "")
	if uid == "" {
		uid = strings.TrimPrefix(c.Get("Authorization", ""), "Bearer ")
	}

	db := mws.DBClient()
	user, err := authenticateWithJWT(c, db, uid)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

func authenticateWithJWT(c *fiber.Ctx, db *database.Database, tokenStr string) (model.User, error) {
	user := model.User{}
	if tokenStr == "" {
		return user, errors.New("JWT token is blank")
	}
	secret := os.Getenv("JWT_SECRET")
	claims, err := DecodeJWTToken(secret, tokenStr)
	if err != nil {
		return user, errors.New("failed to validase JWT token")
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		uid = ""
	}

	result := db.Where("id = ?", uid).Take(&user)
	if result.Error != nil {
		return user, errors.New("user NOT FOUND with UID in JWT token")
	}

	RefreshAuthCookie(c, claims)

	return user, nil
}

// DecodeJWTToken returns the uid string from the JWT claims if valid, and an
// error if not valid or able to parse the token
func DecodeJWTToken(secret, tokenStr string) (claims jwt.MapClaims, err error) {
	tokenJWT, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the algorithm is what you expect:
		// if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		// 	return "", errors.New("JWT token can't be parsed")
		// }

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
	if err != nil {
		return jwt.MapClaims{}, err
	}

	claims, ok := tokenJWT.Claims.(jwt.MapClaims)
	if !ok {
		return jwt.MapClaims{}, errors.New("JWT token claims can't be parsed")
	}

	uid, ok := claims["uid"].(string)
	if !ok || uid == "" {
		return jwt.MapClaims{}, errors.New("uid not found in token claims")
	}

	return claims, nil
}

func RefreshAuthCookie(c *fiber.Ctx, claims jwt.MapClaims) {
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return
	}

	t1 := exp.Time
	t2 := time.Now()
	diff := t2.Sub(t1)
	if diff > time.Hour {
		return
	}

	email, err := claims.GetSubject()
	if err != nil {
		return
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		return
	}

	jwt, err := JWTCreateToken(email, uid)
	if err != nil {
		return
	}

	fmt.Println("New Token Created and saved in new cookie.")
	c.Cookie(newCookie("Auth", jwt, time.Hour*8))
}
