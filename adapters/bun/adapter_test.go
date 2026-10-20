package main_test

import (
	"database/sql"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/schema"
	_ "modernc.org/sqlite"

	"github.com/mokevnin/schemalens/adapters/bun/testdata/shop"
	"github.com/mokevnin/schemalens/adapters/staticmodel"
	"github.com/mokevnin/schemalens/adapters/staticmodel/adaptertest"
)

var update = flag.Bool("update", false, "rewrite golden files")

var bins *adaptertest.Binaries

func TestMain(m *testing.M) {
	flag.Parse()
	var err error
	bins, err = adaptertest.Build("bun", "github.com/mokevnin/schemalens/adapters/bun")
	if err != nil {
		panic(err)
	}
	code := m.Run()
	bins.Close()
	os.Exit(code)
}

func runIn(t *testing.T, dir, bin string, args ...string) (string, string, int) {
	t.Helper()
	return adaptertest.RunIn(t, os.Environ(), dir, bin, args...)
}

func TestSampleProjectManifestMatchesGolden(t *testing.T) {
	stdout, stderr, code := runIn(t, "testdata/shop", bins.Adapter)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	adaptertest.Golden(t, *update, "testdata/shop.manifest.golden.json", stdout)
}

// The manifest agrees with what bun itself computes at run time for the same
// structs: table names, column names, not-null and primary keys.
func TestManifestAgreesWithBunRuntime(t *testing.T) {
	stdout, stderr, code := runIn(t, "testdata/shop", bins.Adapter)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	var m staticmodel.Manifest
	if err := json.Unmarshal([]byte(stdout), &m); err != nil {
		t.Fatal(err)
	}
	byName := map[string]staticmodel.Table{}
	for _, tb := range m.Tables {
		name := tb.Name
		if tb.Schema != "" {
			name = tb.Schema + "." + name
		}
		byName[name] = tb
	}
	models := map[string]any{
		"users":             (*shop.User)(nil),
		"orders":            (*shop.Order)(nil),
		"billing.companies": (*shop.Company)(nil),
		"user_profiles":     (*shop.UserProfile)(nil),
		"categories":        (*shop.Category)(nil),
	}
	tables := schema.NewTables(sqlitedialect.New())
	for name, model := range models {
		want := tables.Get(reflect.TypeOf(model).Elem())
		if want.Name != name {
			t.Fatalf("bun names %T %q, test expects %q", model, want.Name, name)
		}
		got, ok := byName[name]
		if !ok {
			t.Errorf("manifest lacks table %q", name)
			continue
		}
		var wantCols, gotCols, wantPK []string
		for _, f := range want.Fields {
			wantCols = append(wantCols, f.Name)
			if f.IsPK {
				wantPK = append(wantPK, f.Name)
			}
		}
		for _, c := range got.Columns {
			gotCols = append(gotCols, c.Name)
			if f, ok := want.FieldMap[c.Name]; ok && f.NotNull != c.Required {
				t.Errorf("%s.%s: manifest required=%v, bun not-null=%v", name, c.Name, c.Required, f.NotNull)
			}
		}
		if !slices.Equal(wantCols, gotCols) {
			t.Errorf("%s columns: manifest %v, bun %v", name, gotCols, wantCols)
		}
		if !slices.Equal(wantPK, got.PrimaryKey) {
			t.Errorf("%s primary key: manifest %v, bun %v", name, got.PrimaryKey, wantPK)
		}
	}
}

func TestConfigTableOverrideWinsOverTag(t *testing.T) {
	stdout, _, code := runIn(t, "testdata/shop", bins.Adapter)
	if code != 0 {
		t.Fatal("adapter failed")
	}
	if !strings.Contains(stdout, `"name": "tags"`) || strings.Contains(stdout, "labels_set_in_code") {
		t.Errorf("config table override not applied:\n%s", stdout)
	}
}

func TestBrokenModelFailsWithPosition(t *testing.T) {
	_, stderr, code := runIn(t, "testdata/broken", bins.Adapter)
	if code == 0 {
		t.Fatal("exit 0, want failure")
	}
	for _, want := range []string{"models.go:", "Post", "Author", "author_id"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q: %s", want, stderr)
		}
	}
}

func TestAdapterRunsUnderSchemalensAdapterFlag(t *testing.T) {
	work := t.TempDir()
	dbPath := filepath.Join(work, "db.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY);`)
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bins.Schemalens, "check", "--dsn", "sqlite://"+dbPath, "--adapter", "bun", "--adapter-arg", "./...")
	cmd.Dir = "testdata/shop"
	cmd.Env = bins.PathEnv()
	_ = cmd.Run()
	// Exit code 2 would mean the adapter protocol failed.
	if cmd.ProcessState.ExitCode() == 2 {
		t.Fatal("adapter run failed")
	}
}

func TestManifestFeedsSchemalensCheck(t *testing.T) {
	manifest, stderr, code := runIn(t, "testdata/shop", bins.Adapter)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	work := t.TempDir()
	mpath := filepath.Join(work, "manifest.json")
	if err := os.WriteFile(mpath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(work, "schemalens.yml")
	if err := os.WriteFile(cfg, []byte("rules:\n  required-field-nullable-column:\n    enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(work, "db.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// orders.buyer_id is nullable although the model says "notnull".
	_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL);
CREATE TABLE orders (id INTEGER PRIMARY KEY, buyer_id INTEGER);`)
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runIn(t, work, bins.Schemalens, "check", "--dsn", "sqlite://"+dbPath, "--config", cfg, "--manifest", mpath)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (stderr: %s)\n%s", code, stderr, stdout)
	}
	// The finding points at Order.BuyerID in the sample project.
	if !strings.Contains(stdout, `"file": "models.go"`) || !strings.Contains(stdout, `"line": 43`) {
		t.Errorf("finding lacks the source position of Order.BuyerID:\n%s", stdout)
	}
}
