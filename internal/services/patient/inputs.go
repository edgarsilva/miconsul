package patient

import "miconsul/internal/models"

type patientUpsertInput struct {
	Name              string `form:"name"`
	Email             string `form:"email"`
	Phone             string `form:"phone"`
	Age               int    `form:"age"`
	Ocupation         string `form:"ocupation"`
	FamilyHistory     string `form:"familyHistory"`
	MedicalBackground string `form:"medicalBackground"`
	Notes             string `form:"notes"`

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

	EnableNotifications bool `form:"enableNotifications"`
	ViaEmail            bool `form:"viaEmail"`
	ViaWhatsapp         bool `form:"viaWhatsapp"`
	ViaTelegram         bool `form:"viaTelegram"`
	ViaMessenger        bool `form:"viaMessenger"`
}

func (in patientUpsertInput) toPatient(id, userID string) models.Patient {
	return models.Patient{
		ID:                id,
		UserID:            userID,
		Name:              in.Name,
		Email:             in.Email,
		Phone:             in.Phone,
		Age:               in.Age,
		Ocupation:         in.Ocupation,
		FamilyHistory:     in.FamilyHistory,
		MedicalBackground: in.MedicalBackground,
		Notes:             in.Notes,

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
		NotificationFlags: models.NotificationFlags{
			EnableNotifications: in.EnableNotifications,
			ViaEmail:            in.ViaEmail,
			ViaWhatsapp:         in.ViaWhatsapp,
			ViaTelegram:         in.ViaTelegram,
			ViaMessenger:        in.ViaMessenger,
		},
	}
}
