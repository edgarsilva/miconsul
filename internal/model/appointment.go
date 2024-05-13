package model

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type Appointment struct {
	BookedAt   time.Time
	CanceledAt time.Time
	OcurredAt  time.Time
	NoShowAt   time.Time
	ModelBase
	ExtID        string
	UserID       string `gorm:"index;default:null;not null"`
	ClinicID     string `gorm:"index;default:null;not null"`
	PatientID    string `gorm:"index;default:null;not null"`
	Summary      string
	Observations string
	Conclusions  string
	Hashtags     string
	Clinic       Clinic
	Patient      Patient
	User         User
	BookedMonth  int
	BookedMinute int
	BookedHour   int
	BookedDay    int
	BookedYear   int
	Duration     int
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
