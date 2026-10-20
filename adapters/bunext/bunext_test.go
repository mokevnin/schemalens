package bunext_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jinzhu/inflection"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/schema"

	"github.com/mokevnin/schemalens/adapters/bunext"
	"github.com/mokevnin/schemalens/manifest"
)

type Company struct {
	ID    int64  `bun:",pk,autoincrement"`
	Title string `bun:",notnull,type:varchar(80)"`
}

type Member struct {
	ID        int64    `bun:",pk,autoincrement"`
	FullName  string   `bun:",notnull,type:varchar(50)"`
	Email     string   `bun:",unique"`
	CompanyID int64    `bun:",unique:company_nick"`
	Nick      string   `bun:",unique:company_nick"`
	Company   *Company `bun:"rel:belongs-to,join:company_id=id"`
	Skipped   string   `bun:"-"`
}

// Legacy has an explicit table name; the tag wins over any inflector.
type Legacy struct {
	bun.BaseModel `bun:"table:legacy_things"`

	ID int64 `bun:",pk"`
}

var schemalensBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "bunext-test")
	if err != nil {
		panic(err)
	}
	schemalensBin = filepath.Join(dir, "schemalens")
	if b, err := exec.Command("go", "build", "-o", schemalensBin, "github.com/mokevnin/schemalens/cmd/schemalens").CombinedOutput(); err != nil {
		panic(string(b))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// prefixNaming sets bun's table-name inflector for the test, as a project
// would at start-up, and restores it afterwards.
func prefixNaming(t *testing.T) {
	t.Helper()
	schema.SetTableNameInflector(func(s string) string { return "tbl_" + inflection.Plural(s) })
	t.Cleanup(func() { schema.SetTableNameInflector(inflection.Plural) })
}

// migrate creates the models' tables with bun's own CreateTable and returns
// the database path and handle.
func migrate(t *testing.T, models ...any) (string, *bun.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db.sqlite")
	sqldb, err := sql.Open(sqliteshim.ShimName, path)
	if err != nil {
		t.Fatal(err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })
	for _, m := range models {
		if _, err := db.NewCreateTable().Model(m).WithForeignKeys().Exec(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return path, db
}

func physicalTables(t *testing.T, db *bun.DB) []string {
	t.Helper()
	var names []string
	if err := db.NewRaw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`).Scan(context.Background(), &names); err != nil {
		t.Fatal(err)
	}
	return names
}

func physicalColumns(t *testing.T, db *bun.DB, table string) []string {
	t.Helper()
	var cols []string
	if err := db.NewRaw(`SELECT name FROM pragma_table_info(?)`, table).Scan(context.Background(), &cols); err != nil {
		t.Fatal(err)
	}
	slices.Sort(cols)
	return cols
}

func TestRunTimeNamingYieldsThePhysicalNamesTheDatabaseHas(t *testing.T) {
	prefixNaming(t)
	_, db := migrate(t, &Company{}, &Member{}, &Legacy{})

	m, err := bunext.Manifest(db.Dialect(), &Company{}, &Member{}, &Legacy{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tb := range m.Tables {
		got = append(got, tb.Name)
		cols := make([]string, 0, len(tb.Columns))
		for _, c := range tb.Columns {
			cols = append(cols, c.Name)
		}
		slices.Sort(cols)
		if want := physicalColumns(t, db, tb.Name); !slices.Equal(cols, want) {
			t.Errorf("columns of %s: manifest %v, database %v", tb.Name, cols, want)
		}
	}
	slices.Sort(got)
	if want := physicalTables(t, db); !slices.Equal(got, want) {
		t.Errorf("tables: manifest %v, database %v", got, want)
	}
	if want := []string{"legacy_things", "tbl_companies", "tbl_members"}; !slices.Equal(got, want) {
		t.Errorf("tables = %v, want %v", got, want)
	}
}

func TestManifestIsAcceptedByCheckUnchanged(t *testing.T) {
	prefixNaming(t)
	dbPath, db := migrate(t, &Company{}, &Member{}, &Legacy{})
	out, errOut, code := check(t, dbPath, manifestOf(t, db, &Company{}, &Member{}, &Legacy{}))
	if code != 0 {
		t.Fatalf("check exit %d, want 0 (a database made by CreateTable agrees with its own model)\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

func TestFindingsLandOnTheInflectedTableTheDatabaseHas(t *testing.T) {
	prefixNaming(t)
	dbPath, db := migrate(t, &Company{}, &Member{})
	// Loosen one column behind bun's back: the model says not null.
	for _, q := range []string{
		`ALTER TABLE tbl_members DROP COLUMN full_name`,
		`ALTER TABLE tbl_members ADD COLUMN full_name TEXT`,
	} {
		if _, err := db.ExecContext(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	out, errOut, code := check(t, dbPath, manifestOf(t, db, &Company{}, &Member{}))
	if code != 1 || !strings.Contains(out, `"table": "tbl_members"`) || !strings.Contains(out, `"column": "full_name"`) {
		t.Fatalf("check exit %d, want 1 on tbl_members.full_name\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

func TestManifestCarriesBunsOwnReadingOfTheTags(t *testing.T) {
	_, db := migrate(t, &Company{}, &Member{})
	m, err := bunext.Manifest(db.Dialect(), &Company{}, &Member{})
	if err != nil {
		t.Fatal(err)
	}
	tb := tableNamed(t, m, "members")
	if !slices.Equal(tb.PrimaryKey, []string{"id"}) {
		t.Errorf("primary key = %v", tb.PrimaryKey)
	}
	col := func(name string) manifest.Column {
		for _, c := range tb.Columns {
			if c.Name == name {
				return c
			}
		}
		t.Fatalf("no column %s in %+v", name, tb.Columns)
		return manifest.Column{}
	}
	if c := col("full_name"); !c.Required || c.MaxLen != 50 {
		t.Errorf("full_name = %+v, want required with max_length 50", c)
	}
	if !col("email").Unique {
		t.Error("email should be unique")
	}
	if !col("id").Required {
		t.Error("the primary key is required")
	}
	if slices.ContainsFunc(tb.Columns, func(c manifest.Column) bool { return c.Name == "skipped" }) {
		t.Error("a field tagged - is not a column")
	}
	if len(tb.Uniques) != 1 || !slices.Equal(tb.Uniques[0].Columns, []string{"company_id", "nick"}) {
		t.Errorf("uniques = %+v, want the composite (company_id, nick)", tb.Uniques)
	}
	if len(tb.Relations) != 1 || !slices.Equal(tb.Relations[0].Columns, []string{"company_id"}) || tb.Relations[0].References.Table != "companies" {
		t.Errorf("relations = %+v, want company_id -> companies", tb.Relations)
	}
	if tb.Source != nil || tb.Columns[0].Source != nil {
		t.Error("extended mode has no source positions")
	}
}

func TestAModelBunCannotUseIsAnError(t *testing.T) {
	_, err := bunext.Manifest(sqlitedialect.New(), 42)
	if err == nil {
		t.Fatal("want an error for a non-struct model")
	}
}

func tableNamed(t *testing.T, m *manifest.Manifest, name string) manifest.Table {
	t.Helper()
	for _, tb := range m.Tables {
		if tb.Name == name {
			return tb
		}
	}
	t.Fatalf("no table %s in manifest", name)
	return manifest.Table{}
}

func manifestOf(t *testing.T, db *bun.DB, models ...any) string {
	t.Helper()
	m, err := bunext.Manifest(db.Dialect(), models...)
	if err != nil {
		t.Fatal(err)
	}
	var doc bytes.Buffer
	if err := m.Write(&doc); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, doc.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func check(t *testing.T, dbPath, manifestPath string) (string, string, int) {
	t.Helper()
	dir := t.TempDir()
	// SQLite ignores VARCHAR(n), so the length rule would always fire on it.
	cfg := filepath.Join(dir, "schemalens.yml")
	rules := "rules:\n  max-length-not-enforced:\n    enabled: false\n  fk-without-index:\n    enabled: false\n  required-field-nullable-column:\n    enabled: true\n"
	if err := os.WriteFile(cfg, []byte(rules), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(schemalensBin, "check", "--config", cfg, "--dsn", "sqlite://"+dbPath, "--manifest", manifestPath)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return out.String(), errb.String(), 0
	case errors.As(err, &ee):
		return out.String(), errb.String(), ee.ExitCode()
	}
	t.Fatal(err)
	return "", "", -1
}
