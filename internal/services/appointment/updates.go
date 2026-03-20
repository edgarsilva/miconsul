package appointment

import (
	"time"

	"miconsul/internal/models"
)

type appointmentPatchUpdates struct {
	BookedAt     time.Time
	BookedYear   int
	BookedMonth  int
	BookedDay    int
	BookedHour   int
	BookedMinute int
	Price        int
	ClinicID     string
	PatientID    string
	Duration     int
}

type appointmentCompleteUpdates struct {
	Status       model.AppointmentStatus
	Observations string
	Conclusions  string
	Summary      string
	Notes        string
}

type appointmentCancelUpdates struct {
	Status     model.AppointmentStatus
	CanceledAt time.Time
}

type appointmentTokenUpdates struct {
	Status      model.AppointmentStatus
	ConfirmedAt time.Time
	CanceledAt  time.Time
	PendingAt   time.Time
}
