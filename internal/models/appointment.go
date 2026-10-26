package models

import (
	"errors"
	"strconv"
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

func AppointmentWithPendingNotifications(db *gorm.DB) *gorm.DB {
	st := time.Now()
	year, month, day := st.Date()
	et := time.Date(year, month, day, st.Hour(), st.Minute(), 0, 0, st.Location()).Add(2 * time.Hour)

	return db.
		Where("booked_at > ?", st).
		Where("booked_at <= ?", et).
		Where("NOT EXISTS (SELECT 1 FROM notifications WHERE notifications.notificationable_id = CAST(appointments.id AS TEXT) AND notifications.notificationable_type = ? AND notifications.name = ? AND notifications.medium = ? AND notifications.status IN ?)", "appointments", "appointment_reminder", NotificationMediumEmail, []NotificationStatus{NotificationSent, NotificationSuccess})
}

func AppointmentBookedToday(db *gorm.DB) *gorm.DB {
	return appointmentBookedWithin(db, time.Now(), "day")
}

func AppointmentBookedThisWeek(db *gorm.DB) *gorm.DB {
	return appointmentBookedWithin(db, time.Now(), "week")
}

func AppointmentBookedThisMonth(db *gorm.DB) *gorm.DB {
	return appointmentBookedWithin(db, time.Now(), "month")
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

func appointmentBookedWithin(db *gorm.DB, now time.Time, timeframe string) *gorm.DB {
	start, end := bookedWindow(now, DefaultTimezone, timeframe)

	return db.Where("booked_at >= ?", start).Where("booked_at < ?", end)
}

// bookedWindow returns the UTC bounds [start, end) of the day, week or month
// that contains now in timezone tz. Any other timeframe means day.
func bookedWindow(now time.Time, tz string, timeframe string) (start, end time.Time) {
	local := libtime.InTimezone(now, tz)

	switch timeframe {
	case "week":
		start = libtime.BoW(local)
		end = start.AddDate(0, 0, 7)
	case "month":
		start = libtime.BoM(local)
		end = start.AddDate(0, 1, 0)
	default:
		start = libtime.BoD(local)
		end = start.AddDate(0, 0, 1)
	}

	return start.UTC(), end.UTC()
}
