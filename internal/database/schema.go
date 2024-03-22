package database

import (
	"fiber-blueprint/internal/xid"
	"time"

	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UID       string `gorm:"uniqueIndex;type:string;not null"`
	ID        uint   `gorm:"primarykey"`
}

type User struct {
	gorm.Model
	ModelBase
	Name     string
	Email    string
	Password string

	// Has many
	Todos    []Todo
	Posts    []Post
	Comments []Comment
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.NewConcat("usr")
	return
}

type Todo struct {
	ModelBase
	Title     string
	Body      string
	Content   string
	Priority  string
	Completed bool
	CreatedAt time.Time `gorm:"index:,sort:desc"`

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User
}

func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.UID = xid.NewConcat("tdo")
	return
}

type Post struct {
	ModelBase
	Title     string
	Content   string
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	UpdatedAt time.Time

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	// Has many
	Comments []Comment
}

func (p *Post) BeforeCreate(tx *gorm.DB) (err error) {
	p.UID = xid.NewConcat("pst")
	return
}

type Comment struct {
	ModelBase
	Content string

	// Belongs to
	PostID string
	Post   Post

	UserID string `gorm:"index;default:null;not null"`
	User   User
}

func (c *Comment) BeforeCreate(tx *gorm.DB) (err error) {
	c.UID = xid.NewConcat("cmt")
	return
}

type PurchaseOrder struct {
	ModelBase
	extID    string
	Amount   int
	Quantity int

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	// Has many
	LineItems []LineItem
}

func (po *PurchaseOrder) BeforeCreate(tx *gorm.DB) (err error) {
	po.UID = xid.NewConcat("por")
	return
}

type LineItem struct {
	ModelBase
	extID    string
	Amount   int
	Quantity int

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	PurchaseOrderID string
	PurchaseOrder   PurchaseOrder
}

func (li *LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	li.UID = xid.NewConcat("lni")
	return
}
