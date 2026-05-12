package models

import (
	"miconsul/internal/lib/xid"

	"gorm.io/gorm"
)

type NotificationMedium string

const (
	NotificationMediumEmail     NotificationMedium = "email"
	NotificationMediumFacebook  NotificationMedium = "facebook"
	NotificationMediumWhatsapp  NotificationMedium = "whatsapp"
	NotificationMediumMessenger NotificationMedium = "messenger"
	NotificationMediumTelegram  NotificationMedium = "telegram"
)

type NotificationStatus string

const (
	NotificationPending   NotificationStatus = "pending"
	NotificationSent      NotificationStatus = "sent"
	NotificationDelivered NotificationStatus = "delivered"
	NotificationViewed    NotificationStatus = "viewed"
	NotificationFailed    NotificationStatus = "failed"
	NotificationSuccess   NotificationStatus = "success"
)

type Notification struct {
	ID                   uint               `gorm:"primaryKey" form:"-"`
	UID                  string             `gorm:"uniqueIndex;default:null;not null" form:"-"`
	Medium               NotificationMedium `gorm:"index;default:null;not null;type:string" form:"-"`
	Name                 string             `gorm:"index;default:null;not null"`
	Title                string
	Sub                  string
	Message              string
	From                 string
	To                   string
	Status               NotificationStatus `gorm:"index;default:pending;not null;type:string" form:"-"`
	NotificationableID   string             `gorm:"column:alertable_id;index:poly_fevnt_idx"`
	NotificationableType string             `gorm:"column:alertable_type;index:poly_fevnt_idx"`
	ModelBase
}

func (Notification) TableName() string {
	return "notifications"
}

func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	n.UID = xid.New("ntfy")
	return nil
}
