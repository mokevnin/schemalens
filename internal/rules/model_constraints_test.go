package rules_test

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/manifest"
	"github.com/mokevnin/schemalens/internal/rules"
)

// Engines render stored CHECK constraints differently from how they were
// written. These renderings cannot all be produced through `check` on one
// engine, so they are driven through Input.
func TestRenderedChecksCountAsEnforcement(t *testing.T) {
	exprs := map[string]string{
		"postgres any-array": `((status)::text = ANY ((ARRAY['draft'::character varying, 'published'::character varying])::text[]))`,
		"mysql charset":      "(`status` in (_utf8mb4'draft',_utf8mb4'published'))",
		"or chain":           `status = 'draft' OR status = 'published'`,
		"null escape":        `status IS NULL OR status IN ('draft', 'published')`,
	}
	for name, expr := range exprs {
		t.Run(name, func(t *testing.T) {
			tbl := schema.NewTable("posts").
				AddColumns(schema.NewColumn("status").SetType(&schema.StringType{T: "text"})).
				AddAttrs(&schema.Check{Expr: expr})
			realm := schema.NewRealm(schema.New("main").AddTables(tbl))
			m := &manifest.Manifest{Version: 1, Tables: []manifest.Table{{Name: "posts",
				Columns: []manifest.Column{{Name: "status", Enum: []string{"published", "draft"}}}}}}
			for _, f := range only(rules.EnumNotEnforced, rules.Run(rules.Input{Dialect: dialect.SQLite, Realm: realm, Manifest: m})) {
				t.Errorf("unexpected finding: %s", f.Message)
			}
		})
	}
}

func TestLengthChecksCountAsEnforcement(t *testing.T) {
	exprs := map[string]bool{
		`length(name) <= 20`:                             true,
		`((char_length((name)::text) <= 20))`:            true,
		"(char_length(`name`) < 21)":                     true,
		`length(name) <= 20 AND name <> ''`:              true,
		`length(name) <= 50`:                             false,
		`length(name) <= 20 OR name = 'a very long one'`: false,
	}
	for expr, enforced := range exprs {
		t.Run(expr, func(t *testing.T) {
			tbl := schema.NewTable("users").
				AddColumns(schema.NewColumn("name").SetType(&schema.StringType{T: "text"})).
				AddAttrs(&schema.Check{Expr: expr})
			realm := schema.NewRealm(schema.New("main").AddTables(tbl))
			m := &manifest.Manifest{Version: 1, Tables: []manifest.Table{{Name: "users",
				Columns: []manifest.Column{{Name: "name", MaxLen: 20}}}}}
			got := only(rules.MaxLengthNotEnforced, rules.Run(rules.Input{Dialect: dialect.Postgres, Realm: realm, Manifest: m}))
			if enforced && len(got) != 0 || !enforced && len(got) != 1 {
				t.Errorf("findings = %+v, enforced = %v", got, enforced)
			}
		})
	}
}

func only(rule string, fs []finding.Finding) []finding.Finding {
	var out []finding.Finding
	for _, f := range fs {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}
