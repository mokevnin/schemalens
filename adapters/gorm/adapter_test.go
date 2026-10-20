package main_test

import (
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

var update = flag.Bool("update", false, "rewrite golden files")

var adapterBin, schemalensBin string

func TestMain(m *testing.M) {
	flag.Parse()
	dir, err := os.MkdirTemp("", "gorm-adapter-test")
	if err != nil {
		panic(err)
	}
	adapterBin = filepath.Join(dir, "schemalens-adapter-gorm")
	schemalensBin = filepath.Join(dir, "schemalens")
	for out, pkg := range map[string]string{
		adapterBin:    "github.com/mokevnin/schemalens/adapters/gorm",
		schemalensBin: "github.com/mokevnin/schemalens/cmd/schemalens",
	} {
		if b, err := exec.Command("go", "build", "-o", out, pkg).CombinedOutput(); err != nil {
			panic(string(b))
		}
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// runIn runs bin in dir and returns stdout, stderr and the exit code.
func runIn(t *testing.T, dir, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), errb.String(), code
}

func golden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil { //nolint:gosec // golden file
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("%s differs from adapter output (rerun with -update to review):\n%s", path, got)
	}
}

func TestSampleProjectManifestMatchesGolden(t *testing.T) {
	stdout, stderr, code := runIn(t, "testdata/shop", adapterBin)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	golden(t, "testdata/shop.manifest.golden.json", stdout)
}

func TestTableNameSetInCodeNeedsAnOverride(t *testing.T) {
	_, stderr, code := runIn(t, "testdata/computed", adapterBin)
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
	cmd := exec.Command(schemalensBin, "check", "--dsn", "sqlite://"+dbPath, "--adapter", "gorm", "--adapter-arg", "./...")
	cmd.Dir = "testdata/shop"
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(adapterBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	var errb bytes.Buffer
	cmd.Stderr = &errb
	_ = cmd.Run()
	// Exit code 2 would mean the adapter protocol failed.
	if cmd.ProcessState.ExitCode() == 2 {
		t.Fatalf("adapter run failed: %s", errb.String())
	}
}

func TestManifestFeedsSchemalensCheck(t *testing.T) {
	manifest, stderr, code := runIn(t, "testdata/shop", adapterBin)
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
	stdout, stderr, code := runIn(t, work, schemalensBin, "check", "--dsn", "sqlite://"+dbPath, "--config", cfg, "--manifest", mpath)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (stderr: %s)\n%s", code, stderr, stdout)
	}
	// The finding points at the struct field in the sample project.
	if !strings.Contains(stdout, `"file": "models.go"`) || !strings.Contains(stdout, `"line": 25`) {
		t.Errorf("finding lacks the source position of User.Email:\n%s", stdout)
	}
}
