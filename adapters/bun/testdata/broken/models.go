// Package broken has a relation whose foreign-key column does not exist.
package broken

import "github.com/uptrace/bun"

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID int64 `bun:",pk"`
}

type Post struct {
	bun.BaseModel `bun:"table:posts"`

	ID     int64 `bun:",pk"`
	Author *User `bun:"rel:belongs-to,join:author_id=id"`
}
