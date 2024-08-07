package model

import (
	"errors"
	"miconsul/internal/lib/xid"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"gorm.io/gorm"
)

// model: FeedEvent
type Patient struct {
	Address
	SocialMedia
	ModelBase
	ID                string `gorm:"primarykey;default:null;not null" form:"_"`
	ExtID             string
	Email             string         `form:"email"`
	Phone             string         `form:"phone"`
	Ocupation         string         `form:"ocupation"`
	UserID            string         `gorm:"index;default:null;not null"`
	Name              string         `gorm:"default:null;not null" form:"name"`
	ProfilePic        string         `form:"profilePic"`
	FamilyHistory     string         `form:"familyHistory"`
	MedicalBackground string         `form:"medicalBackground"`
	Notes             string         `form:"notes"`
	DeletedAt         gorm.DeletedAt `form:"_"`
	User              User
	Appointments      []Appointment
	Age               int `form:"age"`
	NotificationFlags
}

func (p *Patient) BeforeCreate(tx *gorm.DB) error {
	err := p.IsValid()
	if err != nil {
		return err
	}

	p.ID = xid.New("ptnt")
	return nil
}

func (p *Patient) IsValid() error {
	if len(p.Name) == 0 {
		p.SetFieldError("name", "Name can't be blank")
	}

	if p.Age <= 0 {
		p.SetFieldError("age", "Age can't be blank")
	}

	if len(p.Phone) == 0 {
		p.SetFieldError("phone", "Phone can't be blank")
	}

	if len(p.FieldErrors()) > 0 {
		return errors.New("can't save invalid data")
	}

	return nil
}

func (p *Patient) Sanitize() {
	p.Email = bluemonday.UGCPolicy().Sanitize(p.Email)
	p.Phone = bluemonday.UGCPolicy().Sanitize(p.Phone)
	p.Ocupation = bluemonday.UGCPolicy().Sanitize(p.Ocupation)
	p.Line1 = bluemonday.UGCPolicy().Sanitize(p.Line1)
	p.Line2 = bluemonday.UGCPolicy().Sanitize(p.Line2)
	p.City = bluemonday.UGCPolicy().Sanitize(p.City)
	p.State = bluemonday.UGCPolicy().Sanitize(p.State)
	p.Zip = bluemonday.UGCPolicy().Sanitize(p.Zip)
	p.Country = bluemonday.UGCPolicy().Sanitize(p.Country)
	p.FamilyHistory = bluemonday.UGCPolicy().Sanitize(p.FamilyHistory)
	p.MedicalBackground = bluemonday.UGCPolicy().Sanitize(p.MedicalBackground)
	p.Notes = bluemonday.UGCPolicy().Sanitize(p.Notes)
	p.Whatsapp = bluemonday.UGCPolicy().Sanitize(p.Whatsapp)
	p.Telegram = bluemonday.UGCPolicy().Sanitize(p.Telegram)
	p.Messenger = bluemonday.UGCPolicy().Sanitize(p.Messenger)
	p.Facebook = bluemonday.UGCPolicy().Sanitize(p.Facebook)
}

func (p Patient) Initials() string {
	if p.Name == "" {
		return "PA"
	}

	parts := strings.Split(p.Name, " ")
	a := string(parts[0][0])
	b := ""

	if len(parts) > 1 {
		b = string(parts[1][0])
	}

	return a + b
}

func (p *Patient) ProfilePicPath() string {
	return p.ProfilePic
}

func (p Patient) AvatarPic() string {
	return p.ProfilePicPath()
}
