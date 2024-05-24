package model

import (
	"strconv"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"github.com/edgarsilva/go-scaffold/internal/util"
	"gorm.io/gorm"
)

type AppointmentStatus string

const (
	ApntStatusDraft       AppointmentStatus = "draft"
	ApntStatusSent        AppointmentStatus = "sent"
	ApntStatusViewed      AppointmentStatus = "viewed"
	ApntStatusConfirmed   AppointmentStatus = "confirmed"
	ApntStatusBegin       AppointmentStatus = "begin"
	ApntStatusDone        AppointmentStatus = "done"
	ApntStatusCanceled    AppointmentStatus = "canceled"
	ApntStatusRescheduled AppointmentStatus = "rescheduled"
)

type Appointment struct {
	BookedAt      time.Time      `gorm:"index;default:null;not null" form:"-"`
	SentAt        time.Time      `gorm:"index;default:null"`
	ViewedAt      time.Time      `gorm:"index;default:null"`
	ConfirmedAt   time.Time      `gorm:"index;default:null"`
	BeginAt       time.Time      `gorm:"index;default:null"`
	DoneAt        time.Time      `gorm:"index;default:null"`
	CanceledAt    time.Time      `gorm:"index;default:null"`
	RescheduledAt time.Time      `gorm:"index;default:null"`
	AcceptedAt    time.Time      `gorm:"index;default:null"`
	NoShowAt      time.Time      `gorm:"index;default:null"`
	DeletedAt     gorm.DeletedAt `form:"index"`
	ModelBase
	Summary      string            `form:"summary"`
	Observations string            `form:"observations"`
	Conclusions  string            `form:"conclusions"`
	Notes        string            `form:"notes"`
	ExtID        string            `form:"extId"`
	Hashtags     string            `form:"hashtags"`
	UserID       string            `gorm:"index;default:null;not null"`
	ClinicID     string            `gorm:"index;default:null;not null" form:"clinicId"`
	PatientID    string            `gorm:"index;default:null;not null" form:"patientId"`
	Status       AppointmentStatus `gorm:"index;default:draft;not null;type:string" form:"status"`
	Clinic       Clinic
	User         User
	Patient      Patient
	Duration     int `form:"duration"`
	Cost         int `form:"-"`
	BookedMonth  int
	BookedMinute int
	BookedHour   int
	BookedDay    int
	BookedYear   int
}

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	a.ID = xid.New("apnt")
	return nil
}

func (a *Appointment) InputCostValue() string {
	v := strconv.FormatFloat(float64(a.Cost/100), 'f', 1, 32)

	return v
}

func AppointmentsBookedToday(db *gorm.DB) *gorm.DB {
	t := util.BoD(time.Now())

	return db.Where("booked_at > ?", t).Where("booked_at < ?", t.Add(time.Hour*24))
}

func AppointmentsBookedThisWeek(db *gorm.DB) *gorm.DB {
	t := util.BoW(time.Now())

	return db.Where("booked_at > ?", t).Where("booked_at < ?", t.Add(time.Hour*24*7))
}

func AppointmentsBookedThisMonth(db *gorm.DB) *gorm.DB {
	t := util.BoM(time.Now())
	dinm := util.DaysInMonth(t.Month(), t.Year())

	return db.Where("booked_at > ?", t).Where("booked_at < ?", t.Add(time.Hour*24*dinm))
}
