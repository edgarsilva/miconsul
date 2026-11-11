package appointment

import (
	"time"

	"miconsul/internal/lib/amount"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	view "miconsul/internal/views"
)

type appointmentUpsertInput struct {
	ClinicUID  string `form:"clinicId"`
	PatientUID string `form:"patientId"`
	Duration   int    `form:"duration"`
	BookedAt   string `form:"bookedAt"`
	Price      string `form:"price"`
}

type appointmentCompleteInput struct {
	Observations string `form:"observations"`
	Conclusions  string `form:"conclusions"`
	Summary      string `form:"summary"`
	Notes        string `form:"notes"`
}

// bookedAtUTC parses the form time as clinic-local time and returns it in UTC.
// It falls back to now when the value is empty or invalid.
func (in appointmentUpsertInput) bookedAtUTC() time.Time {
	bookedAt, err := time.Parse(view.FormTimeFormat, in.BookedAt)
	if err != nil {
		bookedAt = time.Now()
	}

	return libtime.NewInTimezone(bookedAt, models.DefaultTimezone).UTC()
}

func (in appointmentUpsertInput) toAppointment(userID, clinicID, patientID uint, token string) models.Appointment {
	bookedAt := in.bookedAtUTC()

	return models.Appointment{
		Token:        token,
		UserID:       userID,
		ClinicID:     clinicID,
		PatientID:    patientID,
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Timezone:     models.DefaultTimezone,
		Price:        amount.StrToAmount(in.Price),
		Duration:     in.Duration,
	}
}

func (in appointmentUpsertInput) toAppointmentUpdates(clinicID, patientID uint) appointmentUpdates {
	bookedAt := in.bookedAtUTC()

	return appointmentUpdates{
		BookedAt:     bookedAt,
		BookedYear:   bookedAt.Year(),
		BookedMonth:  int(bookedAt.Month()),
		BookedDay:    bookedAt.Day(),
		BookedHour:   bookedAt.Hour(),
		BookedMinute: bookedAt.Minute(),
		Price:        amount.StrToAmount(in.Price),
		ClinicID:     clinicID,
		PatientID:    patientID,
		Duration:     in.Duration,
	}
}

func (in appointmentCompleteInput) toAppointmentSessionUpdates() appointmentSessionUpdates {
	return appointmentSessionUpdates{
		Observations: in.Observations,
		Conclusions:  in.Conclusions,
		Summary:      in.Summary,
		Notes:        in.Notes,
	}
}

func (in appointmentCompleteInput) toAppointmentCompleteUpdates() appointmentCompleteUpdates {
	return appointmentCompleteUpdates{
		Status:       models.AppointmentDone,
		Observations: in.Observations,
		Conclusions:  in.Conclusions,
		Summary:      in.Summary,
		Notes:        in.Notes,
	}
}
