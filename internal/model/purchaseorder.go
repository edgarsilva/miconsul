package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type PurchaseOrder struct {
	extID  string
	UserID string `gorm:"index;default:null;not null"`
	ModelBase
	LineItems []LineItem
	User      User
	Amount    uint
	Quantity  uint
}

func (po *PurchaseOrder) BeforeCreate(tx *gorm.DB) (err error) {
	po.ID = xid.New("por")
	return nil
}
