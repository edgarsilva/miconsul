package database

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/lib/xid"

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
	ResetTokenExpiresAt time.Time
	Name                string
	Email               string   `gorm:"uniqueIndex;default:null;not null"`
	Role                UserRole `gorm:"index;default:null;not null;type:string"`
	Password            string   `json:"-"`
	Theme               string
	ResetToken          string
	ModelBase
	Todos          []Todo
	Articles       []Article
	Comments       []Comment
	confirmedEmail bool
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.UID = xid.New("usr")
	return
}

type Todo struct {
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	Content   string    `gorm:"default:null;not null"`
	ModelBase
	User      User
	UserID    uint `gorm:"index;default:null;not null"`
	Completed bool
}

func (t *Todo) BeforeCreate(tx *gorm.DB) (err error) {
	t.UID = xid.New("tdo")
	return
}

type Article struct {
	CreatedAt time.Time `gorm:"index:,sort:desc"`
	UpdatedAt time.Time
	Title     string
	Content   string
	ModelBase
	Comments []Comment
	User     User
	UserID   uint `gorm:"index;default:null;not null"`
}

func (a *Article) BeforeCreate(tx *gorm.DB) (err error) {
	a.UID = xid.New("pst")
	return
}

type Comment struct {
	Content   string
	ArticleID string `gorm:"index;default:null;not null"`
	User      User   // Belongs to User
	ModelBase
	Article Article // Belongs to Article
	UserID  uint    `gorm:"index;default:null;not null"`
}

func (c *Comment) BeforeCreate(tx *gorm.DB) (err error) {
	c.UID = xid.New("cmt")
	return
}

type PurchaseOrder struct {
	extID string
	ModelBase
	LineItems []LineItem
	User      User
	Amount    uint
	Quantity  uint
	UserID    uint `gorm:"index;default:null;not null"`
}

func (po *PurchaseOrder) BeforeCreate(tx *gorm.DB) (err error) {
	po.UID = xid.New("por")
	return
}

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

func (li *LineItem) BeforeCreate(tx *gorm.DB) (err error) {
	li.UID = xid.New("lni")
	return
}
