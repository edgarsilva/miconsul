package model

import (
	"github.com/edgarsilva/miconsul/internal/lib/xid"
	"gorm.io/gorm"
)

type AlertMedium string

const (
	AlertMediumEmail     AlertMedium = "email"
	AlertMediumFacebook  AlertMedium = "facebook"
	AlertMediumWhatsapp  AlertMedium = "whatsapp"
	AlertMediumMessenger AlertMedium = "messenger"
	AlertMediumTelegram  AlertMedium = "telegram"
)

type AlertStatus string

const (
	AlertPending   AlertStatus = "pending"
	AlertSent      AlertStatus = "sent"
	AlertDelivered AlertStatus = "delivered"
	AlertViewed    AlertStatus = "viewed"
	AlertFailed    AlertStatus = "failed"
	AlertSuccess   AlertStatus = "success"
)

type Alert struct {
	Medium        AlertMedium `gorm:"index;default:null;not null;type:string" form:"-"`
	Name          string      `gorm:"index;default:null;not null"`
	Title         string
	Sub           string
	Message       string
	From          string
	To            string
	Status        AlertStatus `gorm:"index;default:pending;not null;type:string" form:"-"`
	AlertableID   string      `gorm:"index:poly_fevnt_idx"`
	AlertableType string      `gorm:"index:poly_fevnt_idx"`
	ModelBase
}

func (a *Alert) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID = xid.New("alrt")
	return nil
}
