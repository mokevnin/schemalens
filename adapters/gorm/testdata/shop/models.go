// Package shop is a sample gorm project the adapter is proven on. It is
// ordinary gorm code; nothing in it knows about schemalens.
package shop

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// Address is embedded into User with a column prefix.
type Address struct {
	Street string `gorm:"size:200;not null"`
	City   string
}

// Audit is embedded anonymously, so its fields become columns of the table.
type Audit struct {
	CreatedBy string `gorm:"column:created_by_user;not null"`
}

type User struct {
	gorm.Model
	Email     string  `gorm:"not null;unique;size:255"`
	Name      string  `gorm:"type:text"`
	Nickname  *string `gorm:"column:nick"`
	Home      Address `gorm:"embedded;embeddedPrefix:home_"`
	Orders    []Order `gorm:"foreignKey:BuyerID"`
	Company   Company
	CompanyID uint
	Secret    string `gorm:"-"`
	notes     string
	LastLogin sql.NullTime
	Languages []Language `gorm:"many2many:user_languages"`
	Friends   []*User    `gorm:"many2many:user_friends;joinForeignKey:UserRef;joinReferences:FriendRef"`
}

type Order struct {
	ID       uint `gorm:"primaryKey"`
	BuyerID  uint `gorm:"not null"`
	Total    int
	PlacedAt time.Time
	Audit
	Items []Item `gorm:"many2many:order_items;foreignKey:ID;references:SKU"`
}

type Company struct {
	ID   uint
	Name string `gorm:"not null"`
}

// TableName is computed at run time, so the config override names the table.
func (Company) TableName() string { return tablePrefix() + "companies" }

func tablePrefix() string { return "legacy_" }

type Label struct {
	ID    uint
	Title string
}

// TableName is a literal: the adapter reads it without help.
func (Label) TableName() string { return "tags" }

// Plain is not a model: no gorm tags, no gorm.Model.
type Plain struct {
	Value string
}

// Language is joined to User through user_languages; gorm names the join
// table's columns user_id and language_id.
type Language struct {
	ID   uint
	Code string `gorm:"not null"`
}

// Item is joined to Order on its SKU instead of its primary key.
type Item struct {
	ID  uint
	SKU string `gorm:"unique;size:40"`
}
