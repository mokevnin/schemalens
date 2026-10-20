package main

import (
	"fmt"
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/mokevnin/schemalens/adapters/staticmodel"
)

const (
	gormModel = "gorm.io/gorm.Model"
	timeTime  = "time.Time"
)

// analyze turns the project's gorm structs into a manifest.
func analyze(prog *staticmodel.Program, cfg *config) (*staticmodel.Manifest, error) {
	embeddable := embeddedOnly(prog)
	m := &staticmodel.Manifest{Version: 1, Tables: []staticmodel.Table{}}
	var built []*built
	for _, s := range prog.Structs() {
		if slices.Contains(cfg.Exclude, s.Name) || slices.Contains(cfg.Exclude, s.PkgPath+"."+s.Name) {
			continue
		}
		if !isModel(s, cfg, embeddable) {
			continue
		}
		b, err := buildTable(s, cfg)
		if err != nil {
			return nil, err
		}
		built = append(built, b)
	}
	addRelations(built)
	for _, b := range built {
		m.Tables = append(m.Tables, b.table)
	}
	joinTables, joins := addJoinTables(built, cfg)
	m.Tables = append(m.Tables, joinTables...)
	cfg.joins = joins
	return m, nil
}

// embeddedOnly collects structs that are embedded into another struct;
// unless they say otherwise they are fragments, not tables.
func embeddedOnly(prog *staticmodel.Program) map[string]bool {
	out := map[string]bool{}
	for _, s := range prog.Structs() {
		for _, f := range s.Fields {
			if isEmbedding(f) {
				out[staticmodel.NamedType(f.Type)] = true
			}
		}
	}
	return out
}

// isModel decides whether a struct maps to a table: it names itself so (gorm.Model,
// a TableName method, a config entry), or it carries gorm tags and is not
// merely embedded elsewhere.
func isModel(s *staticmodel.Struct, cfg *config, embeddable map[string]bool) bool {
	if _, ok := lookupTable(cfg, s); ok || s.HasMethod("TableName") {
		return true
	}
	hasTag := false
	for _, f := range s.Fields {
		if f.Embedded && staticmodel.NamedType(f.Type) == gormModel {
			return true
		}
		if f.Tag.Get("gorm") != "" {
			hasTag = true
		}
	}
	return hasTag && !embeddable[s.PkgPath+"."+s.Name]
}

func lookupTable(cfg *config, s *staticmodel.Struct) (string, bool) {
	if v, ok := cfg.Tables[s.PkgPath+"."+s.Name]; ok {
		return v, true
	}
	v, ok := cfg.Tables[s.Name]
	return v, ok
}

func buildTable(s *staticmodel.Struct, cfg *config) (*built, error) {
	namer := cfg.namer()
	name, err := tableName(s, cfg)
	if err != nil {
		return nil, err
	}
	b := &built{strukt: s, fieldCols: map[string]string{}}
	t := &b.table
	*t = staticmodel.Table{Name: name, Source: staticmodel.ToSource(s.Pos), Columns: []staticmodel.Column{}}
	t.SplitSchema(name)

	type pkCandidate struct {
		col      string
		explicit bool
	}
	var pks []pkCandidate
	groups := map[string][]string{} // named unique indexes: name -> columns
	var groupOrder []string
	depth := map[string]int{}
	for _, f := range s.Flatten(isEmbedding) {
		if !f.Exported {
			continue
		}
		tag := parseTag(f.Tag.Get("gorm"))
		if tag.ignoredByMigration() {
			continue
		}
		if isRelation(f, tag) {
			b.relFields = append(b.relFields, f)
			continue
		}
		col := f.Name
		if v, ok := tag["COLUMN"]; ok && v != "" {
			col = v
		} else {
			col = namer.ColumnName("", f.Name)
		}
		col = embedPrefix(f) + col

		// A field declared directly on the struct wins over an embedded one
		// with the same column, as in gorm.
		if d, seen := depth[col]; seen {
			if len(f.Path) >= d {
				continue
			}
			t.Columns = slices.DeleteFunc(t.Columns, func(c staticmodel.Column) bool { return c.Name == col })
			pks = slices.DeleteFunc(pks, func(p pkCandidate) bool { return p.col == col })
			for name, cols := range groups {
				groups[name] = slices.DeleteFunc(cols, func(c string) bool { return c == col })
			}
		}
		depth[col] = len(f.Path)

		explicitPK := tag.isSet("PRIMARYKEY") || tag.isSet("PRIMARY_KEY")
		c := staticmodel.Column{
			Name:     col,
			Required: tag.isSet("NOT NULL") || tag.isSet("NOTNULL") || explicitPK,
			Unique:   tag.isSet("UNIQUE"),
			Source:   staticmodel.ToSource(f.Pos),
		}
		if staticmodel.IsString(f.Type) {
			if n, err := strconv.Atoi(tag["SIZE"]); err == nil && n > 0 {
				c.MaxLen = n
			}
		}
		if v, ok := tag["UNIQUEINDEX"]; ok {
			if name, _, _ := strings.Cut(v, ","); name == "" {
				c.Unique = true
			} else {
				if _, seen := groups[name]; !seen {
					groupOrder = append(groupOrder, name)
				}
				groups[name] = append(groups[name], col)
			}
		}
		t.Columns = append(t.Columns, c)
		b.fieldCols[f.Name] = col
		if explicitPK || col == "id" {
			pks = append(pks, pkCandidate{col: col, explicit: explicitPK})
		}
	}
	explicit := slices.ContainsFunc(pks, func(p pkCandidate) bool { return p.explicit })
	for _, p := range pks {
		if p.explicit || !explicit {
			t.PrimaryKey = append(t.PrimaryKey, p.col)
		}
	}
	for _, name := range groupOrder {
		switch cols := groups[name]; len(cols) {
		case 0:
		case 1:
			for i := range t.Columns {
				if t.Columns[i].Name == cols[0] {
					t.Columns[i].Unique = true
				}
			}
		default:
			t.Uniques = append(t.Uniques, staticmodel.Unique{Columns: cols})
		}
	}
	// An implicit "id" key is Required too.
	if !explicit {
		for i := range t.Columns {
			if slices.Contains(t.PrimaryKey, t.Columns[i].Name) {
				t.Columns[i].Required = true
			}
		}
	}
	return b, nil
}

func tableName(s *staticmodel.Struct, cfg *config) (string, error) {
	if v, ok := lookupTable(cfg, s); ok {
		return v, nil
	}
	if v, ok := s.StringMethod("TableName"); ok {
		return v, nil
	}
	if s.HasMethod("TableName") {
		return "", fmt.Errorf("%s:%d: the table name of %s is computed by its TableName method, which cannot be read statically; add it under \"tables\" in %s",
			s.Pos.File, s.Pos.Line, s.Name, defaultConfigFile)
	}
	n := cfg.namer()
	return n.TableName(s.Name), nil
}

// isEmbedding reports whether gorm flattens the field into the parent table:
// an anonymous struct field, or a struct field tagged "embedded".
func isEmbedding(f staticmodel.Field) bool {
	st, ok := f.Type.Underlying().(*types.Struct)
	if p, isPtr := f.Type.(*types.Pointer); isPtr {
		st, ok = p.Elem().Underlying().(*types.Struct)
	}
	if !ok || st == nil || isScalarStruct(f.Type, parseTag(f.Tag.Get("gorm"))) {
		return false
	}
	return f.Embedded || parseTag(f.Tag.Get("gorm")).isSet("EMBEDDED")
}

// embedPrefix is the column prefix contributed by the embedding chain.
func embedPrefix(f staticmodel.Field) string {
	var b strings.Builder
	for _, p := range f.Path {
		b.WriteString(parseTag(p.Tag.Get("gorm"))["EMBEDDEDPREFIX"])
	}
	return b.String()
}

// isRelation reports whether the field is an association (a struct or a
// slice of structs gorm joins on foreign keys) rather than a column.
func isRelation(f staticmodel.Field, tag tagSettings) bool {
	t := staticmodel.Deref(f.Type)
	if sl, ok := t.Underlying().(*types.Slice); ok {
		t = staticmodel.Deref(sl.Elem())
	}
	switch t.Underlying().(type) {
	case *types.Struct:
		return !isScalarStruct(t, tag)
	case *types.Chan, *types.Signature:
		return true
	}
	return false
}

// isScalarStruct reports whether a struct type is stored in one column:
// time.Time, anything implementing sql.Scanner or driver.Valuer, or a field
// with an explicit serializer or database type.
func isScalarStruct(t types.Type, tag tagSettings) bool {
	if staticmodel.NamedType(t) == timeTime {
		return true
	}
	if tag.isSet("SERIALIZER") || tag.isSet("TYPE") || tag.isSet("DATATYPE") {
		return true
	}
	return staticmodel.HasMethod(staticmodel.Deref(t), "Scan") || staticmodel.HasMethod(staticmodel.Deref(t), "Value")
}

// tagSettings is a parsed `gorm:"..."` tag: upper-cased keys, raw values.
type tagSettings map[string]string

func (t tagSettings) isSet(key string) bool {
	v, ok := t[key]
	return ok && !strings.EqualFold(v, "false")
}

// ignoredByMigration covers `-`, `-:all` and `-:migration`.
func (t tagSettings) ignoredByMigration() bool {
	v, ok := t["-"]
	return ok && (v == "" || v == "all" || v == "migration")
}

// parseTag follows gorm's tag syntax: settings separated by ';', key and value
// by ':', a backslash escapes ';'.
func parseTag(tag string) tagSettings {
	out := tagSettings{}
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(tag); i++ {
		switch {
		case tag[i] == '\\' && i+1 < len(tag) && tag[i+1] == ';':
			cur.WriteByte(';')
			i++
		case tag[i] == ';':
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(tag[i])
		}
	}
	parts = append(parts, cur.String())
	for _, p := range parts {
		k, v, _ := strings.Cut(strings.TrimSpace(p), ":")
		k = strings.ToUpper(strings.TrimSpace(k))
		if k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

// built is a table under construction, with what relations need to find it.
type built struct {
	table     staticmodel.Table
	strukt    *staticmodel.Struct
	fieldCols map[string]string // Go field name -> column
	relFields []staticmodel.Field
}

// addRelations records the foreign keys gorm infers from association fields:
// belongs-to puts the key on the declaring table, has-one and has-many on the
// associated one. many2many has no key on either side; addJoinTables covers it.
func addRelations(all []*built) {
	byType := map[string]*built{}
	for _, b := range all {
		byType[b.strukt.PkgPath+"."+b.strukt.Name] = b
	}
	for _, owner := range all {
		for _, f := range owner.relFields {
			tag := parseTag(f.Tag.Get("gorm"))
			if tag.isSet("MANY2MANY") {
				continue
			}
			elem := staticmodel.Deref(f.Type)
			many := false
			if sl, ok := elem.Underlying().(*types.Slice); ok {
				elem, many = staticmodel.Deref(sl.Elem()), true
			}
			target := byType[staticmodel.NamedType(elem)]
			if target == nil {
				continue
			}
			fk := tag["FOREIGNKEY"]
			if !many {
				// belongs-to: the key is a field of the declaring struct.
				name := fk
				if name == "" {
					name = f.Name + "ID"
				}
				if col, ok := owner.fieldCols[name]; ok {
					owner.table.Relations = append(owner.table.Relations, staticmodel.Relation{Columns: []string{col}, References: target.table.Ref()})
					continue
				}
			}
			// has-one / has-many: the key is a field of the associated struct.
			if fk == "" {
				fk = owner.strukt.Name + "ID"
			}
			if col, ok := target.fieldCols[fk]; ok {
				target.table.Relations = append(target.table.Relations, staticmodel.Relation{Columns: []string{col}, References: owner.table.Ref()})
			}
		}
	}
}
