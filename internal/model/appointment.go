package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Appointment struct {
	BookedAt      time.Time      `gorm:"bookedAt"`
	OcurredAt     time.Time      `gorm:"occurredAt"`
	DeliveredAt   time.Time      `gorm:"deliveredAt"`
	ServedAt      time.Time      `gorm:"servedAt"`
	ConfirmedAt   time.Time      `gorm:"confirmedAt"`
	AcceptedAt    time.Time      `gorm:"acceptedAt"`
	CanceledAt    time.Time      `gorm:"canceledAt"`
	NoShowAt      time.Time      `gorm:"noShowAt"`
	RescheduledAt time.Time      `gorm:"rescheduledAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	ModelBase
	Summary      string `form:"summary"`
	Observations string `form:"observations"`
	Conclusions  string `form:"conclusions"`
	Notes        string `form:"notes"`
	ExtID        string `form:"extId"`
	Hashtags     string `form:"hashtags"`
	UserID       string `gorm:"index;default:null;not null"`
	ClinicID     string `gorm:"index;default:null;not null" form:"clinicId"`
	PatientID    string `gorm:"index;default:null;not null" form:"patientId"`
	Clinic       Clinic
	User         User
	Patient      Patient
	BookedMonth  int
	BookedMinute int
	BookedHour   int
	BookedDay    int
	BookedYear   int
	Duration     int `form:"duration"`
	Notified     bool
	Served       bool
	Confirmed    bool
	Accepted     bool
	Canceled     bool
	NoShow       bool
}

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	a.ID = xid.New("apnt")
	return nil
}

func (a *Appointment) IsCanceled(tx *gorm.DB) bool {
	return a.CanceledAt.IsZero()
}

func (a *Appointment) DidHappen(tx *gorm.DB) bool {
	return !a.OcurredAt.IsZero()
}
