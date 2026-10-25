package cli_test

import (
	"path/filepath"
	"testing"
)

func TestCheckTextFormat(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE audit_log (message TEXT);")
	stdout, _, code := run("check", "--dsn", "sqlite://"+dbPath, "--format", "text")

	want := `table-without-primary-key  main.audit_log
  table "audit_log" has no primary key
  fix: Declare a primary key for the model that maps to table "audit_log" (for example an id field marked as the primary key), then regenerate the schema.

1 finding, 0 stale exclusions
`
	if code != 1 || stdout != want {
		t.Errorf("code=%d stdout:\n%s", code, stdout)
	}
}

func TestCheckTextFormatClean(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE t (id INTEGER PRIMARY KEY);")
	stdout, _, code := run("check", "--dsn", "sqlite://"+dbPath, "--format", "text")
	if code != 0 || stdout != "no findings\n" {
		t.Errorf("code=%d stdout=%q", code, stdout)
	}
}
