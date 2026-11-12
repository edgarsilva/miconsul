package appointment

import (
	"time"

	"miconsul/internal/models"
)

// appointmentColumner is implemented by every updates type: the columns it writes, zero values included.
type appointmentColumner interface {
	columns() []string
}

type appointmentUpdates struct {
	BookedAt      time.Time
	RescheduledAt time.Time
	BookedYear    int
	BookedMonth   int
	BookedDay     int
	BookedHour    int
	BookedMinute  int
	Price         int
	ClinicID      uint
	PatientID     uint
	Duration      int
}

type appointmentCompleteUpdates struct {
	Status       models.AppointmentStatus
	Observations string
	Conclusions  string
	Summary      string
	Notes        string
}

type appointmentStartUpdates struct {
	Status models.AppointmentStatus
}

type appointmentSessionUpdates struct {
	Observations string
	Conclusions  string
	Summary      string
	Notes        string
}

type appointmentCancelUpdates struct {
	Status     models.AppointmentStatus
	CanceledAt time.Time
}

type appointmentPauseUpdates struct {
	Status    models.AppointmentStatus
	PendingAt time.Time
}

func (u appointmentUpdates) columns() []string {
	columns := []string{
		"BookedAt", "BookedYear", "BookedMonth", "BookedDay", "BookedHour", "BookedMinute",
		"Price", "ClinicID", "PatientID", "Duration",
	}
	if !u.RescheduledAt.IsZero() {
		columns = append(columns, "RescheduledAt")
	}

	return columns
}

func (u appointmentCompleteUpdates) columns() []string {
	return []string{"Status", "Observations", "Conclusions", "Summary", "Notes"}
}

func (u appointmentSessionUpdates) columns() []string {
	return []string{"Observations", "Conclusions", "Summary", "Notes"}
}

func (u appointmentStartUpdates) columns() []string {
	return []string{"Status"}
}

func (u appointmentPauseUpdates) columns() []string {
	return []string{"Status", "PendingAt"}
}

func (u appointmentCancelUpdates) columns() []string {
	return []string{"Status", "CanceledAt"}
}
