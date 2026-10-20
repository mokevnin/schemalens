// Package bunext is schemalens's extended mode for bun: a library a project
// calls from a small program of its own to build the manifest from real Go
// types.
//
//	func main() {
//		m, err := bunext.Manifest(db.Dialect(), &User{}, &Order{})
//		if err != nil { log.Fatal(err) }
//		_ = m.Write(os.Stdout)
//	}
//
// bun itself reads the models (its schema package), so table and column names
// are exactly what CreateTable would use, including run-time naming (a custom
// schema.SetTableNameInflector, table: tags, schema-qualified names). The price
// is that the program must be written and compiled with the project; the
// manifest carries no source positions. The program can be used as an adapter
// (schemalens check --adapter name) or its output saved and passed to
// --manifest.
//
// Relations: only those bun turns into foreign keys (belongs-to and has-one
// whose key is a plain column; CreateTable().WithForeignKeys()) are emitted. A
// model used as a many-to-many junction table is declared with join_table_for.
package bunext

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"

	"github.com/uptrace/bun/schema"

	"github.com/mokevnin/schemalens/manifest"
)

// Manifest builds the manifest of the given models (struct values or pointers,
// as passed to CreateTable) under the dialect, normally db.Dialect(). The
// dialect carries the table registry and the default schema.
func Manifest(dialect schema.Dialect, models ...any) (m *manifest.Manifest, err error) {
	defer func() {
		// bun panics on a model it cannot use.
		if r := recover(); r != nil {
			m, err = nil, fmt.Errorf("bunext: %v", r)
		}
	}()
	var tables []*schema.Table
	for _, model := range models {
		typ := reflect.TypeOf(model)
		for typ != nil && typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ == nil || typ.Kind() != reflect.Struct {
			return nil, fmt.Errorf("bunext: model %T is not a struct", model)
		}
		tables = append(tables, dialect.Tables().Get(typ))
	}

	byName := map[string]bool{}
	for _, t := range tables {
		byName[qualified(t)] = true
	}
	out := manifest.New()
	seen := map[string]bool{}
	for _, t := range tables {
		if seen[qualified(t)] {
			continue
		}
		seen[qualified(t)] = true
		out.Tables = append(out.Tables, buildTable(dialect, t, byName))
	}
	return out, nil
}

func qualified(t *schema.Table) string { return t.Schema + "." + t.Name }

func ref(d schema.Dialect, t *schema.Table) manifest.TableRef {
	r := manifest.TableRef{Table: t.Name}
	if t.Schema != d.DefaultSchema() {
		r.Schema = t.Schema
	}
	return r
}

var varcharLen = regexp.MustCompile(`(?i)^\s*(?:n?var)?char\s*\(\s*(\d+)\s*\)`)

func buildTable(d schema.Dialect, t *schema.Table, known map[string]bool) manifest.Table {
	r := ref(d, t)
	mt := manifest.Table{Schema: r.Schema, Name: r.Table, Columns: []manifest.Column{}}

	single := map[string]bool{}
	for _, name := range sortedKeys(t.Unique) {
		fields := t.Unique[name]
		if name == "" || len(fields) == 1 {
			for _, f := range fields {
				single[f.Name] = true
			}
			continue
		}
		cols := make([]string, 0, len(fields))
		for _, f := range fields {
			cols = append(cols, f.Name)
		}
		mt.Uniques = append(mt.Uniques, manifest.Unique{Columns: cols})
	}

	for _, f := range t.Fields {
		c := manifest.Column{Name: f.Name, Required: f.NotNull || f.IsPK, Unique: single[f.Name]}
		if mm := varcharLen.FindStringSubmatch(f.UserSQLType); mm != nil {
			c.MaxLen, _ = strconv.Atoi(mm[1])
		}
		mt.Columns = append(mt.Columns, c)
	}
	for _, f := range t.PKs {
		mt.PrimaryKey = append(mt.PrimaryKey, f.Name)
	}

	var joined []manifest.TableRef
	for _, name := range sortedKeys(t.Relations) {
		rel := t.Relations[name]
		if rel.Type != schema.BelongsToRelation && rel.Type != schema.HasOneRelation {
			continue
		}
		if !rel.References() || !known[qualified(rel.JoinTable)] {
			if t.IsM2MTable && rel.Type == schema.BelongsToRelation {
				joined = append(joined, ref(d, rel.JoinTable))
			}
			continue
		}
		cols := make([]string, 0, len(rel.BasePKs))
		for _, f := range rel.BasePKs {
			cols = append(cols, f.Name)
		}
		refCols := make([]string, 0, len(rel.JoinPKs))
		for _, f := range rel.JoinPKs {
			refCols = append(refCols, f.Name)
		}
		mt.Relations = append(mt.Relations, manifest.Relation{Columns: cols, References: ref(d, rel.JoinTable), RefColumns: refCols})
		if t.IsM2MTable && rel.Type == schema.BelongsToRelation {
			joined = append(joined, ref(d, rel.JoinTable))
		}
	}
	if len(joined) == 2 {
		mt.JoinTableFor = joined
	}
	return mt
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
