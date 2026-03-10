package patient

import "testing"

func TestPatientUpsertInputToPatient(t *testing.T) {
	in := patientUpsertInput{
		Name:                "Patient",
		Email:               "patient@example.com",
		Phone:               "+999",
		Age:                 30,
		Ocupation:           "Engineer",
		FamilyHistory:       "none",
		MedicalBackground:   "none",
		Notes:               "notes",
		AddressLine1:        "L1",
		AddressLine2:        "L2",
		AddressCity:         "City",
		AddressState:        "State",
		AddressCountry:      "Country",
		AddressZipCode:      "2000",
		Whatsapp:            "wa",
		Telegram:            "tg",
		Messenger:           "msg",
		Instagram:           "ig",
		Facebook:            "fb",
		EnableNotifications: true,
		ViaEmail:            true,
		ViaWhatsapp:         true,
		ViaTelegram:         false,
		ViaMessenger:        true,
	}

	patient := in.toPatient("pat_1", "usr_1")
	if patient.ID != "pat_1" || patient.UserID != "usr_1" {
		t.Fatalf("unexpected identifiers mapping: %#v", patient)
	}
	if patient.Name != in.Name || patient.Email != in.Email || patient.Ocupation != in.Ocupation {
		t.Fatalf("unexpected core field mapping: %#v", patient)
	}
	if patient.Address.Line1 != in.AddressLine1 || patient.Address.Zip != in.AddressZipCode {
		t.Fatalf("unexpected address mapping: %#v", patient.Address)
	}
	if patient.SocialMedia.Telegram != in.Telegram || patient.SocialMedia.Facebook != in.Facebook {
		t.Fatalf("unexpected social mapping: %#v", patient.SocialMedia)
	}
	if !patient.NotificationFlags.EnableNotifications || !patient.NotificationFlags.ViaEmail || !patient.NotificationFlags.ViaMessenger {
		t.Fatalf("unexpected notification flags mapping: %#v", patient.NotificationFlags)
	}
}
