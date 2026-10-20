package entadapter_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	_ "modernc.org/sqlite" // database/sql driver "sqlite"
)

const coreCommand = "../../cmd/schemalens"

// migrate creates the database ent's own migration produces for the sample
// project. tweak may change the tables first, to model a database that has
// drifted from the schema.
func migrate(t *testing.T, tweak func([]*schema.Table)) string {
	t.Helper()
	g, err := entc.LoadGraph("./"+sampleSchema, &gen.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tables, err := g.Tables()
	if err != nil {
		t.Fatal(err)
	}
	if tweak != nil {
		tweak(tables)
	}
	path := filepath.Join(t.TempDir(), "app.sqlite")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	m, err := schema.NewMigrate(entsql.OpenDB(dialect.SQLite, db))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Create(context.Background(), tables...); err != nil {
		t.Fatal(err)
	}
	return path
}

// build compiles a main package of the workspace into dir.
func build(t *testing.T, dir, name, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, name), pkg)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s: %v\n%s", pkg, err, out)
	}
}

// check runs `schemalens check --adapter ent` against the sample project and
// the given database, and returns stdout and the exit code.
func check(t *testing.T, dbPath string, off ...string) (string, int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("builds and runs executables through PATH")
	}
	if _, err := os.Stat(coreCommand); err != nil {
		t.Skip("the core module is not part of this workspace")
	}
	bin := t.TempDir()
	build(t, bin, "schemalens", coreCommand)
	build(t, bin, "schemalens-adapter-ent", "./cmd/schemalens-adapter-ent")
	cfg := filepath.Join(t.TempDir(), "schemalens.yml")
	// ent does not index foreign keys on its own; that is a database rule, not
	// what these tests are about, so it is switched off.
	config := "rules:\n  required-field-nullable-column:\n    enabled: true\n  fk-without-index:\n    enabled: false\n"
	for _, id := range off {
		config += "  " + id + ":\n    enabled: false\n"
	}
	if err := os.WriteFile(cfg, []byte(config), 0o644); err != nil { //nolint:gosec // test config
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(bin, "schemalens"), "check",
		"--dsn", "sqlite://"+dbPath, "--config", cfg,
		"--adapter", "ent", "--adapter-arg", sampleSchema)
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return stdout.String(), 0
	case errors.As(err, &exit):
		if exit.ExitCode() == 2 {
			t.Fatalf("schemalens failed: %s", stderr.String())
		}
		return stdout.String(), exit.ExitCode()
	default:
		t.Fatal(err)
		return "", -1
	}
}

// SQLite ignores VARCHAR(n) and has no native enum, so ent's migration leaves
// those promises unenforced there; the tests that are about something else
// switch the two rules off.
var sqliteBlindRules = []string{"max-length-not-enforced", "enum-not-enforced"}

type checkedFinding struct {
	Rule   string
	Object struct{ Table, Column string }
	Source *struct {
		File string
		Line int
	}
}

func findings(t *testing.T, stdout string) []checkedFinding {
	t.Helper()
	var doc struct{ Findings []checkedFinding }
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	return doc.Findings
}

// onlyFinding asserts exactly one finding, of rule, on table.column, pointing
// at file:line of the ent schema.
func onlyFinding(t *testing.T, stdout, rule, table, col, file string, line int) {
	t.Helper()
	fs := findings(t, stdout)
	if len(fs) != 1 {
		t.Fatalf("findings = %+v, want exactly one", fs)
	}
	f := fs[0]
	if f.Rule != rule || f.Object.Table != table || f.Object.Column != col {
		t.Errorf("unexpected finding %+v", f)
	}
	if f.Source == nil || f.Source.File != file || f.Source.Line != line {
		t.Errorf("finding source = %+v, want %s:%d", f.Source, file, line)
	}
}

func editColumn(table, column string, edit func(*schema.Column)) func([]*schema.Table) {
	return func(tables []*schema.Table) {
		for _, tb := range tables {
			for _, c := range tb.Columns {
				if tb.Name == table && c.Name == column {
					edit(c)
				}
			}
		}
	}
}

func TestDatabaseFromEntMigrationAgreesWithManifest(t *testing.T) {
	// Names, nullability, unique fields, composite uniques and relations all
	// match what ent itself creates.
	stdout, code := check(t, migrate(t, nil), sqliteBlindRules...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0:\n%s", code, stdout)
	}
}

func TestLengthAndEnumPromisesSQLiteCannotEnforceAreReportedAtTheEntField(t *testing.T) {
	stdout, code := check(t, migrate(t, nil))
	if code != 1 {
		t.Fatalf("exit code = %d, want 1:\n%s", code, stdout)
	}
	got := map[string]int{}
	for _, f := range findings(t, stdout) {
		if f.Source == nil {
			t.Fatalf("finding without source: %+v", f)
		}
		got[f.Rule+" "+f.Source.File] = f.Source.Line
	}
	want := map[string]int{
		"max-length-not-enforced testdata/sample/ent/schema/user.go": 19,
		"enum-not-enforced testdata/sample/ent/schema/post.go":       17,
	}
	if len(got) != len(want) {
		t.Fatalf("findings = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: line %d, want %d (all: %v)", k, got[k], v, got)
		}
	}
}

func TestRequiredFieldOnNullableColumnNamesEntFileAndLine(t *testing.T) {
	dbPath := migrate(t, editColumn("users", "name", func(c *schema.Column) { c.Nullable = true }))
	stdout, code := check(t, dbPath, sqliteBlindRules...)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1:\n%s", code, stdout)
	}
	onlyFinding(t, stdout, "required-field-nullable-column", "users", "name", "testdata/sample/ent/schema/user.go", 19)
}

func TestUniqueFieldWithoutUniqueIndexNamesEntField(t *testing.T) {
	dbPath := migrate(t, editColumn("posts", "slug", func(c *schema.Column) { c.Unique = false }))
	stdout, code := check(t, dbPath, sqliteBlindRules...)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1:\n%s", code, stdout)
	}
	onlyFinding(t, stdout, "unique-field-without-unique-index", "posts", "slug", "testdata/sample/ent/schema/post.go", 15)
}
