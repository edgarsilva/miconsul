// Package auth provides authentication services
// e.g. signup, login, logout, etc.
package auth

import (
	"context"
	"errors"
	"fmt"
	"miconsul/internal/database"
	"miconsul/internal/lib/xid"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"go.opentelemetry.io/otel/trace"

	"golang.org/x/crypto/bcrypt"
)

type MiddlewareService interface {
	Session(c *fiber.Ctx) *session.Session
	DBClient() *database.Database
	Trace(ctx context.Context, spanName string) (context.Context, trace.Span)
}

type AuthStrategy interface {
	Authenticate(c *fiber.Ctx) (model.User, error)
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
	if result := s.DB.Model(user).Where(user, "Email").Take(&user); result.RowsAffected != 0 {
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
func (s service) userFetch(ctx context.Context, email string) (model.User, error) {
	ctx, span := s.Tracer.Start(ctx, "auth/services:userFetch")
	defer span.End()

	user := model.User{Email: email}
	s.DB.WithContext(ctx).Model(&user).Where(user, "Email").Take(&user)
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
		Select("ID, Email, ConfirmEmailToken").
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

// Authenticate an user based on Req Ctx Cookie 'Auth'
// cookies
func Authenticate(c *fiber.Ctx, mws MiddlewareService) (model.User, error) {
	strategy := selectStrategy(c, mws)
	user, err := strategy.Authenticate(c)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func selectStrategy(c *fiber.Ctx, mws MiddlewareService) AuthStrategy {
	switch {
	case LogtoEnabled():
		return NewLogtoStrategy(c, mws)
	default:
		return NewLocalStrategy(c, mws)
	}
}

func (s *service) saveLogtoUser(ctx context.Context, logtoUser LogtoUser) error {
	ctx, span := s.Trace(ctx, "auth/logto:saveLogtoUser")
	defer span.End()

	user := model.User{Email: logtoUser.Email}
	result := s.DB.WithContext(ctx).Model(&user).Where(user, "Email").Take(&user)

	userExists := user.ID != ""
	extIDMatchesUID := user.ExtID == logtoUser.UID
	if result.RowsAffected == 1 && userExists && extIDMatchesUID {
		// user exists and has the same extID as the logtoUser, user is now logged in
		return nil
	}

	// user does not exist or has a different extID
	// since user has a different extID and logtoUser.UID
	// we need to create a new user or update the existing one ExtID
	if user.Password == "" {
		rndPwd, err := bcrypt.GenerateFromPassword([]byte(xid.New("rpwd")), 8)
		if err != nil {
			return errors.New("failed to generate password placeholder for user")
		}
		user.Password = string(rndPwd)
	}

	user.Name = logtoUser.Name
	user.ExtID = logtoUser.UID
	user.Email = logtoUser.Email
	user.ProfilePic = logtoUser.Picture
	if logtoUser.Picture == "" && logtoUser.Identities.Google.Details.Avatar != "" {
		user.ProfilePic = logtoUser.Identities.Google.Details.Avatar
	}
	user.Phone = logtoUser.PhoneNumber
	user.Role = model.UserRoleUser

	if result := s.DB.WithContext(ctx).Save(&user); result.Error != nil {
		return fmt.Errorf("failed to create or update user from logto claims, GORM error: %w", result.Error)
	}

	return nil
}
