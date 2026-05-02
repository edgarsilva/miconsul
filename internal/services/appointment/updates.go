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
	ClinicID     uint
	PatientID    uint
	Duration     int
}

type appointmentCompleteUpdates struct {
	Status       models.AppointmentStatus
	Observations string
	Conclusions  string
	Summary      string
	Notes        string
}

type appointmentCancelUpdates struct {
	Status     models.AppointmentStatus
	CanceledAt time.Time
}

type appointmentTokenUpdates struct {
	Status      models.AppointmentStatus
	ConfirmedAt time.Time
	CanceledAt  time.Time
	PendingAt   time.Time
}
