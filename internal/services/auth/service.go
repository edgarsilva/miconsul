// Package auth provides authentication services
// e.g. signup, login, logout, etc.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"miconsul/internal/lib/xid"
	"miconsul/internal/mailer"
	"miconsul/internal/model"
	"miconsul/internal/server"

	"github.com/asaskevich/govalidator"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type service struct {
	*server.Server
	tracer trace.Tracer
}

var (
	errAuthInvalidCredentials  = errors.New("invalid credentials")
	errAuthPendingConfirmation = errors.New("email pending confirmation")
	errAuthSessionCreate       = errors.New("failed to create session")
)

func New(s *server.Server) (*service, error) {
	if s == nil {
		return nil, errors.New("auth service requires a non-nil server")
	}
	if s.Env == nil {
		return nil, errors.New("auth service requires a non-nil server env")
	}

	tracerName := s.Env.OTelTracerAuth
	if tracerName == "" {
		tracerName = s.Env.AppName + ".auth"
	}

	return &service{
		Server: s,
		tracer: otel.Tracer(tracerName),
	}, nil
}

func (s *service) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	return s.tracer.Start(ctx, spanName, opts...)
}

// Signup creates a new user record if req.body Email & Password are valid
func (s *service) signup(ctx context.Context, email string, password string) error {
	if err := s.signupIsEmailValid(ctx, email); err != nil {
		return err
	}

	if err := s.signupIsPasswordValid(password); err != nil {
		return err
	}

	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return errors.New("failed to save email or password, try again")
	}

	token := newConfirmEmailToken()
	_, err = s.userCreate(ctx, email, string(pwd), token)
	if err != nil {
		return errors.New("failed to save email or password, try again")
	}

	go mailer.ConfirmEmail(email, token)

	return nil
}

// IsEmailValidForSignup returns nil if valid, otherwise returns an error
func (s *service) signupIsEmailValid(ctx context.Context, email string) error {
	validEmail := govalidator.IsEmail(email)
	if !validEmail {
		return errors.New("email address is invalid")
	}

	user, err := gorm.G[model.User](s.DB.GormDB()).Where("email = ?", email).Take(ctx)
	if err == nil && user.ID != "" {
		return errors.New("email already exists, try login instead")
	}

	return nil
}

// signupIsPasswordValid returns nil if valid, otherwise returns an error
func (s *service) signupIsPasswordValid(pwd string) error {
	if len(pwd) < 8 {
		return errors.New("password is too short")
	}

	if !strings.ContainsAny(pwd, "1234567890") {
		return errors.New("password must contain at least 1 digit (numbers from 0 to 9)")
	}

	if !strings.ContainsAny(pwd, "!@#$%^&*()") {
		return errors.New("password must contain at least 1 special character e.g. !@#$%^&*")
	}

	return nil
}

func (s *service) userPendingConfirmation(ctx context.Context, email string) error {
	user, err := gorm.G[model.User](s.DB.GormDB()).
		Select("ID, Email, ConfirmEmailToken").
		Where("email = ? AND confirm_email_token IS NOT null AND confirm_email_token != ''", email).
		Take(ctx)
	if err == nil && user.ID != "" { // If a row/record exists it means confirmation is pending and we should re-send
		return errors.New("user pending confirmation")
	}

	return nil
}

// userCreate creates a new row in the users table
func (s *service) userCreate(ctx context.Context, email, password, token string) (model.User, error) {
	user := model.User{
		Email:                 email,
		Password:              password,
		ConfirmEmailToken:     token,
		ConfirmEmailExpiresAt: time.Now().Add(time.Hour * 1),
		Role:                  model.UserRoleUser,
	}

	err := gorm.G[model.User](s.DB.GormDB()).Create(ctx, &user)
	if err != nil {
		err := errors.New("faild to save email or password, try again")
		return model.User{}, err
	}

	return user, nil
}

// userFetch returns a User by email
func (s *service) userFetch(ctx context.Context, email string) (model.User, error) {
	ctx, span := s.Trace(ctx, "auth/services:userFetch")
	defer span.End()

	user, err := gorm.G[model.User](s.DB.GormDB()).Where("email = ?", email).Take(ctx)
	if err != nil {
		return model.User{}, errors.New("user not found")
	}

	return user, nil
}

func (s *service) authenticateCredentials(ctx context.Context, email, password string) (model.User, error) {
	user, err := s.userFetch(ctx, email)
	if err != nil {
		return model.User{}, errAuthInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return model.User{}, errAuthInvalidCredentials
	}

	if user.ConfirmEmailToken != "" {
		return model.User{}, errAuthPendingConfirmation
	}

	return user, nil
}

// userUpdatePassword updates a user password
func (s *service) userUpdatePassword(ctx context.Context, email, password, token string) (model.User, error) {
	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return model.User{}, errors.New("failed to update password")
	}

	user := model.User{
		Email:    email,
		Password: string(pwd),
	}

	rowsAffected, err := gorm.G[model.User](s.DB.GormDB()).
		Select("Password", "ResetToken", "ResetTokenExpiresAt").
		Where("email = ? AND reset_token = ? AND reset_token_expires_at > ?", email, token, time.Now()).
		Limit(1).
		Updates(ctx, user)
	if err != nil || rowsAffected != 1 {
		return model.User{}, errors.New("failed to update password")
	}

	return user, nil
}

func (s *service) userUpdateConfirmToken(ctx context.Context, email, token string) error {
	user := model.User{
		ConfirmEmailToken:     token,
		ConfirmEmailExpiresAt: time.Now().Add(time.Hour * 24),
	}
	rowsAffected, err := gorm.G[model.User](s.DB.GormDB()).
		Select("ConfirmEmailToken", "ConfirmEmailExpiresAt").
		Where("email = ?", email).
		Limit(1).
		Updates(ctx, user)
	if err != nil || rowsAffected != 1 {
		return errors.New("failed to update confirm token")
	}

	return nil
}

func (s *service) resetPasswordVerifyToken(ctx context.Context, token string) (email string, err error) {
	user, err := gorm.G[model.User](s.DB.GormDB()).
		Select("id, email").
		Where("reset_token != '' AND reset_token IS NOT null AND reset_token = ? AND reset_token_expires_at > ?", token, time.Now()).
		Take(ctx)
	if err != nil {
		return "", errors.New("password reset token not found or expired")
	}

	return user.Email, nil
}

func (s *service) saveLogtoUser(ctx context.Context, logtoUser LogtoUser) error {
	ctx, span := s.Trace(ctx, "auth/logto:saveLogtoUser")
	defer span.End()

	user, err := gorm.G[model.User](s.DB.GormDB()).Where("email = ?", logtoUser.Email).Take(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to load user from logto claims, GORM error: %w", err)
	}

	userExists := user.ID != ""
	extIDMatchesUID := user.ExtID == logtoUser.UID
	if userExists && extIDMatchesUID {
		return nil
	}

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
	if !userExists || user.Role == "" {
		user.Role = model.UserRoleUser
	}

	if result := s.DB.WithContext(ctx).Save(&user); result.Error != nil {
		return fmt.Errorf("failed to create or update user from logto claims, GORM error: %w", result.Error)
	}

	return nil
}
