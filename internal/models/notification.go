package models

import (
	"miconsul/internal/lib/xid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationMedium string

const (
	NotificationMediumEmail     NotificationMedium = "email"
	NotificationMediumSMS       NotificationMedium = "sms"
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
	ID             uint               `gorm:"primaryKey" form:"-"`
	UID            string             `gorm:"uniqueIndex;default:null;not null" form:"-"`
	Medium         NotificationMedium `gorm:"index;default:null;not null;type:string" form:"-"`
	Name           string             `gorm:"index;default:null;not null"`
	Title          string
	Sub            string
	Message        string
	From           string
	To             string
	Status         NotificationStatus `gorm:"index;default:pending;not null;type:string" form:"-"`
	NotifiableID   string             `gorm:"column:notifiable_id;index:poly_fevnt_idx"`
	NotifiableType string             `gorm:"column:notifiable_type;index:poly_fevnt_idx"`
	ModelBase
}

func (Notification) TableName() string {
	return "notifications"
}

func (n *Notification) BeforeCreate(tx *gorm.DB) (err error) {
	n.UID = xid.New("ntfy")
	return nil
}

// Notifiable is a record that notifications belong to.
type Notifiable interface {
	NotifiableID() string
	NotifiableType() string
}

// NotificationsOf scopes notifications to those that belong to owner, for the
// classic *gorm.DB API.
func NotificationsOf(owner Notifiable) func(*gorm.DB) *gorm.DB {
	ofOwner := NotificationsOfStatement(owner)

	return func(db *gorm.DB) *gorm.DB {
		ofOwner(db.Statement)

		return db
	}
}

// NotificationsOfStatement scopes notifications to those that belong to owner,
// for gorm.G[T](db).Scopes.
func NotificationsOfStatement(owner Notifiable) func(*gorm.Statement) {
	return func(stmt *gorm.Statement) {
		stmt.AddClause(clause.Where{Exprs: []clause.Expression{
			clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: "notifiable_id"}, Value: owner.NotifiableID()},
			clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: "notifiable_type"}, Value: owner.NotifiableType()},
		}})
	}
}
