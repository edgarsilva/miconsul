package appointment

type appointmentUpsertInput struct {
	ClinicUID  string `form:"clinicId"`
	PatientUID string `form:"patientId"`
	Duration   int    `form:"duration"`
}

type appointmentCompleteInput struct {
	Observations string `form:"observations"`
	Conclusions  string `form:"conclusions"`
	Summary      string `form:"summary"`
	Notes        string `form:"notes"`
}
