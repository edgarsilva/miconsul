package auth

import (
	"errors"
	"fmt"
	"rtx-blog/internal/database"
	"rtx-blog/internal/server"

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

	if err := s.isEmailValidForSignup(email); err != nil {
		return err
	}

	pwd, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %q", err)
	}

	_, err = s.createUser(email, string(pwd))
	if err != nil {
		return err
	}

	return nil
}

// IsEmailValidForSignup returns nil if valid, otherwise returns an error
func (s service) isEmailValidForSignup(email string) error {
	user := database.User{Email: email}
	if result := s.DB.Where(user, "Email").Take(&user); result.RowsAffected != 0 {
		return errors.New("email already exists")
	}

	return nil
}

// createUser creates a new row in the users table
func (s service) createUser(email string, password string) (database.User, error) {
	user := database.User{
		Email:    email,
		Password: password,
		Role:     "user",
	}
	result := s.DB.Create(&user) // pass pointer of data to Create
	if result.Error != nil {
		return database.User{}, fmt.Errorf("failed to write user row to the DB: %q", result.Error)
	}

	return user, nil
}

// fetchUser returns a User by email
func (s service) fetchUser(email string) (database.User, error) {
	user := database.User{Email: email}
	s.DB.Where(user, "Email").Take(&user)
	if user.ID == 0 {
		return user, errors.New("user not found")
	}

	return user, nil
}
