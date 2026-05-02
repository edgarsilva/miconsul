package clinic

import "miconsul/internal/models"

type clinicUpsertInput struct {
	Name  string `form:"name"`
	Email string `form:"email"`
	Phone string `form:"phone"`

	AddressLine1   string `form:"addressLine1"`
	AddressLine2   string `form:"addressLine2"`
	AddressCity    string `form:"addressCity"`
	AddressState   string `form:"addressState"`
	AddressCountry string `form:"addressCountry"`
	AddressZipCode string `form:"addressZipCode"`

	Whatsapp  string `form:"whatsapp"`
	Telegram  string `form:"telegram"`
	Messenger string `form:"messenger"`
	Instagram string `form:"instagram"`
	Facebook  string `form:"facebook"`
}

func (in clinicUpsertInput) toClinic(id uint, uid string, userID uint, price int) models.Clinic {
	return models.Clinic{
		ID:     id,
		UID:    uid,
		UserID: userID,
		Price:  price,
		Name:   in.Name,
		Email:  in.Email,
		Phone:  in.Phone,
		Address: models.Address{
			Line1:   in.AddressLine1,
			Line2:   in.AddressLine2,
			City:    in.AddressCity,
			State:   in.AddressState,
			Country: in.AddressCountry,
			Zip:     in.AddressZipCode,
		},
		SocialMedia: models.SocialMedia{
			Whatsapp:  in.Whatsapp,
			Telegram:  in.Telegram,
			Messenger: in.Messenger,
			Instagram: in.Instagram,
			Facebook:  in.Facebook,
		},
	}
}
