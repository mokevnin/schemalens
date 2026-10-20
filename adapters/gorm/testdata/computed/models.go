// Package computed has a table name only known at run time and no config.
package computed

import "os"

type Account struct {
	ID    uint `gorm:"primaryKey"`
	Owner string
}

func (Account) TableName() string { return os.Getenv("TABLE_PREFIX") + "accounts" }
