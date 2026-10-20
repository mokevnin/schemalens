// Package dialect names the database engines schemalens supports.
package dialect

// Dialect identifies a database engine.
type Dialect string

// Supported dialects.
const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
)

// All lists every supported dialect.
var All = []Dialect{SQLite, Postgres, MySQL}
