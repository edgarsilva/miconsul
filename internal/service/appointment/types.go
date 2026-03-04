package appointment

import "miconsul/internal/model"

type appointmentUpsertInput struct {
	ClinicID  string `form:"clinicId"`
	PatientID string `form:"patientId"`
	Duration  int    `form:"duration"`
}

type appointmentCompleteInput struct {
	Status       model.AppointmentStatus `form:"status"`
	Observations string                  `form:"observations"`
	Conclusions  string                  `form:"conclusions"`
	Summary      string                  `form:"summary"`
	Notes        string                  `form:"notes"`
}
