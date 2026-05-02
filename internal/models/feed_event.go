package models

import (
	"miconsul/internal/lib/xid"
	"time"

	"gorm.io/gorm"
)

type EventAction string

const (
	EventActionCreated   EventAction = "created"
	EventActionReplaced  EventAction = "replaced"
	EventActionUpdated   EventAction = "updated"
	EventActionDeleted   EventAction = "deleted"
	EventActionChanged   EventAction = "changed"
	EventActionCanceled  EventAction = "canceled"
	EventActionSent      EventAction = "sent"
	EventActionDelivered EventAction = "delivered"
	EventActionFailed    EventAction = "failed"
	EventActionSuccess   EventAction = "success"
)

type FeedEvent struct {
	ID                uint   `gorm:"primaryKey"`
	UID               string `gorm:"uniqueIndex;default:null;not null"`
	extID             string `gorm:"index;default:null;not null"`
	Name              string `gorm:"index;default:null;not null"`
	Subject           string
	SubjectID         string `gorm:"index:fe_subject_idx;default:null;not null"`
	SubjectType       string `gorm:"index:fe_subject_idx;default:null;not null"`
	SubjectURL        string
	Action            string `gorm:"index"`
	Target            string `gorm:"index:fe_target_idx;default:null;not null"`
	TargetID          string `gorm:"index:fe_target_idx;default:null;not null"`
	TargetType        string
	TargetURL         string
	OcurredAt         time.Time `gorm:"index;default:null"`
	onAttr            string
	from              string
	to                string
	Extra1            string
	Extra2            string
	Extra3            string
	FeedEventableID   string `gorm:"index:fe_poly_idx"`
	FeedEventableType string `gorm:"index:fe_poly_idx"`
	ModelBase
}

func (fe *FeedEvent) BeforeCreate(tx *gorm.DB) (err error) {
	fe.UID = xid.New("fevn")
	return nil
}
