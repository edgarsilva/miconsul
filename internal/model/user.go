package model

import (
	"miconsul/internal/lib/xid"
	"strings"
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

// --model:User
type User struct {
	ConfirmEmailExpiresAt time.Time
	ResetTokenExpiresAt   time.Time
	ID                    string `gorm:"primarykey;default:null;not null" form:"__blank__"`
	ExtID                 string
	ProfilePic            string
	Name                  string
	Email                 string `gorm:"uniqueIndex;default:null;not null"`
	Password              string `json:"-"`
	Theme                 string
	ResetToken            string
	ConfirmEmailToken     string
	Phone                 string
	Timezone              string
	LocaleLang            string
	Role                  UserRole `gorm:"index;default:null;not null;type:string" form:"__blank__"`
	ModelBase
	Clinics      []Clinic
	Patients     []Patient
	Appointments []Appointment
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = xid.New("user")
	return nil
}

func (u User) IsLoggedIn() bool {
	return u.ID != ""
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
	return u.ProfilePic
}

func (u User) AvatarPic() string {
	return u.ProfilePicPath()
}
