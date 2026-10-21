package models

import (
	"fmt"
	"strings"
	"time"

	"miconsul/internal/lib/xid"

	"gorm.io/gorm"
)

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
	UserRoleGuest UserRole = "guest"
	UserRoleAnon  UserRole = "anon"
	UserRoleTest  UserRole = "test"
)

// User represents a DB users row in the application.
type User struct {
	ConfirmEmailExpiresAt time.Time `json:"-" form:"-"`
	ResetTokenExpiresAt   time.Time `json:"-" form:"-"`
	ID                    uint      `gorm:"primaryKey" form:"-"`
	UID                   string    `gorm:"uniqueIndex;default:null;not null" form:"-"`
	ExtID                 string
	ProfilePic            string
	Name                  string
	Email                 string `gorm:"uniqueIndex;default:null;not null"`
	Password              string `json:"-" form:"-"`
	Theme                 string
	ResetToken            string `json:"-" form:"-"`
	ConfirmEmailToken     string `json:"-" form:"-"`
	Phone                 string
	Timezone              string
	// LocaleLang            string
	Role UserRole `gorm:"index;default:null;not null;type:string" form:"-"`
	ModelBase
	Clinics      []Clinic
	Patients     []Patient
	Appointments []Appointment
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.New("user")
	return nil
}

func (u User) IsLoggedIn() bool {
	return u.UID != ""
}

func (u User) Initials() string {
	if u.Name == "" {
		return "PA"
	}

	parts := strings.Split(u.Name, " ")
	a := string(parts[0][0])
	b := ""

	if len(parts) > 1 {
		b = string(parts[1][0])
	}

	return a + b
}

func (u User) ProfilePicPath() string {
	return fmt.Sprintf("%s?v=%d", u.ProfilePic, time.Now().UnixNano())
}

func (u User) AvatarPic() string {
	return u.ProfilePicPath()
}

// PublicUser is the JSON view of a User for API responses.
type PublicUser struct {
	UID        string
	ExtID      string
	Name       string
	Email      string
	ProfilePic string
	Phone      string
	Timezone   string
	Theme      string
	Role       UserRole
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Public returns the user's public fields.
func (u User) Public() PublicUser {
	return PublicUser{
		UID:        u.UID,
		ExtID:      u.ExtID,
		Name:       u.Name,
		Email:      u.Email,
		ProfilePic: u.ProfilePic,
		Phone:      u.Phone,
		Timezone:   u.Timezone,
		Theme:      u.Theme,
		Role:       u.Role,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}
