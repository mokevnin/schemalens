package rules_test

import (
	"testing"

	"ariga.io/atlas/sql/schema"

	"github.com/mokevnin/schemalens/internal/dialect"
	"github.com/mokevnin/schemalens/internal/rules"
)

// realmWithUnindexedFK has a foreign key with no index on its columns.
func realmWithUnindexedFK() *schema.Realm {
	id := schema.NewIntColumn("id", "bigint")
	parentID := schema.NewIntColumn("parent_id", "bigint")
	parent := schema.NewTable("parent").AddColumns(id).SetPrimaryKey(schema.NewPrimaryKey(id))
	child := schema.NewTable("child").AddColumns(schema.NewIntColumn("id", "bigint"), parentID)
	child.SetPrimaryKey(schema.NewPrimaryKey(child.Columns[0]))
	child.AddForeignKeys(schema.NewForeignKey("child_parent_fk").
		AddColumns(parentID).SetRefTable(parent).AddRefColumns(id))
	return schema.NewRealm(schema.New("db").AddTables(parent, child))
}

// A rule that does not apply to a dialect must be reported as skipped, so a
// clean result for that dialect is never mistaken for "checked and fine".
func TestRuleNotApplicableToDialectIsSkippedNotClean(t *testing.T) {
	in := rules.Input{Dialect: dialect.MySQL, Realm: realmWithUnindexedFK()}

	if got := rules.Run(in); len(got) != 0 {
		t.Errorf("findings on mysql = %v, want none", got)
	}
	skipped := rules.Skipped(dialect.MySQL)
	found := false
	for _, r := range skipped {
		if r.ID == rules.FKWithoutIndex {
			found = true
		}
	}
	if !found {
		t.Errorf("%s not reported as skipped on mysql: %v", rules.FKWithoutIndex, skipped)
	}

	in.Dialect = dialect.Postgres
	if got := rules.Run(in); len(got) != 1 {
		t.Errorf("findings on postgres = %v, want one", got)
	}
	for _, r := range rules.Skipped(dialect.Postgres) {
		if r.ID == rules.FKWithoutIndex {
			t.Errorf("%s reported as skipped on postgres", rules.FKWithoutIndex)
		}
	}
}
