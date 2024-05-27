package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
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

type User struct {
	ConfirmEmailExpiresAt time.Time
	ResetTokenExpiresAt   time.Time
	ExtID                 string
	ProfilePic            string
	Name                  string
	Email                 string   `gorm:"uniqueIndex;default:null;not null"`
	Role                  UserRole `gorm:"index;default:null;not null;type:string"`
	Password              string   `json:"-"`
	Theme                 string
	ResetToken            string
	ConfirmEmailToken     string
	ModelBase
	Clinics      []Clinic
	Patients     []Patient
	Appointments []Appointment

	Todos []Todo
	// Articles []Article
	// Comments []Comment
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = xid.New("user")
	return nil
}

func (u User) IsLoggedIn() bool {
	return u.ID != ""
}
