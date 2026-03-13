package clinic

import "miconsul/internal/model"

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

func (in clinicUpsertInput) toClinic(id, userID string, price int) model.Clinic {
	return model.Clinic{
		ID:     id,
		UserID: userID,
		Price:  price,
		Name:   in.Name,
		Email:  in.Email,
		Phone:  in.Phone,
		Address: model.Address{
			Line1:   in.AddressLine1,
			Line2:   in.AddressLine2,
			City:    in.AddressCity,
			State:   in.AddressState,
			Country: in.AddressCountry,
			Zip:     in.AddressZipCode,
		},
		SocialMedia: model.SocialMedia{
			Whatsapp:  in.Whatsapp,
			Telegram:  in.Telegram,
			Messenger: in.Messenger,
			Instagram: in.Instagram,
			Facebook:  in.Facebook,
		},
	}
}
