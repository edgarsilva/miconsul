package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type PurchaseOrder struct {
	extID string
	ModelBase
	LineItems []LineItem
	User      User
	Amount    uint
	Quantity  uint
	UserID    uint `gorm:"index;default:null;not null"`
}

func (po PurchaseOrder) BeforeCreate(tx *gorm.DB) (err error) {
	po.UID = xid.New("cmt")
	return nil
}
