// Package shop is a sample bun project the adapter is proven on. It is
// ordinary bun code; nothing in it knows about schemalens.
package shop

import (
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

// Address is embedded into User with a column prefix.
type Address struct {
	Street string `bun:",notnull,type:varchar(200)"`
	City   string
}

// Audit is embedded anonymously, so its fields become columns of the table.
type Audit struct {
	CreatedBy string `bun:"created_by_user,notnull"`
}

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        int64    `bun:",pk,autoincrement"`
	Email     string   `bun:",notnull,unique,type:varchar(255)"`
	Name      string   `bun:"type:text"`
	Nickname  *string  `bun:"nick"`
	Home      Address  `bun:"embed:home_"`
	Orders    []Order  `bun:"rel:has-many,join:id=buyer_id"`
	Company   *Company `bun:"rel:belongs-to,join:company_id=id"`
	CompanyID int64
	Secret    string `bun:"-"`
	notes     string
	LastLogin sql.NullTime `bun:",nullzero"`
}

type Order struct {
	bun.BaseModel `bun:"table:orders"`

	ID       int64 `bun:",pk,autoincrement"`
	BuyerID  int64 `bun:",notnull"`
	Total    int
	PlacedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	Slot     string    `bun:",unique:buyer_slot"`
	Region   string    `bun:",unique:buyer_slot"`
	Audit
}

// Company is named in code only by its tag, with a schema.
type Company struct {
	bun.BaseModel `bun:"table:billing.companies"`

	ID   int64  `bun:",pk"`
	Name string `bun:",notnull"`
}

// UserProfile has no table tag: bun names the table user_profiles.
type UserProfile struct {
	bun.BaseModel

	UserID int64 `bun:",pk"`
	User   *User `bun:"rel:belongs-to"`
	Bio    string
}

// Category shows bun's pluralization: categories.
type Category struct {
	bun.BaseModel

	ID    int64 `bun:",pk"`
	Title string
}

// Label is renamed by the config override.
type Label struct {
	bun.BaseModel `bun:"table:labels_set_in_code"`

	ID    int64 `bun:",pk"`
	Title string
}

// Plain is not a model: no bun tags, no bun.BaseModel.
type Plain struct {
	Value string
}
