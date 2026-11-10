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
	AddressCity    string `form:"addressCity"`
	AddressState   string `form:"addressState"`
	AddressCountry string `form:"addressCountry"`
	AddressZipCode string `form:"addressZipCode"`

	EnableNotifications bool `form:"enableNotifications"`
	ViaEmail            bool `form:"viaEmail"`
	ViaWhatsapp         bool `form:"viaWhatsapp"`
	ViaTelegram         bool `form:"viaTelegram"`
	ViaMessenger        bool `form:"viaMessenger"`
}

// patientFormColumns are the columns the patient form writes, zero values included.
var patientFormColumns = []string{
	"Name", "Email", "Phone", "Age", "Ocupation", "FamilyHistory", "MedicalBackground", "Notes",
	"Line1", "City", "State", "Country", "Zip",
	"EnableNotifications", "ViaEmail", "ViaWhatsapp", "ViaTelegram", "ViaMessenger",
}

func (in patientUpsertInput) toPatient(id uint, uid string, userID uint) models.Patient {
	return models.Patient{
		ID:                id,
		UID:               uid,
		UserID:            userID,
		Name:              in.Name,
		Email:             in.Email,
		Phone:             in.Phone,
		Age:               in.Age,
		Ocupation:         in.Ocupation,
		FamilyHistory:     in.FamilyHistory,
		MedicalBackground: in.MedicalBackground,
		Notes:             in.Notes,

		Line1:               in.AddressLine1,
		City:                in.AddressCity,
		State:               in.AddressState,
		Country:             in.AddressCountry,
		Zip:                 in.AddressZipCode,
		EnableNotifications: in.EnableNotifications,
		ViaEmail:            in.ViaEmail,
		ViaWhatsapp:         in.ViaWhatsapp,
		ViaTelegram:         in.ViaTelegram,
		ViaMessenger:        in.ViaMessenger,
	}
}
