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
	EventActionCompleted EventAction = "completed"
	EventActionConfirmed EventAction = "confirmed"
	EventActionSent      EventAction = "sent"
	EventActionDelivered EventAction = "delivered"
	EventActionFailed    EventAction = "failed"
	EventActionSuccess   EventAction = "success"
)

// FeedEventSource defines the contract for models that can generate feed events.
type FeedEventSource interface {
	FeedEventRef() (id string, typ string)
	FeedEventSubject() (name, id, typ, url string)
	FeedEventTarget() (name, id, typ, url string)
}

// NewFeedEvent builds a FeedEvent from a source model and action.
func NewFeedEvent(action EventAction, actorName, actorID, actorURL string, source FeedEventSource) FeedEvent {
	refID, refType := source.FeedEventRef()
	subName, subID, subType, subURL := source.FeedEventSubject()
	tgtName, tgtID, tgtType, tgtURL := source.FeedEventTarget()

	return FeedEvent{
		Name:              string(action),
		Action:            string(action),
		Subject:           subName,
		SubjectID:         subID,
		SubjectType:       subType,
		SubjectURL:        subURL,
		Target:            tgtName,
		TargetID:          tgtID,
		TargetType:        tgtType,
		TargetURL:         tgtURL,
		Actor:             actorName,
		ActorID:           actorID,
		ActorURL:          actorURL,
		FeedEventableID:   refID,
		FeedEventableType: refType,
		OcurredAt:         time.Now(),
	}
}

type FeedEvent struct {
	ID                uint   `gorm:"primaryKey"`
	UID               string `gorm:"uniqueIndex;default:null;not null"`
	ExtID             string `gorm:"index;default:'';not null"`
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
	Actor             string
	ActorID           string
	ActorURL          string
	OcurredAt         time.Time `gorm:"index;default:null"`
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
