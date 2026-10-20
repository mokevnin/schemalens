package entadapter

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

// loadManifest loads the ent schema package at schemaPath and maps its graph
// to a manifest. Tables and columns come from the same graph that ent's own
// migration uses (gen.Graph.Tables), so physical names cannot drift from the
// database ent produces.
func loadManifest(schemaPath string) (*manifest, error) {
	// entc treats a bare "a/b" as an import path; the argument is a directory.
	pattern := filepath.ToSlash(schemaPath)
	if !filepath.IsAbs(schemaPath) && !strings.HasPrefix(pattern, "./") && !strings.HasPrefix(pattern, "../") {
		pattern = "./" + pattern
	}
	g, err := entc.LoadGraph(pattern, &gen.Config{})
	if err != nil {
		return nil, fmt.Errorf("cannot load ent schema %s: %w", schemaPath, err)
	}
	tables, err := g.Tables()
	if err != nil {
		return nil, fmt.Errorf("cannot derive tables from ent schema %s: %w", schemaPath, err)
	}
	src, err := newSourceIndex(schemaPath)
	if err != nil {
		return nil, err
	}

	colPos := columnPositions(g, src)
	joins := joinTables(g)
	m := &manifest{Version: ManifestVersion, Tables: []table{}}
	for _, t := range tables {
		m.Tables = append(m.Tables, mapTable(t, colPos[t.Name], src, joins[t.Name]))
	}
	return m, nil
}

func mapTable(t *schema.Table, pos map[string]*position, src *sourceIndex, isJoin bool) table {
	out := table{
		Schema:  t.Schema,
		Name:    t.Name,
		Source:  src.parsePos(t.Pos),
		Columns: []column{},
	}
	for _, c := range t.Columns {
		col := column{Name: c.Name, Required: !c.Nullable, Unique: c.Unique, Enum: c.Enums, Source: pos[c.Name]}
		if c.Type == field.TypeString && c.Size > 0 && c.Size < math.MaxInt32 { // field.Text carries MaxInt32 as "no limit"
			col.MaxLen = int(c.Size)
		}
		out.Columns = append(out.Columns, col)
	}
	for _, idx := range t.Indexes {
		// A partial unique index (entsql.IndexWhere) promises uniqueness only among the rows its
		// predicate selects, so it is not the unconditional claim the manifest's `uniques` makes.
		if idx.Unique && (idx.Annotation == nil || idx.Annotation.Where == "") {
			out.Uniques = append(out.Uniques, unique{Columns: columnNames(idx.Columns)})
		}
	}
	for _, c := range t.PrimaryKey {
		out.PrimaryKey = append(out.PrimaryKey, c.Name)
	}
	if isJoin && len(t.ForeignKeys) == 2 {
		for _, fk := range t.ForeignKeys {
			out.JoinTableFor = append(out.JoinTableFor, tableRef{Schema: fk.RefTable.Schema, Table: fk.RefTable.Name})
		}
	}
	if !isJoin {
		for _, fk := range t.ForeignKeys {
			out.Relations = append(out.Relations, relation{
				Columns:    columnNames(fk.Columns),
				References: tableRef{Schema: fk.RefTable.Schema, Table: fk.RefTable.Name},
				RefColumns: columnNames(fk.RefColumns),
			})
		}
	}
	return out
}

func columnNames(cs []*schema.Column) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

// joinTables names the tables ent builds for many-to-many edges. Edge schemas
// (edge.Through) are ordinary entities and are not join tables.
func joinTables(g *gen.Graph) map[string]bool {
	out := map[string]bool{}
	for _, n := range g.Nodes {
		for _, e := range n.Edges {
			if e.Rel.Type == gen.M2M && e.Through == nil && (e.Ref == nil || e.Ref.Through == nil) {
				out[e.Rel.Table] = true
			}
		}
	}
	return out
}

// columnPositions maps table -> column -> source position: the ent field for
// field columns (including edge fields), the edge for foreign key columns the
// edge owns.
func columnPositions(g *gen.Graph, src *sourceIndex) map[string]map[string]*position {
	out := map[string]map[string]*position{}
	set := func(table, column string, p *position) {
		if p == nil {
			return
		}
		if out[table] == nil {
			out[table] = map[string]*position{}
		}
		if out[table][column] == nil {
			out[table][column] = p
		}
	}
	for _, n := range g.Nodes {
		for _, f := range n.Fields {
			if !f.UserDefined {
				continue
			}
			set(n.Table(), f.StorageKey(), src.fieldPos(n.Name, f.Name, f.Position))
		}
	}
	// Foreign keys the edge owns, unless an explicit edge field already named them.
	for _, n := range g.Nodes {
		for _, e := range n.Edges {
			if e.Rel.Table == "" || len(e.Rel.Columns) == 0 {
				continue
			}
			if e.Rel.Type == gen.M2M {
				continue
			}
			set(e.Rel.Table, e.Rel.Column(), src.edgePos(n.Name, e.Name))
		}
	}
	return out
}

func (s *sourceIndex) parsePos(p string) *position {
	i := strings.LastIndex(p, ":")
	if i < 0 {
		return nil
	}
	line, err := strconv.Atoi(p[i+1:])
	if err != nil || line < 1 {
		return nil
	}
	return &position{File: s.rel(p[:i]), Line: line}
}

func (s *sourceIndex) rel(file string) string {
	if filepath.IsAbs(file) && s.cwd != "" {
		if r, err := filepath.Rel(s.cwd, file); err == nil && !strings.HasPrefix(r, "..") {
			return filepath.ToSlash(r)
		}
	}
	return filepath.ToSlash(file)
}
