package database

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/xid"

	"gorm.io/gorm"
)

type ModelBase struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	UID       string `gorm:"uniqueIndex;default:null;not null"`
	ID        uint   `gorm:"primarykey"`
}

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
	UserRoleGuest UserRole = "guest"
	UserRoleAnon  UserRole = "anon"
	UserRoleTest  UserRole = "test"
)

type User struct {
	Name     string
	Email    string   `gorm:"uniqueIndex;default:null;not null"`
	Role     UserRole `gorm:"index;default:null;not null;type:string"`
	Password string   `json:"-"`
	Theme    string

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
	Content   string `gorm:"default:null;not null"`
	Completed bool
	CreatedAt time.Time `gorm:"index:,sort:desc"`

	// Belongs to
	UserID uint `gorm:"index;default:null;not null"`
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
	UserID uint `gorm:"index;default:null;not null"`
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

	UserID uint `gorm:"index;default:null;not null"`
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
	UserID uint `gorm:"index;default:null;not null"`
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
	UserID uint `gorm:"index;default:null;not null"`
	User   User

	PurchaseOrderID string
	PurchaseOrder   PurchaseOrder

	ModelBase
}

func (li *LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	li.UID = xid.New("lni")
	return
}
