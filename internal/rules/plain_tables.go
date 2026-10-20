package rules

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
)

// PlainTables is the id of the rule below.
const PlainTables = "plain-tables"

func init() {
	register(Rule{
		ID:         PlainTables,
		Dialects:   dialect.All,
		DefaultOff: true,
		Explain: Explanation{
			Why:       "Functions, triggers and views are logic and structure that live in the database but not in the model, so ORMs and migration diffs cannot see them and they change behavior behind the application's back.",
			Fix:       "Move the logic into application code, or keep the object and exclude it with a written reason. Options under rules.plain-tables.options: exclude (exact object names), exclude_prefixes (name prefixes, for objects a migration tool creates) and exclude_extensions (true skips PostgreSQL objects owned by an extension).",
			Exception: "Objects created by an extension or by the migration tool itself, and deliberate database-side logic such as an audit trigger, when the reason is written down.",
		},
		CheckDB: checkPlainTables,
	})
}

// catalogObject is one function, trigger or view read from a catalog.
type catalogObject struct {
	kind  string // "function", "trigger" or "view"
	table string // owning table, triggers only
	name  string
}

func checkPlainTables(ctx context.Context, in Input) ([]finding.Finding, error) {
	opts := parsePlainTablesOptions(in.Options)
	var (
		schemaName string
		objs       []catalogObject
		err        error
	)
	switch in.Dialect {
	case dialect.SQLite:
		schemaName = "main"
		objs, err = sqliteObjects(ctx, in.DB)
	case dialect.Postgres:
		schemaName, objs, err = postgresObjects(ctx, in.DB, opts.excludeExtensions)
	case dialect.MySQL:
		schemaName, objs, err = mysqlObjects(ctx, in.DB)
	}
	if err != nil {
		return nil, fmt.Errorf("read %s catalog: %w", in.Dialect, err)
	}

	var out []finding.Finding
	for _, o := range objs {
		if opts.excluded(o.name) {
			continue
		}
		obj := finding.Object{Schema: schemaName}
		var msg string
		switch o.kind {
		case "function":
			obj.Function = o.name
			msg = fmt.Sprintf("function %q exists in schema %q but is not part of the model", o.name, schemaName)
		case "view":
			obj.View = o.name
			msg = fmt.Sprintf("view %q exists in schema %q but is not part of the model", o.name, schemaName)
		case "trigger":
			obj.Table, obj.Trigger = o.table, o.name
			msg = fmt.Sprintf("trigger %q on table %q exists but is not part of the model", o.name, o.table)
		}
		out = append(out, finding.Finding{
			Rule:    PlainTables,
			Object:  obj,
			Message: msg,
			Fix: fmt.Sprintf("Move the logic of %s %q into application code and drop it, or exclude it in schemalens.yml with a reason (rules.plain-tables.options for extension and migration-tool objects).",
				o.kind, o.name),
		})
	}
	return out, nil
}

type plainTablesOptions struct {
	exclude           map[string]bool
	prefixes          []string
	excludeExtensions bool
}

func (o plainTablesOptions) excluded(name string) bool {
	if o.exclude[name] {
		return true
	}
	for _, p := range o.prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func parsePlainTablesOptions(m map[string]any) plainTablesOptions {
	o := plainTablesOptions{exclude: map[string]bool{}}
	for _, n := range stringList(m["exclude"]) {
		o.exclude[n] = true
	}
	o.prefixes = stringList(m["exclude_prefixes"])
	o.excludeExtensions, _ = m["exclude_extensions"].(bool)
	return o
}

func sqliteObjects(ctx context.Context, db *sql.DB) ([]catalogObject, error) {
	// SQLite has no stored functions; only views and triggers live in the catalog.
	// Internal sqlite_ objects are never views or triggers.
	rows, err := db.QueryContext(ctx, `SELECT type, name, tbl_name FROM sqlite_master WHERE type IN ('view', 'trigger') ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []catalogObject
	for rows.Next() {
		var o catalogObject
		var table string
		if err := rows.Scan(&o.kind, &o.name, &table); err != nil {
			return nil, err
		}
		if o.kind == "trigger" {
			o.table = table
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func postgresObjects(ctx context.Context, db *sql.DB, skipExtensions bool) (string, []catalogObject, error) {
	var schemaName string
	if err := db.QueryRowContext(ctx, `SELECT current_schema()`).Scan(&schemaName); err != nil {
		return "", nil, err
	}
	ext := func(class string) string {
		if !skipExtensions {
			return ""
		}
		return ` AND NOT EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid = '` + class + `'::regclass AND d.objid = o.oid AND d.deptype = 'e')`
	}
	queries := []struct{ kind, query string }{
		{"function", `SELECT DISTINCT o.proname, '' FROM pg_proc o JOIN pg_namespace n ON n.oid = o.pronamespace
			WHERE n.nspname = $1 AND o.prokind IN ('f', 'p')` + ext("pg_proc")},
		{"view", `SELECT o.relname, '' FROM pg_class o JOIN pg_namespace n ON n.oid = o.relnamespace
			WHERE n.nspname = $1 AND o.relkind IN ('v', 'm')` + ext("pg_class")},
		// Internal triggers back foreign keys and constraints; they are not user logic.
		{"trigger", `SELECT o.tgname, c.relname FROM pg_trigger o JOIN pg_class c ON c.oid = o.tgrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = $1 AND NOT o.tgisinternal`},
	}
	var out []catalogObject
	for _, q := range queries {
		objs, err := queryObjects(ctx, db, q.kind, q.query, schemaName)
		if err != nil {
			return "", nil, err
		}
		out = append(out, objs...)
	}
	return schemaName, out, nil
}

func mysqlObjects(ctx context.Context, db *sql.DB) (string, []catalogObject, error) {
	var schemaName string
	if err := db.QueryRowContext(ctx, `SELECT DATABASE()`).Scan(&schemaName); err != nil {
		return "", nil, err
	}
	queries := []struct{ kind, query string }{
		{"function", `SELECT ROUTINE_NAME, '' FROM information_schema.routines WHERE ROUTINE_SCHEMA = ?`},
		{"view", `SELECT TABLE_NAME, '' FROM information_schema.views WHERE TABLE_SCHEMA = ?`},
		{"trigger", `SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE FROM information_schema.triggers WHERE TRIGGER_SCHEMA = ?`},
	}
	var out []catalogObject
	for _, q := range queries {
		objs, err := queryObjects(ctx, db, q.kind, q.query, schemaName)
		if err != nil {
			return "", nil, err
		}
		out = append(out, objs...)
	}
	return schemaName, out, nil
}

// queryObjects runs a query returning (name, table) rows for one object kind.
func queryObjects(ctx context.Context, db *sql.DB, kind, query, schemaName string) ([]catalogObject, error) {
	rows, err := db.QueryContext(ctx, query, schemaName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []catalogObject
	for rows.Next() {
		o := catalogObject{kind: kind}
		var table string
		if err := rows.Scan(&o.name, &table); err != nil {
			return nil, err
		}
		if kind == "trigger" {
			o.table = table
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
