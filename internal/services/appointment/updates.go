package appointment

import (
	"fmt"
	"time"

	"miconsul/internal/models"
)

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

// appointmentUpdateColumns returns the columns an updates value writes, zero values included.
func appointmentUpdateColumns(updates any) ([]string, error) {
	switch u := updates.(type) {
	case appointmentUpdates:
		columns := []string{
			"BookedAt", "BookedYear", "BookedMonth", "BookedDay", "BookedHour", "BookedMinute",
			"Price", "ClinicID", "PatientID", "Duration",
		}
		if !u.RescheduledAt.IsZero() {
			columns = append(columns, "RescheduledAt")
		}
		return columns, nil
	case appointmentCompleteUpdates:
		return []string{"Status", "Observations", "Conclusions", "Summary", "Notes"}, nil
	case appointmentSessionUpdates:
		return []string{"Observations", "Conclusions", "Summary", "Notes"}, nil
	case appointmentStartUpdates:
		return []string{"Status"}, nil
	case appointmentPauseUpdates:
		return []string{"Status", "PendingAt"}, nil
	case appointmentCancelUpdates:
		return []string{"Status", "CanceledAt"}, nil
	default:
		return nil, fmt.Errorf("select appointment columns: unsupported updates %T", updates)
	}
}
