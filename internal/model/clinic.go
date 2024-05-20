package model

import (
	"errors"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Clinic struct {
	ExtID      string
	CoverPic   string
	ProfilePic string
	Name       string `gorm:"default:null;not null"`
	Email      string `form:"email"`
	Phone      string `form:"phone"`
	UserID     string `gorm:"index;default:null;not null"`
	Address
	SocialMedia
	ModelBase
	User User
}

func (c *Clinic) BeforeCreate(tx *gorm.DB) error {
	err := c.IsValid()
	if err != nil {
		return err
	}

	if len(c.FieldErrors()) > 0 {
		return errors.New("invalid data found in clinic record")
	}
	c.ID = xid.New("clnc")
	return nil
}

func (c *Clinic) IsValid() error {
	if len(c.Name) == 0 {
		c.SetFieldError("name", "Name can't be blank")
	}

	if len(c.fieldErrors) > 0 {
		return errors.New("can't save invalid data")
	}

	return nil
}
