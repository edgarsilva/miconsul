package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type LineItem struct {
	extID           string
	PurchaseOrderID string
	User            User // Belongs to User
	ModelBase
	PurchaseOrder PurchaseOrder // Belongs to PurchaseOrder
	Amount        uint
	Quantity      uint
	UserID        uint `gorm:"index;default:null;not null"`
}

func (li LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	li.UID = xid.New("cmt")
	return nil
}
