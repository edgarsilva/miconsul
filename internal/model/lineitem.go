package model

import (
	"github.com/edgarsilva/go-scaffold/internal/lib/xid"
	"gorm.io/gorm"
)

type LineItem struct {
	UserID          string `gorm:"index;default:null;not null"`
	extID           string
	PurchaseOrderID string
	User            User // Belongs to User
	ModelBase
	PurchaseOrder PurchaseOrder // Belongs to PurchaseOrder
	Amount        uint
	Quantity      uint
}

func (li *LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	li.ID = xid.New("lit")
	return nil
}
