package clinic

import "testing"

func TestClinicUpsertInputToClinic(t *testing.T) {
	in := clinicUpsertInput{
		Name:           "Clinic",
		Email:          "clinic@example.com",
		Phone:          "+123",
		AddressLine1:   "L1",
		AddressLine2:   "L2",
		AddressCity:    "City",
		AddressState:   "State",
		AddressCountry: "Country",
		AddressZipCode: "1000",
		Whatsapp:       "wa",
		Telegram:       "tg",
		Messenger:      "msg",
		Instagram:      "ig",
		Facebook:       "fb",
	}

	clinic := in.toClinic(1, "cln_1", 2, 250)
	if clinic.ID != 1 || clinic.UID != "cln_1" || clinic.UserID != 2 || clinic.Price != 250 {
		t.Fatalf("unexpected identifiers mapping: %#v", clinic)
	}
	if clinic.Name != in.Name || clinic.Email != in.Email || clinic.Phone != in.Phone {
		t.Fatalf("unexpected core field mapping: %#v", clinic)
	}
	if clinic.Address.Line1 != in.AddressLine1 || clinic.Address.Zip != in.AddressZipCode {
		t.Fatalf("unexpected address mapping: %#v", clinic.Address)
	}
	if clinic.SocialMedia.Whatsapp != in.Whatsapp || clinic.SocialMedia.Facebook != in.Facebook {
		t.Fatalf("unexpected social media mapping: %#v", clinic.SocialMedia)
	}
}
