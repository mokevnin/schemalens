package cli_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mokevnin/schemalens/internal/cli"
	"github.com/mokevnin/schemalens/internal/dialect"
)

var update = flag.Bool("update", false, "rewrite golden files")

// Golden cases live in testdata/check/<case>/ and run against every dialect:
//
//	schema.sql             DDL applied to a fresh database, for every dialect
//	schema.<dialect>.sql   DDL for one dialect, instead of schema.sql
//	args                   optional extra arguments, whitespace separated; {case} is this directory
//	schemalens.yml         optional configuration, passed as --config {case}/schemalens.yml in args
//	expected.json          stdout of `schemalens check`
//	expected.<dialect>.json  stdout for one dialect, instead of expected.json
//	exit_code              expected exit code (exit_code.<dialect> likewise)
//	stderr.<dialect>       optional text stderr must contain for that dialect
//	stderr_contains        optional; every non-empty line must appear in stderr
//
// The schema name differs per dialect (main, public, schemalens), so cases
// with findings carry per-dialect expected files. Every future rule adds a
// directory here; no Go changes are needed. PostgreSQL and MySQL run in
// containers and skip when Docker is not available.
// Run `go test ./internal/cli -update` to rewrite expected files, then review the diff.
func TestCheckGolden(t *testing.T) {
	cases, err := filepath.Glob("../../testdata/check/*")
	if err != nil || len(cases) == 0 {
		t.Fatalf("no golden cases found: %v", err)
	}
	for _, b := range backends {
		t.Run(string(b.dialect), func(t *testing.T) {
			for _, dir := range cases {
				t.Run(filepath.Base(dir), func(t *testing.T) {
					runGolden(t, b, dir)
				})
			}
		})
	}
}

// pick returns the dialect-specific file name if it exists, else the shared one.
func pick(dir string, d dialect.Dialect, shared, specific string) (string, bool) {
	for _, name := range []string{specific, shared} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return filepath.Join(dir, shared), false
}

func runGolden(t *testing.T, b backend, dir string) {
	d := b.dialect
	schemaFile, ok := pick(dir, d, "schema.sql", "schema."+string(d)+".sql")
	if !ok {
		t.Skipf("case has no schema for %s", d)
	}
	ddl, err := os.ReadFile(schemaFile)
	if err != nil {
		t.Fatal(err)
	}
	dsn := b.setup(t, string(ddl))

	args := []string{"check", "--dsn", dsn}
	if extra, err := os.ReadFile(filepath.Join(dir, "args")); err == nil {
		// {case} expands to the case directory so args can point at a config file.
		args = append(args, strings.Fields(strings.ReplaceAll(string(extra), "{case}", dir))...)
	}
	stdout, stderr, code := run(args...)

	expectedFile, expectedSpecific := pick(dir, d, "expected.json", "expected."+string(d)+".json")
	codeFile, codeSpecific := pick(dir, d, "exit_code", "exit_code."+string(d))
	if *update {
		// Shared expectations are rewritten from SQLite only; other dialects
		// rewrite only the files that are specific to them.
		if d != dialect.SQLite && (!expectedSpecific || !codeSpecific) {
			return
		}
		writeFile(t, expectedFile, stdout)
		writeFile(t, codeFile, strconv.Itoa(code)+"\n")
		return
	}
	want, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stdout != string(want) {
		t.Errorf("stdout mismatch\n--- got\n%s\n--- want\n%s", stdout, want)
	}
	wantCode, err := os.ReadFile(codeFile)
	if err != nil {
		t.Fatal(err)
	}
	if strconv.Itoa(code) != strings.TrimSpace(string(wantCode)) {
		t.Errorf("exit code = %d, want %s", code, wantCode)
	}
	if code != cli.ExitUsage && !json.Valid([]byte(stdout)) {
		t.Errorf("stdout is not valid JSON: %s", stdout)
	}
	if want, err := os.ReadFile(filepath.Join(dir, "stderr_contains")); err == nil {
		for _, line := range strings.Split(string(want), "\n") {
			if line != "" && !strings.Contains(stderr, line) {
				t.Errorf("stderr does not contain %q\n--- stderr\n%s", line, stderr)
			}
		}
	}
	if wantErr, err := os.ReadFile(filepath.Join(dir, "stderr."+string(d))); err == nil {
		if !strings.Contains(stderr, strings.TrimSpace(string(wantErr))) {
			t.Errorf("stderr = %q, want it to contain %q", stderr, wantErr)
		}
	}
	again, _, _ := run(args...)
	if again != stdout {
		t.Errorf("two runs differ\n--- first\n%s\n--- second\n%s", stdout, again)
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
