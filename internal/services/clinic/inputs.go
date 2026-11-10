package clinic

import "miconsul/internal/models"

type clinicUpsertInput struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	Phone string `form:"phone"`

	AddressLine1   string `form:"addressLine1"`
	AddressCity    string `form:"addressCity"`
	AddressState   string `form:"addressState"`
	AddressCountry string `form:"addressCountry"`
	AddressZipCode string `form:"addressZipCode"`

	Whatsapp  string `form:"whatsapp"`
	Telegram  string `form:"telegram"`
	Instagram string `form:"instagram"`
	Facebook  string `form:"facebook"`
}

// clinicFormColumns are the columns the clinic form writes, zero values included.
var clinicFormColumns = []string{
	"Name", "Email", "Phone", "Price",
	"Line1", "City", "State", "Country", "Zip",
	"Whatsapp", "Telegram", "Instagram", "Facebook",
}

func (in clinicUpsertInput) toClinic(id uint, uid string, userID uint, price int) models.Clinic {
	return models.Clinic{
		ID:        id,
		UID:       uid,
		UserID:    userID,
		Price:     price,
		Name:      in.Name,
		Email:     in.Email,
		Phone:     in.Phone,
		Line1:     in.AddressLine1,
		City:      in.AddressCity,
		State:     in.AddressState,
		Country:   in.AddressCountry,
		Zip:       in.AddressZipCode,
		Whatsapp:  in.Whatsapp,
		Telegram:  in.Telegram,
		Instagram: in.Instagram,
		Facebook:  in.Facebook,
	}
}
