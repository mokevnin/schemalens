package cli_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/internal/cli"
)

// installAdapter writes an executable shell script named
// schemalens-adapter-<name> into a fresh directory and prepends it to PATH.
func installAdapter(t *testing.T, name, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script adapters need a POSIX shell")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "schemalens-adapter-"+name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

const adapterManifest = `{"version":1,"tables":[{"name":"users","columns":[{"name":"email","required":true}]}]}`

const requiredFieldConfig = "rules:\n  required-field-nullable-column:\n    enabled: true\n"

func adapterDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "db.sqlite")
	applyDDL(t, dbPath, "CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT);")
	return "sqlite://" + dbPath
}

func adapterConfig(t *testing.T) string {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "schemalens.yml")
	writeFile(t, cfg, requiredFieldConfig)
	return cfg
}

func TestAdapterManifestFeedsTheModelComparison(t *testing.T) {
	installAdapter(t, "test", "cat <<'JSON'\n"+adapterManifest+"\nJSON\n")
	stdout, stderr, code := run("check", "--dsn", adapterDB(t), "--config", adapterConfig(t), "--adapter", "test")
	if code != cli.ExitFindings {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "required-field-nullable-column") || !strings.Contains(stdout, "email") {
		t.Errorf("stdout lacks the model finding:\n%s", stdout)
	}
}

func TestMissingAdapterNamesExecutableAndSearchPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	_, stderr, code := run("check", "--dsn", adapterDB(t), "--adapter", "nope")
	if code != cli.ExitUsage {
		t.Fatalf("exit code = %d, want 2", code)
	}
	for _, want := range []string{"schemalens-adapter-nope", "not found", dir} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q: %s", want, stderr)
		}
	}
}

func TestFailingAdapterSurfacesItsStderr(t *testing.T) {
	installAdapter(t, "test", "echo 'cannot load ent schema: boom' >&2\nexit 3\n")
	_, stderr, code := run("check", "--dsn", adapterDB(t), "--adapter", "test")
	if code != cli.ExitUsage {
		t.Fatalf("exit code = %d, want 2", code)
	}
	for _, want := range []string{"schemalens-adapter-test", "exit status 3", "cannot load ent schema: boom"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q: %s", want, stderr)
		}
	}
}

func TestAdapterOutputIsValidatedLikeAManifestFile(t *testing.T) {
	installAdapter(t, "test", `echo '{"version":1,"tables":[{"columns":[]}]}'`+"\n")
	_, stderr, code := run("check", "--dsn", adapterDB(t), "--adapter", "test")
	if code != cli.ExitUsage {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "schemalens-adapter-test") || !strings.Contains(stderr, "name") {
		t.Errorf("stderr should name the adapter and the violation: %s", stderr)
	}

	installAdapter(t, "old", `echo '{"version":99,"tables":[]}'`+"\n")
	_, stderr, code = run("check", "--dsn", adapterDB(t), "--adapter", "old")
	if code != cli.ExitUsage || !strings.Contains(stderr, "version 99 is not supported") {
		t.Errorf("unsupported version: exit %d, stderr %s", code, stderr)
	}
}

func TestAdapterReceivesPassThroughArguments(t *testing.T) {
	installAdapter(t, "test",
		`[ "$1" = "--flag" ] || exit 9`+"\n"+`printf '{"version":1,"tables":[{"name":"%s","columns":[]}]}' "$2"`+"\n")
	_, stderr, code := run("check", "--dsn", adapterDB(t), "--config", adapterConfig(t),
		"--adapter", "test", "--adapter-arg", "--flag", "--adapter-arg", "users")
	if code != cli.ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, stderr)
	}
}

func TestAdapterAndManifestAreMutuallyExclusive(t *testing.T) {
	_, stderr, code := run("check", "--dsn", adapterDB(t), "--adapter", "test", "--manifest", "m.json")
	if code != cli.ExitUsage || !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("exit %d, stderr %s", code, stderr)
	}
}
