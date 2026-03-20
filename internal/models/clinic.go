package model

import (
	"errors"
	"miconsul/internal/lib/xid"
	"strconv"

	"gorm.io/gorm"
)

type Clinic struct {
	ID         string `gorm:"primarykey;default:null;not null" form:"_"`
	ExtID      string `gorm:"index;default:null;" form:"_"`
	UserID     string `gorm:"index;default:null;not null"`
	CoverPic   string
	ProfilePic string         `form:"profilePic"`
	Name       string         `gorm:"default:null;not null" form:"name"`
	Email      string         `form:"email"`
	Phone      string         `form:"phone"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	Address
	SocialMedia
	ModelBase
	User     User
	Favorite bool `form:"favorite"`
	Price    int  `form:"_"`
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

func (c Clinic) AvatarPic() string {
	return c.ProfilePic
}

func (c Clinic) Initials() string {
	if len(c.Name) < 2 {
		return "CL"
	}

	return string([]rune(c.Name)[0:2])
}

func (c *Clinic) PriceInputValue() string {
	v := strconv.FormatFloat(float64(c.Price/100), 'f', 1, 32)

	return v
}
