package main_test

import (
	"bytes"
	"database/sql"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/adapters/staticmodel/adaptertest"
	_ "modernc.org/sqlite"
)

var update = flag.Bool("update", false, "rewrite golden files")

var bins *adaptertest.Binaries

func TestMain(m *testing.M) {
	flag.Parse()
	var err error
	bins, err = adaptertest.Build("gorm", "github.com/mokevnin/schemalens/adapters/gorm")
	if err != nil {
		panic(err)
	}
	code := m.Run()
	bins.Close()
	os.Exit(code)
}

// runIn runs bin in dir and returns stdout, stderr and the exit code.
func runIn(t *testing.T, dir, bin string, args ...string) (string, string, int) {
	t.Helper()
	return adaptertest.RunIn(t, os.Environ(), dir, bin, args...)
}

func golden(t *testing.T, path, got string) {
	t.Helper()
	adaptertest.Golden(t, *update, path, got)
}

func TestSampleProjectManifestMatchesGolden(t *testing.T) {
	stdout, stderr, code := runIn(t, "testdata/shop", bins.Adapter)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	golden(t, "testdata/shop.manifest.golden.json", stdout)
}

func TestTableNameSetInCodeNeedsAnOverride(t *testing.T) {
	_, stderr, code := runIn(t, "testdata/computed", bins.Adapter)
	if code == 0 {
		t.Fatal("exit 0, want failure")
	}
	for _, want := range []string{"models.go:", "Account", "tables", "schemalens-gorm.yml"} {
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
	_, err = db.Exec(`CREATE TABLE app_users (id INTEGER PRIMARY KEY);`)
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bins.Schemalens, "check", "--dsn", "sqlite://"+dbPath, "--adapter", "gorm", "--adapter-arg", "./...")
	cmd.Dir = "testdata/shop"
	cmd.Env = bins.PathEnv()
	var errb bytes.Buffer
	cmd.Stderr = &errb
	_ = cmd.Run()
	// Exit code 2 would mean the adapter protocol failed.
	if cmd.ProcessState.ExitCode() == 2 {
		t.Fatalf("adapter run failed: %s", errb.String())
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
	// users.email is nullable although the model says "not null".
	_, err = db.Exec(`CREATE TABLE app_users (id INTEGER PRIMARY KEY, email TEXT);
CREATE TABLE app_orders (id INTEGER PRIMARY KEY, buyer_id INTEGER NOT NULL);`)
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runIn(t, work, bins.Schemalens, "check", "--dsn", "sqlite://"+dbPath, "--config", cfg, "--manifest", mpath)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (stderr: %s)\n%s", code, stderr, stdout)
	}
	// The finding points at the struct field in the sample project.
	if !strings.Contains(stdout, `"file": "models.go"`) || !strings.Contains(stdout, `"line": 25`) {
		t.Errorf("finding lacks the source position of User.Email:\n%s", stdout)
	}
}
