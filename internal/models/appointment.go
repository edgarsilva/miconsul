package models

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/lib/libtime"
	"miconsul/internal/lib/xid"

	"gorm.io/gorm"
)

const DefaultTimezone = string(libtime.AmericaMexicoCity)

type AppointmentStatus string

const (
	AppointmentPending    AppointmentStatus = "pending"
	AppointmentConfirmed  AppointmentStatus = "confirmed"
	AppointmentInProgress AppointmentStatus = "in_progress"
	AppointmentDone       AppointmentStatus = "done"
	AppointmentCanceled   AppointmentStatus = "canceled"
)

func (s AppointmentStatus) IsValid() bool {
	switch s {
	case AppointmentConfirmed,
		AppointmentInProgress,
		AppointmentDone,
		AppointmentCanceled,
		AppointmentPending:
		return true
	default:
		return false
	}
}

type Appointment struct {
	BookedAt      time.Time `gorm:"index;default:null;not null" form:"-"`
	ViewedAt      time.Time `gorm:"default:null"`
	ConfirmedAt   time.Time `gorm:"default:null"`
	DoneAt        time.Time `gorm:"default:null"`
	CanceledAt    time.Time `gorm:"default:null"`
	PendingAt     time.Time `gorm:"default:null"`
	RescheduledAt time.Time `gorm:"default:null"`
	DeletedAt     gorm.DeletedAt
	ModelBase
	ID            uint              `gorm:"primaryKey" form:"-"`
	UID           string            `gorm:"uniqueIndex;default:null;not null" form:"-"`
	ExtID         string            `form:"extId"`
	Token         string            `json:"-" form:"-"`
	Summary       string            `form:"summary"`
	Observations  string            `form:"observations"`
	Conclusions   string            `form:"conclusions"`
	Notes         string            `form:"notes"`
	Hashtags      string            `form:"hashtags"`
	Timezone      string            `form:"timezone"`
	UserID        uint              `gorm:"index;default:null;not null" form:"-"`
	ClinicID      uint              `gorm:"index;default:null;not null" form:"clinicId"`
	PatientID     uint              `gorm:"index;default:null;not null" form:"patientId"`
	Status        AppointmentStatus `gorm:"index;default:pending;not null;type:string" form:"status"`
	FeedEvents    []FeedEvent       `gorm:"polymorphic:FeedEventable;"`
	Notifications []Notification    `gorm:"polymorphic:Notificationable;"`
	Clinic        Clinic
	User          User
	Patient       Patient
	Duration      int `form:"duration"`
	Price         int `form:"-"`
	BookedYear    int
	BookedMonth   int
	BookedDay     int
	BookedHour    int
	BookedMinute  int
	NoShow        bool `form:"noShow"`
	NotificationFlags
}

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	a.UID = xid.New("apnt")
	if a.Status == "" {
		a.Status = AppointmentPending
	}

	return nil
}

// BeforeSave stores BookedAt in UTC and validates the status.
func (a *Appointment) BeforeSave(tx *gorm.DB) error {
	a.BookedAt = a.BookedAt.UTC()

	// Allow partial updates where status is intentionally omitted.
	if a.Status == "" {
		return nil
	}

	if !a.Status.IsValid() {
		return errors.New("invalid appointment status")
	}

	return nil
}

func (a *Appointment) PriceInputValue() string {
	v := strconv.FormatFloat(float64(a.Price/100), 'f', 1, 32)

	return v
}

// PatientLinkGrace is how long after the appointment time its patient links
// keep working.
const PatientLinkGrace = time.Hour

// PatientCanConfirm reports whether the patient may still confirm or ask to
// change the date.
func (a Appointment) PatientCanConfirm() bool {
	switch a.Status {
	case AppointmentCanceled, AppointmentInProgress, AppointmentDone:
		return false
	default:
		return true
	}
}

// PatientCanCancel reports whether the patient may still cancel.
func (a Appointment) PatientCanCancel() bool {
	switch a.Status {
	case AppointmentInProgress, AppointmentDone:
		return false
	default:
		return true
	}
}

// ConfirmPath is the patient's public link to confirm the appointment.
func (a *Appointment) ConfirmPath() string {
	return "/p/a/" + a.Token + "/confirm"
}

// CancelPath is the patient's public link to cancel the appointment.
func (a *Appointment) CancelPath() string {
	return "/p/a/" + a.Token + "/cancel"
}

func (a Appointment) AlertableID() string {
	if a.ID == 0 {
		return ""
	}

	return strconv.FormatUint(uint64(a.ID), 10)
}

func (a Appointment) AlertableType() string {
	return "appointments"
}

// Scopes

// AppointmentBookedIn scopes appointments to the current timeframe in the
// default timezone.
func AppointmentBookedIn(timeframe libtime.Timeframe) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return appointmentBookedWithin(db, time.Now(), timeframe)
	}
}

// AppointmentWithStatus scopes appointments to statuses; with none it adds no
// condition.
func AppointmentWithStatus(statuses ...AppointmentStatus) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(statuses) == 0 {
			return db
		}

		return db.Where("appointments.status IN ?", statuses)
	}
}

// ParseAppointmentStatuses splits a comma-separated status list, trimming
// spaces and dropping empty entries.
func ParseAppointmentStatuses(list string) []AppointmentStatus {
	statuses := []AppointmentStatus{}
	for token := range strings.SplitSeq(list, ",") {
		if token = strings.TrimSpace(token); token != "" {
			statuses = append(statuses, AppointmentStatus(token))
		}
	}

	return statuses
}

func (a *Appointment) LocalTimezone() string {
	if a.Timezone == "" {
		return DefaultTimezone
	}

	if _, err := time.LoadLocation(a.Timezone); err != nil {
		return DefaultTimezone
	}

	return a.Timezone
}

func (a *Appointment) BookedAtInLocalTime() time.Time {
	localTime := libtime.InTimezone(a.BookedAt, a.LocalTimezone())
	return localTime
}

// FeedEventRef returns the polymorphic identity for this appointment.
func (a Appointment) FeedEventRef() (id string, typ string) {
	return a.UID, "appointments"
}

// FeedEventSubject returns the appointment as the subject of the event.
func (a Appointment) FeedEventSubject() (name, id, typ, url string) {
	return "Appointment", a.UID, "appointments", "/appointments/" + a.UID
}

// FeedEventTarget returns the patient as the target of the event.
func (a Appointment) FeedEventTarget() (name, id, typ, url string) {
	return a.Patient.Name, a.Patient.UID, "patients", "/patients/" + a.Patient.UID
}

func appointmentBookedWithin(db *gorm.DB, now time.Time, timeframe libtime.Timeframe) *gorm.DB {
	start, end := libtime.TimeframeBounds(now, DefaultTimezone, timeframe)

	return db.Where("booked_at >= ?", start.UTC()).Where("booked_at < ?", end.UTC())
}
