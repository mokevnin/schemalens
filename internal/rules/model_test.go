package rules_test

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/finding"
	"github.com/mokevnin/schemalens/internal/manifest"
	"github.com/mokevnin/schemalens/internal/rules"
)

// SQLite exposes a single schema, so the same-name-in-two-schemas case cannot
// go through `schemalens check` yet; it is driven through Input instead.
func TestSameNamedTablesInDifferentSchemasAreNotConfused(t *testing.T) {
	users := func(null bool) *schema.Table {
		return schema.NewTable("users").AddColumns(
			schema.NewColumn("email").SetType(&schema.StringType{T: "text"}).SetNull(null))
	}
	realm := schema.NewRealm(
		schema.New("public").AddTables(users(false)),
		schema.New("billing").AddTables(users(true)),
	)
	m := &manifest.Manifest{Version: 1, Tables: []manifest.Table{
		{Schema: "public", Name: "users", Columns: []manifest.Column{{Name: "email", Required: true}}},
		{Schema: "billing", Name: "users", Columns: []manifest.Column{{Name: "email"}}},
	}}
	got := run(rules.Input{Dialect: dialect.SQLite, Realm: realm, Manifest: m})
	if len(got) != 0 {
		t.Fatalf("findings = %+v, want none", got)
	}

	m.Tables[1].Columns[0].Required = true
	got = run(rules.Input{Dialect: dialect.SQLite, Realm: realm, Manifest: m})
	if len(got) != 1 || got[0].Object.Schema != "billing" || got[0].Object.Table != "users" {
		t.Fatalf("findings = %+v, want exactly one for billing.users", got)
	}
}

func run(in rules.Input) []finding.Finding {
	var out []finding.Finding
	for _, f := range rules.Run(in) {
		if f.Rule == rules.RequiredFieldNullableColumn {
			out = append(out, f)
		}
	}
	return out
}
