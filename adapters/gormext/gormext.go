// Package gormext is schemalens's extended mode for gorm: a library a project
// calls from a small program of its own to build the manifest from real Go
// types.
//
//	func main() {
//		m, err := gormext.Manifest(db.NamingStrategy, &User{}, &Order{})
//		if err != nil { log.Fatal(err) }
//		_ = m.Write(os.Stdout)
//	}
//
// Because gorm itself parses the models (schema.Parse) under the project's real
// naming strategy, table and column names are exactly what AutoMigrate would
// create, including run-time naming (prefixes, singular tables, custom
// Namer implementations, TableName methods computed at run time). The price is
// that the program must be written and compiled with the project; the manifest
// carries no source positions. The program can be used as an adapter
// (schemalens check --adapter name) or its output saved and passed to
// --manifest.
package gormext

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"gorm.io/gorm/schema"

	"github.com/mokevnin/schemalens/manifest"
)

// Manifest builds the manifest of the given models (struct values or pointers,
// as passed to AutoMigrate) with namer deciding physical names. A nil namer
// means gorm's default naming. Pass db.NamingStrategy, or the
// schema.NamingStrategy / schema.Namer used to open the connection.
func Manifest(namer schema.Namer, models ...any) (*manifest.Manifest, error) {
	if namer == nil {
		namer = schema.NamingStrategy{}
	}
	cache := &sync.Map{}
	var schemas []*schema.Schema
	for _, model := range models {
		s, err := schema.Parse(model, cache, namer)
		if err != nil {
			return nil, fmt.Errorf("gormext: cannot parse %T: %w", model, err)
		}
		schemas = append(schemas, s)
	}

	m := manifest.New()
	byTable := map[string]int{} // table name -> index in m.Tables
	for _, s := range schemas {
		if _, dup := byTable[s.Table]; dup {
			continue
		}
		t := buildTable(s)
		byTable[s.Table] = len(m.Tables)
		m.Tables = append(m.Tables, t)
	}
	addRelations(m, byTable, schemas)
	return m, nil
}

func ref(table string) manifest.TableRef {
	if schemaName, rest, ok := strings.Cut(table, "."); ok {
		return manifest.TableRef{Schema: schemaName, Table: rest}
	}
	return manifest.TableRef{Table: table}
}

func buildTable(s *schema.Schema) manifest.Table {
	r := ref(s.Table)
	t := manifest.Table{Schema: r.Schema, Name: r.Table, Columns: []manifest.Column{}}
	unique := map[string]bool{} // single-column unique indexes
	idx := s.ParseIndexes()
	for _, i := range idx {
		if i.Class != "UNIQUE" {
			continue
		}
		cols := make([]string, 0, len(i.Fields))
		for _, f := range i.Fields {
			cols = append(cols, f.DBName)
		}
		if len(cols) == 1 {
			unique[cols[0]] = true
		} else if len(cols) > 1 {
			t.Uniques = append(t.Uniques, manifest.Unique{Columns: cols})
		}
	}
	for _, f := range s.Fields {
		if f.DBName == "" || f.IgnoreMigration {
			continue
		}
		c := manifest.Column{
			Name:     f.DBName,
			Required: f.NotNull || f.PrimaryKey,
			Unique:   f.Unique || unique[f.DBName],
		}
		if f.Size > 0 && f.DataType == schema.String {
			c.MaxLen = f.Size
		}
		t.Columns = append(t.Columns, c)
	}
	for _, f := range s.PrimaryFields {
		if f.DBName != "" {
			t.PrimaryKey = append(t.PrimaryKey, f.DBName)
		}
	}
	return t
}

// addRelations records the foreign keys gorm derives from association fields
// (on whichever table owns the key) and the join tables of many2many ones.
func addRelations(m *manifest.Manifest, byTable map[string]int, schemas []*schema.Schema) {
	seen := map[string]bool{}
	for _, s := range schemas {
		for _, name := range sortedKeys(s.Relationships.Relations) {
			rel := s.Relationships.Relations[name]
			if rel.Type == schema.Many2Many {
				addJoinTable(m, byTable, rel, seen)
				continue
			}
			var cols []string
			var owner, target *schema.Schema
			for _, rf := range rel.References {
				if rf.PrimaryValue != "" {
					continue // polymorphic type discriminator, not a key
				}
				owner, target = rf.ForeignKey.Schema, rf.PrimaryKey.Schema
				cols = append(cols, rf.ForeignKey.DBName)
			}
			if owner == nil || target == nil {
				continue
			}
			oi, okO := byTable[owner.Table]
			_, okT := byTable[target.Table]
			if !okO || !okT {
				continue
			}
			key := owner.Table + "|" + strings.Join(cols, ",") + "|" + target.Table
			if seen[key] {
				continue
			}
			seen[key] = true
			m.Tables[oi].Relations = append(m.Tables[oi].Relations, manifest.Relation{Columns: cols, References: ref(target.Table)})
		}
	}
}

func addJoinTable(m *manifest.Manifest, byTable map[string]int, rel *schema.Relationship, seen map[string]bool) {
	js := rel.JoinTable
	if js == nil || seen["join|"+js.Table] {
		return
	}
	if _, exists := byTable[js.Table]; exists {
		return // the project models the join table itself
	}
	seen["join|"+js.Table] = true
	r := ref(js.Table)
	t := manifest.Table{Schema: r.Schema, Name: r.Table, Columns: []manifest.Column{}}
	for _, f := range js.Fields {
		if f.DBName == "" {
			continue
		}
		t.Columns = append(t.Columns, manifest.Column{Name: f.DBName, Required: f.NotNull || f.PrimaryKey})
	}
	for _, f := range js.PrimaryFields {
		t.PrimaryKey = append(t.PrimaryKey, f.DBName)
	}
	t.JoinTableFor = []manifest.TableRef{ref(rel.Schema.Table), ref(rel.FieldSchema.Table)}
	for _, rf := range rel.References {
		if rf.PrimaryKey == nil || rf.ForeignKey == nil {
			continue
		}
		t.Relations = append(t.Relations, manifest.Relation{
			Columns:    []string{rf.ForeignKey.DBName},
			References: ref(rf.PrimaryKey.Schema.Table),
		})
	}
	byTable[js.Table] = len(m.Tables)
	m.Tables = append(m.Tables, t)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
