package model

import (
	"strconv"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/common"
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type AppointmentStatus string

const (
	ApntStatusDraft       AppointmentStatus = "draft"
	ApntStatusConfirmed   AppointmentStatus = "confirmed"
	ApntStatusDone        AppointmentStatus = "done"
	ApntStatusCanceled    AppointmentStatus = "canceled"
	ApntStatusPending     AppointmentStatus = "pending"
	ApntStatusRescheduled AppointmentStatus = "rescheduled"
)

type Appointment struct {
	BookedAt            time.Time      `gorm:"index;default:null;not null" form:"-"`
	OldBookedAt         time.Time      `gorm:"index;default:null" form:"-"`
	BookedAlertSentAt   time.Time      `gorm:"default:null"`
	ReminderAlertSentAt time.Time      `gorm:"default:null"`
	ViewedAt            time.Time      `gorm:"default:null"`
	ConfirmedAt         time.Time      `gorm:"default:null"`
	DoneAt              time.Time      `gorm:"default:null"`
	CanceledAt          time.Time      `gorm:"default:null"`
	PendingAt           time.Time      `gorm:"default:null"`
	RescheduledAt       time.Time      `gorm:"default:null"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
	ModelBase
	ID           string            `gorm:"primarykey;default:null;not null"`
	ExtID        string            `form:"extId"`
	Token        string            `form:"-"`
	Summary      string            `form:"summary"`
	Observations string            `form:"observations"`
	Conclusions  string            `form:"conclusions"`
	Notes        string            `form:"notes"`
	Hashtags     string            `form:"hashtags"`
	UserID       string            `gorm:"index;default:null;not null"`
	ClinicID     string            `gorm:"index;default:null;not null" form:"clinicId"`
	PatientID    string            `gorm:"index;default:null;not null" form:"patientId"`
	Status       AppointmentStatus `gorm:"index;default:draft;not null;type:string" form:"status"`
	FeedEvents   []FeedEvent       `gorm:"polymorphic:FeedEventable;"`
	Alerts       []Alert           `gorm:"polymorphic:Alertable;"`
	Clinic       Clinic
	User         User
	Patient      Patient
	Duration     int `form:"duration"`
	Cost         int `form:"-"`
	BookedYear   int
	BookedMonth  int
	BookedDay    int
	BookedHour   int
	BookedMinute int
	NoShow       bool `form:"noShow"`
	NotificationFlags
}

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	a.ID = xid.New("apnt")

	return nil
}

func (a *Appointment) InputCostValue() string {
	v := strconv.FormatFloat(float64(a.Cost/100), 'f', 1, 32)

	return v
}

func (a *Appointment) ConfirmURL() string {
	return "/appointments/" + a.ID + "/confirm"
}

func (a *Appointment) CancelURL() string {
	return "/appointments/" + a.ID + "/cancel"
}

func (a *Appointment) RescheduledURL() string {
	return "/appointments/" + a.ID + "/reschedule"
}

// Scopes
func AppointmentWithPendingAlerts(db *gorm.DB) *gorm.DB {
	st := time.Now()
	year, month, day := st.Date()
	et := time.Date(year, month, day, st.Hour(), st.Minute(), 0, 0, st.Location()).Add(2 * time.Hour)

	return db.
		Where("booked_at > ?", st).
		Where("booked_at <= ?", et).
		Where("reminder_alert_sent_at IS NULL")
}

func AppointmentBookedToday(db *gorm.DB) *gorm.DB {
	t := common.BoD(time.Now())

	return db.Where("booked_at > ?", t).Where("booked_at < ?", t.Add(time.Hour*24))
}

func AppointmentBookedThisWeek(db *gorm.DB) *gorm.DB {
	t := common.BoW(time.Now())

	return db.Where("booked_at > ?", t).Where("booked_at < ?", t.Add(time.Hour*24*7))
}

func AppointmentBookedThisMonth(db *gorm.DB) *gorm.DB {
	t := common.BoM(time.Now())
	dinm := common.DaysInMonth(t.Month(), t.Year())

	return db.Where("booked_at > ?", t).Where("booked_at < ?", t.Add(time.Hour*24*dinm))
}
