package gormext_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/mokevnin/schemalens/adapters/gormext"
	"github.com/mokevnin/schemalens/manifest"
)

type Company struct {
	ID    uint
	Title string `gorm:"size:80;not null"`
}

type Member struct {
	ID        uint
	FullName  string `gorm:"not null;size:50"`
	Email     string `gorm:"uniqueIndex"`
	CompanyID uint   `gorm:"uniqueIndex:idx_company_nick"`
	Nick      string `gorm:"uniqueIndex:idx_company_nick"`
	Company   Company
	Skipped   string `gorm:"-"`
}

var schemalensBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gormext-test")
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

// migrate creates the models' tables with gorm's own AutoMigrate under the
// naming strategy and returns the database file and the DB handle.
func migrate(t *testing.T, namer schema.NamingStrategy, models ...any) (string, *gorm.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "db.sqlite")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{NamingStrategy: namer})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	return path, db
}

func physicalTables(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	if err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`).Scan(&names).Error; err != nil {
		t.Fatal(err)
	}
	return names
}

func physicalColumns(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	var cols []string
	if err := db.Raw(`SELECT name FROM pragma_table_info(?)`, table).Scan(&cols).Error; err != nil {
		t.Fatal(err)
	}
	slices.Sort(cols)
	return cols
}

func TestRunTimeNamingStrategyYieldsThePhysicalNamesTheDatabaseHas(t *testing.T) {
	namer := schema.NamingStrategy{TablePrefix: "app_", SingularTable: true}
	_, db := migrate(t, namer, &Company{}, &Member{})

	m, err := gormext.Manifest(namer, &Company{}, &Member{})
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
	if want := []string{"app_company", "app_member"}; !slices.Equal(got, want) {
		t.Errorf("tables = %v, want %v (the prefix and singular naming applied)", got, want)
	}
}

func TestManifestIsAcceptedByCheckUnchanged(t *testing.T) {
	namer := schema.NamingStrategy{TablePrefix: "app_", SingularTable: true}
	dbPath, _ := migrate(t, namer, &Company{}, &Member{})
	m, err := gormext.Manifest(namer, &Company{}, &Member{})
	if err != nil {
		t.Fatal(err)
	}
	var doc bytes.Buffer
	if err := m.Write(&doc); err != nil {
		t.Fatal(err)
	}
	mpath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(mpath, doc.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCheck(t, "--dsn", "sqlite://"+dbPath, "--manifest", mpath)
	if code != 0 {
		t.Fatalf("check exit %d, want 0 (a database made by AutoMigrate agrees with its own model)\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

func TestFindingsLandOnThePrefixedTableTheDatabaseHas(t *testing.T) {
	namer := schema.NamingStrategy{TablePrefix: "app_", SingularTable: true}
	dbPath, db := migrate(t, namer, &Company{}, &Member{})
	// Loosen one column behind gorm's back: the model says not null.
	if err := db.Exec(`ALTER TABLE app_member DROP COLUMN full_name`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`ALTER TABLE app_member ADD COLUMN full_name TEXT`).Error; err != nil {
		t.Fatal(err)
	}
	m, err := gormext.Manifest(namer, &Company{}, &Member{})
	if err != nil {
		t.Fatal(err)
	}
	var doc bytes.Buffer
	if err := m.Write(&doc); err != nil {
		t.Fatal(err)
	}
	mpath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(mpath, doc.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCheck(t, "--dsn", "sqlite://"+dbPath, "--manifest", mpath)
	if code != 1 || !strings.Contains(out, `"table": "app_member"`) || !strings.Contains(out, `"column": "full_name"`) {
		t.Fatalf("check exit %d, want 1 on app_member.full_name\nstdout: %s\nstderr: %s", code, out, errOut)
	}
}

func TestManifestCarriesGormsOwnReadingOfTheTags(t *testing.T) {
	m, err := gormext.Manifest(schema.NamingStrategy{}, &Company{}, &Member{})
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

func TestAModelGormCannotParseIsAnError(t *testing.T) {
	_, err := gormext.Manifest(schema.NamingStrategy{}, 42)
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

func runCheck(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	dir := t.TempDir()
	// SQLite ignores VARCHAR(n), so the length rule would always fire on it.
	cfg := filepath.Join(dir, "schemalens.yml")
	rules := "rules:\n  max-length-not-enforced:\n    enabled: false\n  fk-without-index:\n    enabled: false\n  required-field-nullable-column:\n    enabled: true\n"
	if err := os.WriteFile(cfg, []byte(rules), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(schemalensBin, append([]string{"check", "--config", cfg}, args...)...)
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
