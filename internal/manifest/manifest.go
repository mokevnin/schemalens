// Package manifest defines the model manifest: the versioned JSON document an
// adapter prints to describe a model. Names are physical, exactly as they
// exist in the database; the core applies no ORM naming rules.
//
// Parse is the single entry point for turning bytes into a Manifest. Every
// source of manifests (a file, an adapter subprocess, a library caller) goes
// through it so they all get the same validation.
package manifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/mokevnin/schemalens/internal/finding"
)

// Version is the manifest version this build understands.
const Version = 1

var printer = message.NewPrinter(language.English)

//go:embed manifest.schema.json
var schemaJSON []byte

// Schema returns the JSON Schema of the manifest, as published by
// `schemalens manifest-schema`.
func Schema() []byte { return bytes.Clone(schemaJSON) }

// Manifest describes a model.
type Manifest struct {
	Version int     `json:"version"`
	Tables  []Table `json:"tables"`
}

// Table is a model entity mapped to a database table.
type Table struct {
	Schema       string            `json:"schema,omitempty"` // empty: the database's only schema
	Name         string            `json:"name"`
	Source       *finding.Position `json:"source,omitempty"`
	Columns      []Column          `json:"columns"`
	PrimaryKey   []string          `json:"primary_key,omitempty"`
	JoinTableFor []TableRef        `json:"join_table_for,omitempty"`
	Uniques      []Unique          `json:"uniques,omitempty"`
	Relations    []Relation        `json:"relations,omitempty"`
}

// Unique is a uniqueness the model promises over one or more columns. Single
// columns are usually declared with Column.Unique; this form carries
// composite uniqueness.
type Unique struct {
	Columns []string `json:"columns"`
}

// Relation is a model relation stored as a foreign key: Columns of the table
// reference References. Many-to-many relations are declared with
// Table.JoinTableFor on the join table instead.
type Relation struct {
	Columns    []string `json:"columns"`
	References TableRef `json:"references"`
	// RefColumns are the referenced columns, in the order of Columns. Empty
	// means "the referenced table's key", and the column match is not checked.
	RefColumns []string `json:"ref_columns,omitempty"`
}

// TableRef names a table, for relations between tables.
type TableRef struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table"`
}

// Column is a model field mapped to a database column.
type Column struct {
	Name     string            `json:"name"`
	Required bool              `json:"required,omitempty"`
	Unique   bool              `json:"unique,omitempty"`     // the field alone is unique
	MaxLen   int               `json:"max_length,omitempty"` // maximum string length in characters
	Enum     []string          `json:"enum,omitempty"`       // the only values the field accepts
	Source   *finding.Position `json:"source,omitempty"`
}

// Parse decodes and validates a manifest document: the version is checked
// first so an incompatible document gets a clear error naming both versions,
// then the document is validated against the JSON Schema.
func Parse(data []byte) (*Manifest, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("manifest is not valid JSON: %w", err)
	}
	if obj, ok := doc.(map[string]any); ok {
		if v, ok := obj["version"].(float64); ok && v == float64(int64(v)) && int64(v) != Version {
			return nil, fmt.Errorf("manifest version %d is not supported; this schemalens supports version %d", int64(v), Version)
		}
	}
	if err := validate(data); err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	return &m, nil
}

func validate(data []byte) error {
	c := jsonschema.NewCompiler()
	var raw any
	if err := json.Unmarshal(schemaJSON, &raw); err != nil {
		return fmt.Errorf("embedded manifest schema: %w", err)
	}
	const url = "schemalens://manifest.schema.json"
	if err := c.AddResource(url, raw); err != nil {
		return fmt.Errorf("embedded manifest schema: %w", err)
	}
	sch, err := c.Compile(url)
	if err != nil {
		return fmt.Errorf("embedded manifest schema: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("manifest is not valid JSON: %w", err)
	}
	err = sch.Validate(inst)
	if err == nil {
		return nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return err
	}
	var problems []string
	seen := map[string]bool{}
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) > 0 {
			for _, c := range e.Causes {
				walk(c)
			}
			return
		}
		loc := "/" + strings.Join(e.InstanceLocation, "/")
		p := fmt.Sprintf("at %s: %s", loc, e.ErrorKind.LocalizedString(printer))
		if !seen[p] {
			seen[p] = true
			problems = append(problems, p)
		}
	}
	walk(ve)
	return fmt.Errorf("manifest does not match the JSON Schema:\n  %s", strings.Join(problems, "\n  "))
}
