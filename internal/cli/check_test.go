package cli_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/mokevnin/schemalens/internal/cli"
)

var update = flag.Bool("update", false, "rewrite golden files")

// Golden cases live in testdata/check/<case>/:
//
//	schema.sql     DDL applied to a fresh SQLite database
//	args           optional extra arguments, whitespace separated
//	expected.json  stdout of `schemalens check`
//	exit_code      expected exit code
//
// Every future rule adds a directory here; no Go changes are needed.
// Run `go test ./internal/cli -update` to rewrite expected files, then review the diff.
func TestCheckGolden(t *testing.T) {
	cases, err := filepath.Glob("../../testdata/check/*/schema.sql")
	if err != nil || len(cases) == 0 {
		t.Fatalf("no golden cases found: %v", err)
	}
	for _, schemaFile := range cases {
		dir := filepath.Dir(schemaFile)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			ddl, err := os.ReadFile(schemaFile)
			if err != nil {
				t.Fatal(err)
			}
			dbPath := filepath.Join(t.TempDir(), "db.sqlite")
			applyDDL(t, dbPath, string(ddl))

			args := []string{"check", "--dsn", "sqlite://" + dbPath}
			if extra, err := os.ReadFile(filepath.Join(dir, "args")); err == nil {
				args = append(args, strings.Fields(string(extra))...)
			}
			stdout, _, code := run(args...)

			if *update {
				writeFile(t, filepath.Join(dir, "expected.json"), stdout)
				writeFile(t, filepath.Join(dir, "exit_code"), strconv.Itoa(code)+"\n")
				return
			}
			want, err := os.ReadFile(filepath.Join(dir, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			if stdout != string(want) {
				t.Errorf("stdout mismatch\n--- got\n%s\n--- want\n%s", stdout, want)
			}
			wantCode, err := os.ReadFile(filepath.Join(dir, "exit_code"))
			if err != nil {
				t.Fatal(err)
			}
			if strconv.Itoa(code) != strings.TrimSpace(string(wantCode)) {
				t.Errorf("exit code = %d, want %s", code, wantCode)
			}
			if !json.Valid([]byte(stdout)) {
				t.Errorf("stdout is not valid JSON: %s", stdout)
			}
			again, _, _ := run(args...)
			if again != stdout {
				t.Errorf("two runs differ\n--- first\n%s\n--- second\n%s", stdout, again)
			}
		})
	}
}

func TestCheckConfigurationErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope", "db.sqlite")
	tests := map[string][]string{
		"unknown flag":        {"check", "--bogus"},
		"missing dsn":         {"check"},
		"unsupported scheme":  {"check", "--dsn", "oracle://x"},
		"unreadable database": {"check", "--dsn", "sqlite://" + missing},
		"unknown command":     {"frobnicate"},
		"no command":          {},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := run(args...)
			if code != cli.ExitUsage {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, cli.ExitUsage, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout should be empty on configuration errors, got %q", stdout)
			}
			if stderr == "" {
				t.Error("stderr should explain the error")
			}
		})
	}
}

func run(args ...string) (stdout, stderr string, code int) {
	var out, errb bytes.Buffer
	code = cli.Run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func applyDDL(t *testing.T, path, ddl string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("applying DDL: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
