package model

import (
	"miconsul/internal/lib/xid"
	"time"

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
	ID                    string `gorm:"primarykey;default:null;not null" form: "__blank"`
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
	Phone                 string
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

// func (u *User) Name() string {
// 	return u.FirstName + " " + u.LastName
// }
