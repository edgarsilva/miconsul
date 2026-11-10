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
		AddressCity:         "City",
		AddressState:        "State",
		AddressCountry:      "Country",
		AddressZipCode:      "2000",
		EnableNotifications: true,
		ViaEmail:            true,
		ViaWhatsapp:         true,
		ViaTelegram:         false,
		ViaMessenger:        true,
	}

	patient := in.toPatient(1, "pat_1", 2)
	if patient.ID != 1 || patient.UID != "pat_1" || patient.UserID != 2 {
		t.Fatalf("unexpected identifiers mapping: %#v", patient)
	}
	if patient.Name != in.Name || patient.Email != in.Email || patient.Ocupation != in.Ocupation {
		t.Fatalf("unexpected core field mapping: %#v", patient)
	}
	if patient.Address.Line1 != in.AddressLine1 || patient.Address.Zip != in.AddressZipCode {
		t.Fatalf("unexpected address mapping: %#v", patient.Address)
	}
	if !patient.NotificationFlags.EnableNotifications || !patient.NotificationFlags.ViaEmail || !patient.NotificationFlags.ViaMessenger {
		t.Fatalf("unexpected notification flags mapping: %#v", patient.NotificationFlags)
	}
}
