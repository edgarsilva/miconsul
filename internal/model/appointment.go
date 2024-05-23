package model

import (
	"strconv"
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Appointment struct {
	BookedAt      time.Time      `gorm:"index;default:null;not null"`
	ConfirmedAt   time.Time      `gorm:"index;default:null"`
	CanceledAt    time.Time      `gorm:"index;default:null"`
	RescheduledAt time.Time      `gorm:"index;default:null"`
	AcceptedAt    time.Time      `gorm:"index;default:null"`
	NoShowAt      time.Time      `gorm:"index;default:null"`
	DeletedAt     gorm.DeletedAt `form:"index"`
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
	Duration     int `form:"duration"`
	Cost         int `form:"cost"`
	BookedMonth  int
	BookedMinute int
	BookedHour   int
	BookedDay    int
	BookedYear   int
	Confirmed    bool
	Canceled     bool
	Rescheduled  bool
	Accepted     bool
	NoShow       bool
}

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	a.ID = xid.New("apnt")
	return nil
}

func (a *Appointment) CostValue() string {
	v := strconv.FormatFloat(float64(a.Cost/10), 'f', 1, 32)

	return v
}
