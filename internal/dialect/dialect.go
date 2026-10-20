// Package dialect names the database engines schemalens supports.
package dialect

// Dialect identifies a database engine.
type Dialect string

// Supported dialects. Later tickets add PostgreSQL and MySQL.
const (
	SQLite Dialect = "sqlite"
)
