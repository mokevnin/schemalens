// Package manifest holds the schemalens manifest types for programs that build
// a manifest themselves: the extended-mode libraries (adapters/gormext,
// adapters/bunext) and any custom adapter written in Go. It knows no ORM and
// has no dependencies. The document it writes is accepted by
// `schemalens check --manifest`; `schemalens manifest-schema` prints the JSON
// Schema it follows.
package manifest

import (
	"encoding/json"
	"io"
)

// Version is the manifest version these types produce.
const Version = 1

// Manifest describes a model. Names are physical, exactly as they exist in the
// database.
type Manifest struct {
	Version int     `json:"version"`
	Tables  []Table `json:"tables"`
}

// New returns an empty manifest of the current version.
func New() *Manifest { return &Manifest{Version: Version, Tables: []Table{}} }

// Table is a model entity mapped to a database table.
type Table struct {
	Schema       string     `json:"schema,omitempty"`
	Name         string     `json:"name"`
	Source       *Position  `json:"source,omitempty"`
	Columns      []Column   `json:"columns"`
	PrimaryKey   []string   `json:"primary_key,omitempty"`
	JoinTableFor []TableRef `json:"join_table_for,omitempty"`
	Uniques      []Unique   `json:"uniques,omitempty"`
	Relations    []Relation `json:"relations,omitempty"`
}

// Column is a model field mapped to a database column.
type Column struct {
	Name     string    `json:"name"`
	Required bool      `json:"required,omitempty"`
	Unique   bool      `json:"unique,omitempty"`
	MaxLen   int       `json:"max_length,omitempty"`
	Enum     []string  `json:"enum,omitempty"`
	Source   *Position `json:"source,omitempty"`
}

// Unique is a uniqueness over several columns.
type Unique struct {
	Columns []string `json:"columns"`
}

// Relation is a foreign key: Columns of the table reference References.
type Relation struct {
	Columns    []string `json:"columns"`
	References TableRef `json:"references"`
	RefColumns []string `json:"ref_columns,omitempty"`
}

// TableRef names a table.
type TableRef struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table"`
}

// Position is a place in the source code.
type Position struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Write prints the manifest as an indented JSON document followed by a
// newline: exactly what an adapter prints on stdout.
func (m *Manifest) Write(w io.Writer) error {
	if m.Tables == nil {
		m.Tables = []Table{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}
