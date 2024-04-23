package database

import (
	"rtx-blog/internal/xid"
	"time"

	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UID       string `gorm:"uniqueIndex;default:null;not null"`
	ID        uint   `gorm:"primarykey"`
}

type User struct {
	Name     string
	Email    string `gorm:"uniqueIndex;default:null;not null"`
	Role     string `gorm:"index;default:null;not null"`
	Password string `json:"-"`

	// Has many
	Todos    []Todo
	Posts    []Post
	Comments []Comment

	ModelBase
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.New("usr")
	return
}

type Todo struct {
	Title     string
	Body      string
	Content   string `gorm:"default:null;not null"`
	Priority  string
	Completed bool
	CreatedAt time.Time `gorm:"index:,sort:desc"`

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	ModelBase
}

func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.UID = xid.New("tdo")
	return
}

type Post struct {
	Title     string
	Content   string
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	UpdatedAt time.Time

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	// Has many
	Comments []Comment

	ModelBase
}

func (p *Post) BeforeCreate(tx *gorm.DB) (err error) {
	p.UID = xid.New("pst")
	return
}

type Comment struct {
	Content string

	// Belongs to
	PostID string `gorm:"index;default:null;not null"`
	Post   Post

	UserID string `gorm:"index;default:null;not null"`
	User   User

	ModelBase
}

func (c *Comment) BeforeCreate(tx *gorm.DB) (err error) {
	c.UID = xid.New("cmt")
	return
}

type PurchaseOrder struct {
	extID    string
	Amount   uint
	Quantity uint

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	// Has many
	LineItems []LineItem

	ModelBase
}

func (po *PurchaseOrder) BeforeCreate(tx *gorm.DB) (err error) {
	po.UID = xid.New("por")
	return
}

type LineItem struct {
	extID    string
	Amount   uint
	Quantity uint

	// Belongs to
	UserID string `gorm:"index;default:null;not null"`
	User   User

	PurchaseOrderID string
	PurchaseOrder   PurchaseOrder

	ModelBase
}

func (li *LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	li.UID = xid.New("lni")
	return
}
